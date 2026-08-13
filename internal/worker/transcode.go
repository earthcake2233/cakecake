package worker

import (
	"cakecake/internal/model/video"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/config"
	"cakecake/internal/ffmpeg"
	"cakecake/internal/logger"
	"cakecake/internal/queue"
	"cakecake/internal/search"
	vsvc "cakecake/internal/service/video"
	"cakecake/internal/storage"
)

// TranscodeJob is the JSON payload on the transcode queue.
type TranscodeJob = queue.TranscodeJob

// ffmpegOps isolates external ffmpeg calls so the pipeline is unit-testable.
type ffmpegOps interface {
	TranscodeToH264MP4(ctx context.Context, rawPath, outMP4 string) (stderr string, err error)
	ScreenshotJPEG(ctx context.Context, inPath, outPath string, second float64) (stderr string, err error)
	IsPermanentTranscodeFailure(stderr string) bool
}

type realFFmpegOps struct{}

// TranscodeToH264MP4 runs the real ffmpeg transcode.
func (realFFmpegOps) TranscodeToH264MP4(ctx context.Context, rawPath, outMP4 string) (string, error) {
	return ffmpeg.TranscodeToH264MP4(ctx, rawPath, outMP4)
}

// ScreenshotJPEG captures a frame from a video via ffmpeg.
func (realFFmpegOps) ScreenshotJPEG(ctx context.Context, inPath, outPath string, second float64) (string, error) {
	return ffmpeg.ScreenshotJPEG(ctx, inPath, outPath, second)
}

// IsPermanentTranscodeFailure reports whether ffmpeg stderr indicates a permanent failure.
func (realFFmpegOps) IsPermanentTranscodeFailure(stderr string) bool {
	return ffmpeg.IsPermanentTranscodeFailure(stderr)
}

// transcodePublisher is the minimal channel surface needed to republish a job.
type transcodePublisher interface {
	PublishConfirmed(ctx context.Context, exchange, key string, mandatory bool, msg amqp.Publishing) error
}

// objectStore is the minimal object-storage surface the pipeline needs.
type objectStore interface {
	UploadFile(objectKey, localPath string) error
	DownloadFile(objectKey, localPath string) error
	DeleteObject(objectKey string) error
}

// maxTranscodeRetries is the number of retryable attempts before a job is
// dead-lettered. Retry backoff is owned by RabbitMQ TTL queues.
const maxTranscodeRetries = 3

// defaultTranscodeTimeout bounds one ffmpeg run when TRANSCODE_TIMEOUT is
// unset or the config is nil (tests).
const defaultTranscodeTimeout = 10 * time.Minute

func transcodeTimeout(cfg *config.C) time.Duration {
	if cfg != nil && cfg.TranscodeTimeout > 0 {
		return cfg.TranscodeTimeout
	}
	return defaultTranscodeTimeout
}

// transcodeSubscriber combines subscription and confirmed publishing so the
// reconnect loop is unit-testable without a real broker.
type transcodeSubscriber interface {
	transcodePublisher
	NewTranscodeConsumer(consumerTag string) (queue.TranscodeConsumer, <-chan amqp.Delivery, error)
}

// transcodeReconnectDelay is the backoff after a subscribe error or channel
// loss; variable for tests.
var transcodeReconnectDelay = 3 * time.Second

// StartTranscodeConsumer runs a blocking AMQP consumer loop. Unlike the old
// fail-fast design, a closed channel or broker blip no longer stops the
// worker silently: the loop reconnects with backoff, so queued jobs do not
// pile up unattended. With TRANSCODE_CONCURRENCY > 1 each consumer has its
// own channel (QoS=1), so one process can transcode several videos at once.
func StartTranscodeConsumer(ctx context.Context, cfg *config.C, db *gorm.DB, mq *queue.Client, ossClient *storage.OSS, esc *search.Client) {
	n := transcodeConcurrency(cfg)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			consumeTranscodeLoop(ctx, cfg, db, mq, ossClient, esc, logger.L.With(zap.Int("worker_id", i)))
		}(i)
	}
	wg.Wait()
}

func transcodeConcurrency(cfg *config.C) int {
	if cfg != nil && cfg.TranscodeConcurrency > 0 {
		return cfg.TranscodeConcurrency
	}
	return 1
}

func consumeTranscodeLoop(ctx context.Context, cfg *config.C, db *gorm.DB, sub transcodeSubscriber, ossClient *storage.OSS, esc *search.Client, lg *zap.Logger) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := consumeTranscodeOnce(ctx, cfg, db, sub, ossClient, esc, lg); err != nil {
			lg.Warn("transcode consumer: subscribe", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(transcodeReconnectDelay):
		}
	}
}

// consumeTranscodeOnce drains one main-queue channel. It returns when the
// channel closes or the context is cancelled; the caller reconnects.
func consumeTranscodeOnce(ctx context.Context, cfg *config.C, db *gorm.DB, sub transcodeSubscriber, ossClient *storage.OSS, esc *search.Client, lg *zap.Logger) error {
	ch, msgs, err := sub.NewTranscodeConsumer("transcode-worker")
	if err != nil {
		return err
	}
	defer ch.Close()
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return errors.New("transcode consumer channel closed")
			}
			handleDelivery(ctx, cfg, db, sub, ossClient, esc, d)
		}
	}
}

func handleDelivery(ctx context.Context, cfg *config.C, db *gorm.DB, pubCh transcodePublisher, ossClient *storage.OSS, esc *search.Client, d amqp.Delivery) {
	var store objectStore
	if ossClient != nil {
		store = ossClient
	}
	handleDeliveryWith(ctx, cfg, db, pubCh, store, esc, d, realFFmpegOps{}, logger.L)
}

func handleDeliveryWith(ctx context.Context, cfg *config.C, db *gorm.DB, pubCh transcodePublisher, ossClient objectStore, esc *search.Client, d amqp.Delivery, ff ffmpegOps, lg *zap.Logger) {
	start := time.Now()
	var job TranscodeJob
	if err := json.Unmarshal(d.Body, &job); err != nil {
		lg.Error("transcode bad json", zap.Error(err))
		ackDelivery(lg, d)
		return
	}
	lg.Info("transcode job received", zap.Uint64("video_id", job.VideoID), zap.String("raw", job.RawPath))
	// At-least-once redelivery guard: a crash between the DB update and the
	// Ack can redeliver an already-processed job. If the video already reached
	// a terminal transcode state, skip and Ack instead of re-transcoding.
	if db != nil {
		var v video.Video
		if err := db.WithContext(ctx).First(&v, job.VideoID).Error; err != nil {
			lg.Warn("transcode video missing, skipping redelivery",
				zap.Uint64("video_id", job.VideoID), zap.Error(err))
			cleanupPaths(job.RawPath, job.CoverPath)
			ackDelivery(lg, d)
			return
		}
		switch v.Status {
		case video.StatusPublished, video.StatusPendingReview, video.StatusFailed, video.StatusRejected:
			lg.Info("transcode job already terminal, skipping redelivery",
				zap.Uint64("video_id", job.VideoID),
				zap.String("status", v.Status),
				zap.String("video_url", v.VideoURL),
			)
			cleanupPaths(job.RawPath, job.CoverPath)
			ackDelivery(lg, d)
			return
		}
	}
	if ossClient == nil {
		lg.Error("oss not configured, failing job", zap.Uint64("video_id", job.VideoID))
		failVideo(db, job.VideoID, "OSS 未配置")
		cleanupPaths(job.RawPath, job.CoverPath, "", "", "")
		ackDelivery(lg, d)
		return
	}

	// The job may reference OSS object keys (RawKey/CoverKey) instead of a
	// local path: download the durable source to this worker before
	// transcoding, so any host/container can process it. Legacy jobs with
	// only local paths keep working unchanged.
	var downloadedSources []string
	rawLocal := job.RawPath
	if job.RawKey != "" {
		rawLocal = filepath.Join(cfg.TempUploadDir, fmt.Sprintf("%d_source%s", job.VideoID, filepath.Ext(job.RawKey)))
		if err := ossClient.DownloadFile(job.RawKey, rawLocal); err != nil {
			lg.Warn("download raw source", zap.Uint64("video_id", job.VideoID), zap.String("key", job.RawKey), zap.Error(err))
			if requeueOrFail(ctx, db, pubCh, lg, job, fmt.Sprintf("下载原始视频失败: %v", err), rawLocal) {
				cleanupPaths(rawLocal)
			}
			ackDelivery(lg, d)
			return
		}
		downloadedSources = append(downloadedSources, rawLocal)
	}
	coverLocal := job.CoverPath
	if job.CoverKey != "" {
		coverLocal = filepath.Join(cfg.TempUploadDir, fmt.Sprintf("%d_cover%s", job.VideoID, filepath.Ext(job.CoverKey)))
		if err := ossClient.DownloadFile(job.CoverKey, coverLocal); err != nil {
			lg.Warn("download cover source", zap.Uint64("video_id", job.VideoID), zap.String("key", job.CoverKey), zap.Error(err))
			cleanup := append(append([]string{}, downloadedSources...), coverLocal)
			if requeueOrFail(ctx, db, pubCh, lg, job, fmt.Sprintf("下载封面失败: %v", err), cleanup...) {
				cleanupPaths(cleanup...)
			}
			ackDelivery(lg, d)
			return
		}
		downloadedSources = append(downloadedSources, coverLocal)
	}

	outMP4 := filepath.Join(cfg.TempUploadDir, fmt.Sprintf("%d_out.mp4", job.VideoID))
	coverOut := filepath.Join(cfg.TempUploadDir, fmt.Sprintf("%d_cover.jpg", job.VideoID))
	_ = os.Remove(outMP4)
	_ = os.Remove(coverOut)

	lg.Info("transcode ffmpeg start", zap.Uint64("video_id", job.VideoID))
	ffCtx, cancelFF := context.WithTimeout(ctx, transcodeTimeout(cfg))
	defer cancelFF()
	stderr, err := ff.TranscodeToH264MP4(ffCtx, rawLocal, outMP4)
	if err != nil {
		lg.Warn("ffmpeg transcode failed", zap.Uint64("video_id", job.VideoID), zap.Error(err), zap.String("stderr", stderr))
		if ff.IsPermanentTranscodeFailure(stderr) {
			incrTranscodePermanentFailure()
			failVideo(db, job.VideoID, strings.TrimSpace(stderr))
			cleanupPaths(rawLocal, coverLocal, outMP4, coverOut, "")
			deleteSourceObjects(ossClient, lg, job.RawKey, job.CoverKey)
			ackDelivery(lg, d)
			return
		}
		extras := append(regenerableExtras(outMP4, coverOut, "", coverLocal), downloadedSources...)
		if requeueOrFail(ctx, db, pubCh, lg, job, stderr, extras...) {
			cleanupPaths(extras...)
		}
		ackDelivery(lg, d)
		return
	}
	lg.Info("transcode ffmpeg done", zap.Uint64("video_id", job.VideoID))

	var finalCoverPath string
	var coverExt string
	if coverLocal != "" {
		finalCoverPath = coverLocal
		coverExt = strings.TrimPrefix(strings.ToLower(filepath.Ext(coverLocal)), ".")
		if coverExt == "jpeg" {
			coverExt = "jpg"
		}
	} else {
		// Default cover: captures a frame from the transcoded H.264 MP4 (more reliable than capturing from the source container).
		se, err := ff.ScreenshotJPEG(ffCtx, outMP4, coverOut, 1)
		if err != nil {
			lg.Warn("ffmpeg screenshot failed", zap.Error(err), zap.String("stderr", se))
			if ff.IsPermanentTranscodeFailure(se) {
				incrTranscodePermanentFailure()
				failVideo(db, job.VideoID, strings.TrimSpace(se))
				cleanupPaths(rawLocal, coverLocal, outMP4, coverOut, "")
				deleteSourceObjects(ossClient, lg, job.RawKey, job.CoverKey)
				ackDelivery(lg, d)
				return
			}
			extras := append(regenerableExtras(outMP4, coverOut, coverOut, coverLocal), downloadedSources...)
			if requeueOrFail(ctx, db, pubCh, lg, job, se, extras...) {
				cleanupPaths(extras...)
			}
			ackDelivery(lg, d)
			return
		}
		finalCoverPath = coverOut
		coverExt = "jpg"
	}

	videoKey := fmt.Sprintf("videos/%d.mp4", job.VideoID)
	coverKey := fmt.Sprintf("covers/%d.%s", job.VideoID, coverExt)

	lg.Info("transcode oss upload start", zap.Uint64("video_id", job.VideoID), zap.String("video_key", videoKey), zap.String("cover_key", coverKey))
	if err := ossClient.UploadFile(videoKey, outMP4); err != nil {
		lg.Error("oss upload video", zap.Error(err))
		extras := append(regenerableExtras(outMP4, coverOut, finalCoverPath, coverLocal), downloadedSources...)
		if requeueOrFail(ctx, db, pubCh, lg, job, err.Error(), extras...) {
			// Retries still depend on RawPath / user cover: only delete
			// regenerable intermediate artifacts.
			cleanupPaths(extras...)
		}
		ackDelivery(lg, d)
		return
	}
	if err := ossClient.UploadFile(coverKey, finalCoverPath); err != nil {
		lg.Error("oss upload cover", zap.Error(err))
		extras := append(regenerableExtras(outMP4, coverOut, finalCoverPath, coverLocal), downloadedSources...)
		if requeueOrFail(ctx, db, pubCh, lg, job, err.Error(), extras...) {
			cleanupPaths(extras...)
		}
		ackDelivery(lg, d)
		return
	}

	videoURL := cfg.OSSObjectURL(videoKey)
	coverURL := cfg.OSSObjectURL(coverKey)

	updates := map[string]interface{}{
		"video_url": videoURL,
		"cover_url": coverURL,
	}
	if cfg.VideoReviewRequired {
		updates["status"] = video.StatusPendingReview
	}
	// The OSS objects are already durable; a DB failure must not be swallowed.
	// Requeue with retry budget instead of Acking a video that would stay
	// stuck in processing forever. Raw media is kept for compensation.
	if err := db.Model(&video.Video{}).Where("id = ?", job.VideoID).Updates(updates).Error; err != nil {
		lg.Error("db update after transcode", zap.Error(err), zap.Uint64("video_id", job.VideoID))
		extras := append(regenerableExtras(outMP4, coverOut, finalCoverPath, coverLocal), downloadedSources...)
		if requeueOrFail(ctx, db, pubCh, lg, job, fmt.Sprintf("记录转码产物失败: %v", err), extras...) {
			cleanupPaths(extras...)
		}
		ackDelivery(lg, d)
		return
	}
	if !cfg.VideoReviewRequired {
		if err := vsvc.PublishVideo(ctx, db, esc, lg, job.VideoID, nil); err != nil {
			lg.Error("publish video after transcode", zap.Error(err), zap.Uint64("video_id", job.VideoID))
			extras := append(regenerableExtras(outMP4, coverOut, finalCoverPath, coverLocal), downloadedSources...)
			if requeueOrFail(ctx, db, pubCh, lg, job, fmt.Sprintf("发布视频失败: %v", err), extras...) {
				cleanupPaths(extras...)
			}
			ackDelivery(lg, d)
			return
		}
	}
	cleanupPaths(job.RawPath, job.CoverPath, outMP4, coverOut, "")
	cleanupPaths(downloadedSources...)
	deleteSourceObjects(ossClient, lg, job.RawKey, job.CoverKey)
	incrTranscodeSuccess(time.Since(start))
	lg.Info("transcode completed", zap.Uint64("video_id", job.VideoID))
	ackDelivery(lg, d)
}

// ackDelivery acknowledges a consumed message and logs (rather than swallows) Ack failures.
func ackDelivery(lg *zap.Logger, d amqp.Delivery) {
	if err := d.Ack(false); err != nil {
		lg.Error("transcode ack failed", zap.Error(err), zap.Uint64("delivery_tag", d.DeliveryTag))
	}
}

func cleanupPaths(paths ...string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		_ = os.Remove(p)
	}
}

// deleteSourceObjects removes durable raw/cover objects once they are no
// longer needed (success or permanent failure). Retry/dead-letter paths keep
// them: they are the compensation input for requeue.
func deleteSourceObjects(oss objectStore, lg *zap.Logger, keys ...string) {
	for _, k := range keys {
		if k == "" {
			continue
		}
		if err := oss.DeleteObject(k); err != nil {
			lg.Warn("delete transcode source object", zap.String("key", k), zap.Error(err))
		}
	}
}

// regenerableExtras returns only artifacts that can be regenerated on the
// next attempt. The user-provided cover is a compensation input, not a
// disposable intermediate, so it is never included.
func regenerableExtras(outMP4, coverOut, finalCoverPath, userCoverPath string) []string {
	extras := []string{outMP4, coverOut}
	if finalCoverPath != "" && finalCoverPath != userCoverPath {
		extras = append(extras, finalCoverPath)
	}
	return extras
}

func failVideo(db *gorm.DB, id uint64, reason string) {
	msg := strings.TrimSpace(reason)
	if msg != "" {
		msg = ffmpeg.HumanizeFailReason(msg)
	}
	if msg == "" {
		msg = "视频处理失败，请稍后重试。"
	}
	if err := db.Model(&video.Video{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      video.StatusFailed,
		"fail_reason": truncate(msg, 1900),
	}).Error; err != nil && logger.L != nil {
		logger.L.Warn("fail video db update failed", zap.Uint64("video_id", id), zap.Error(err))
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	// Rune-safe: varchar(1900) counts characters, and slicing by bytes could
	// split a multi-byte UTF-8 sequence (e.g. Chinese in ffmpeg stderr),
	// producing invalid UTF-8 that MySQL utf8mb4 rejects with Error 1366.
	return string(r[:n])
}

// deadLetterTranscode publishes an exhausted job to the dead-letter queue and
// records it in the DB, so failed transcodes are observable and compensable.
func deadLetterTranscode(ctx context.Context, db *gorm.DB, pubCh transcodePublisher, lg *zap.Logger, job TranscodeJob, reason string) {
	incrTranscodeDeadLetters()
	if pubCh != nil {
		body, _ := json.Marshal(job)
		if err := pubCh.PublishConfirmed(ctx, "", queue.TranscodeDeadQueue, true, amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		}); err != nil {
			lg.Error("publish transcode dead letter", zap.Uint64("video_id", job.VideoID), zap.Error(err))
		}
	}
	if db == nil {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{"job": job, "reason": reason})
	rec := video.TranscodeDeadLetter{
		VideoID:     job.VideoID,
		Reason:      truncate(reason, 1900),
		RetryCount:  job.RetryCount,
		PayloadJSON: string(payload),
	}
	if err := db.Create(&rec).Error; err != nil {
		lg.Warn("record transcode dead letter", zap.Uint64("video_id", job.VideoID), zap.Error(err))
	}
}

// requeueOrFail re-enqueues on retryable failure and returns true (caller must preserve RawPath / user cover).
// On terminal failure returns false, and has already deleted only the
// intermediate files in terminalLocalExtras. RawPath/CoverPath are
// intentionally KEPT: the dead letter is the compensation trail, and the
// admin requeue re-runs ffmpeg from those files. They are removed either by
// a later successful transcode or by retention cleanup once the dead letter
// is resolved and never requeued.
func requeueOrFail(ctx context.Context, db *gorm.DB, pubCh transcodePublisher, lg *zap.Logger, job TranscodeJob, reason string, terminalLocalExtras ...string) bool {
	if job.RetryCount >= maxTranscodeRetries {
		deadLetterTranscode(ctx, db, pubCh, lg, job, reason)
		failVideo(db, job.VideoID, reason)
		cleanupPaths(terminalLocalExtras...)
		lg.Error("transcode exhausted retries", zap.Uint64("video_id", job.VideoID))
		return false
	}
	job.RetryCount++
	retryQueue := queue.RetryQueueForAttempt(job.RetryCount)
	body, _ := json.Marshal(job)
	if err := pubCh.PublishConfirmed(ctx, "", retryQueue, true, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	}); err != nil {
		lg.Error("schedule transcode retry", zap.Error(err), zap.String("retry_queue", retryQueue))
		deadLetterTranscode(ctx, db, pubCh, lg, job, reason)
		failVideo(db, job.VideoID, reason)
		cleanupPaths(terminalLocalExtras...)
		return false
	}
	lg.Info("transcode retry scheduled",
		zap.Uint64("video_id", job.VideoID),
		zap.String("retry_queue", retryQueue),
		zap.Int("retry", job.RetryCount))
	incrTranscodeRetriesScheduled()
	return true
}

// transcodeDeadSubscriber is the RabbitMQ surface the dead-letter consumer
// needs, kept small so the reconnect loop is unit-testable.
type transcodeDeadSubscriber interface {
	NewTranscodeDeadConsumer(consumerTag string) (interface{ Close() error }, <-chan amqp.Delivery, error)
}

// transcodeDeadRetryDelay is the reconnect backoff; variable for tests.
var transcodeDeadRetryDelay = 3 * time.Second

// StartTranscodeDeadConsumer drains the dead-letter queue, logging each
// exhausted job. The DB record is the durable audit/compensation trail.
// The loop reconnects with a backoff after channel loss.
func StartTranscodeDeadConsumer(ctx context.Context, cfg *config.C, db *gorm.DB, mq transcodeDeadSubscriber, lg *zap.Logger) {
	if mq == nil {
		return
	}
	for {
		if ctx.Err() != nil {
			return
		}
		ch, msgs, err := mq.NewTranscodeDeadConsumer("transcode-dead-worker")
		if err != nil {
			lg.Warn("transcode dead consumer: subscribe", zap.Error(err))
		} else {
			consumeTranscodeDead(ctx, ch, msgs, lg, db)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(transcodeDeadRetryDelay):
		}
	}
}

// consumeTranscodeDead drains one dead-letter channel. It returns when the
// channel closes or the context is cancelled; the caller reconnects.
func consumeTranscodeDead(ctx context.Context, ch interface{ Close() error }, msgs <-chan amqp.Delivery, lg *zap.Logger, db *gorm.DB) {
	defer ch.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-msgs:
			if !ok {
				return
			}
			handleTranscodeDeadLetter(d, lg, db)
		}
	}
}

// handleTranscodeDeadLetter marks the matching audit row processed and only
// then acks the message. A DB failure leaves the message unacked so it is
// redelivered and retried instead of being observably lost.
func handleTranscodeDeadLetter(d amqp.Delivery, lg *zap.Logger, db *gorm.DB) {
	var job TranscodeJob
	_ = json.Unmarshal(d.Body, &job)
	lg.Warn("transcode dead letter consumed",
		zap.Uint64("video_id", job.VideoID),
		zap.Int("retry_count", job.RetryCount),
	)
	if db != nil {
		if err := db.Model(&video.TranscodeDeadLetter{}).
			Where("video_id = ? AND retry_count = ? AND processed_at IS NULL", job.VideoID, job.RetryCount).
			Order("id DESC").
			Limit(1).
			Update("processed_at", time.Now()).Error; err != nil {
			lg.Warn("mark transcode dead letter processed", zap.Uint64("video_id", job.VideoID), zap.Error(err))
			_ = d.Nack(false, true)
			return
		}
	}
	_ = d.Ack(false)
}

const (
	transcodeDeadRetention     = 30 * 24 * time.Hour
	transcodeDeadCleanInterval = 24 * time.Hour
)

// StartTranscodeDeadRetention periodically archives resolved dead letters
// (processed or requeued) older than the retention window. Archiving sets
// archived_at and releases their source media instead of deleting the audit
// trail.
func StartTranscodeDeadRetention(ctx context.Context, db *gorm.DB, oss objectStore, lg *zap.Logger) {
	if db == nil {
		return
	}
	cleanupTranscodeDeadLetters(db, time.Now().Add(-transcodeDeadRetention), oss, lg)
	t := time.NewTicker(transcodeDeadCleanInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			cleanupTranscodeDeadLetters(db, time.Now().Add(-transcodeDeadRetention), oss, lg)
		}
	}
}

// cleanupTranscodeDeadLetters archives resolved dead letters (processed or
// requeued) older than the retention cutoff. It returns the archived row
// count.
func cleanupTranscodeDeadLetters(db *gorm.DB, cutoff time.Time, oss objectStore, lg *zap.Logger) int64 {
	// Rows that were consumed and never requeued are safe to archive: no
	// successor job references their raw media. Remove those files before
	// marking them archived. Rows that were requeued may still be feeding a
	// successor job, so their files are left untouched.
	var archival []video.TranscodeDeadLetter
	if err := db.Where("archived_at IS NULL AND processed_at IS NOT NULL AND processed_at < ? AND requeued_at IS NULL", cutoff).
		Find(&archival).Error; err != nil {
		lg.Warn("scan transcode dead letters for archival", zap.Error(err))
		return 0
	}
	for i := range archival {
		var payload struct {
			Job TranscodeJob `json:"job"`
		}
		if err := json.Unmarshal([]byte(archival[i].PayloadJSON), &payload); err == nil {
			cleanupPaths(payload.Job.RawPath, payload.Job.CoverPath)
			if oss != nil {
				deleteSourceObjects(oss, lg, payload.Job.RawKey, payload.Job.CoverKey)
			}
		}
	}
	res := db.Model(&video.TranscodeDeadLetter{}).
		Where("archived_at IS NULL AND ((processed_at IS NOT NULL AND processed_at < ?) OR (requeued_at IS NOT NULL AND requeued_at < ?))", cutoff, cutoff).
		Update("archived_at", time.Now())
	if res.Error != nil {
		lg.Warn("archive transcode dead letters", zap.Error(res.Error))
		return 0
	}
	if res.RowsAffected > 0 {
		lg.Info("archived transcode dead letters", zap.Int64("rows", res.RowsAffected))
	}
	return res.RowsAffected
}
