package worker

import (
	"cakecake/internal/model/video"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	TranscodeToH264MP4(rawPath, outMP4 string) (stderr string, err error)
	ScreenshotJPEG(inPath, outPath string, second float64) (stderr string, err error)
	IsPermanentTranscodeFailure(stderr string) bool
}

type realFFmpegOps struct{}

// TranscodeToH264MP4 runs the real ffmpeg transcode.
func (realFFmpegOps) TranscodeToH264MP4(rawPath, outMP4 string) (string, error) {
	return ffmpeg.TranscodeToH264MP4(rawPath, outMP4)
}

// ScreenshotJPEG captures a frame from a video via ffmpeg.
func (realFFmpegOps) ScreenshotJPEG(inPath, outPath string, second float64) (string, error) {
	return ffmpeg.ScreenshotJPEG(inPath, outPath, second)
}

// IsPermanentTranscodeFailure reports whether ffmpeg stderr indicates a permanent failure.
func (realFFmpegOps) IsPermanentTranscodeFailure(stderr string) bool {
	return ffmpeg.IsPermanentTranscodeFailure(stderr)
}

// transcodePublisher is the minimal channel surface needed to republish a job.
type transcodePublisher interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

// objectStore is the minimal object-storage surface the pipeline needs.
type objectStore interface {
	UploadFile(objectKey, localPath string) error
}

// transcodeRetryBaseDelay is the base retry backoff; variable for tests.
var transcodeRetryBaseDelay = 30 * time.Second

// StartTranscodeConsumer runs a blocking AMQP consumer loop.
func StartTranscodeConsumer(ctx context.Context, cfg *config.C, db *gorm.DB, mq *queue.Client, ossClient *storage.OSS, esc *search.Client) {
	ch, err := mq.NewConsumerChannel()
	if err != nil {
		logger.L.Fatal("transcode: 无法打开消费 Channel（请检查 RabbitMQ）", zap.Error(err))
	}
	defer ch.Close()
	if err := ch.Qos(1, 0, false); err != nil {
		logger.L.Fatal("transcode: QoS 失败", zap.Error(err))
	}
	msgs, err := ch.Consume(queue.TranscodeQueue, "transcode-worker", false, false, false, false, nil)
	if err != nil {
		logger.L.Fatal("transcode: 无法订阅队列 "+queue.TranscodeQueue+"（任务将堆积、OSS 不会更新）", zap.Error(err))
	}
	pubCh, err := mq.NewConsumerChannel()
	if err != nil {
		logger.L.Fatal("transcode: 无法打开重投 Channel", zap.Error(err))
	}
	defer pubCh.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-msgs:
			if !ok {
				return
			}
			handleDelivery(ctx, cfg, db, ch, pubCh, ossClient, esc, d)
		}
	}
}

func handleDelivery(ctx context.Context, cfg *config.C, db *gorm.DB, ch, pubCh *amqp.Channel, ossClient *storage.OSS, esc *search.Client, d amqp.Delivery) {
	var store objectStore
	if ossClient != nil {
		store = ossClient
	}
	handleDeliveryWith(ctx, cfg, db, pubCh, store, esc, d, realFFmpegOps{}, logger.L)
}

func handleDeliveryWith(ctx context.Context, cfg *config.C, db *gorm.DB, pubCh transcodePublisher, ossClient objectStore, esc *search.Client, d amqp.Delivery, ff ffmpegOps, lg *zap.Logger) {
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

	outMP4 := filepath.Join(cfg.TempUploadDir, fmt.Sprintf("%d_out.mp4", job.VideoID))
	coverOut := filepath.Join(cfg.TempUploadDir, fmt.Sprintf("%d_cover.jpg", job.VideoID))
	_ = os.Remove(outMP4)
	_ = os.Remove(coverOut)

	lg.Info("transcode ffmpeg start", zap.Uint64("video_id", job.VideoID))
	stderr, err := ff.TranscodeToH264MP4(job.RawPath, outMP4)
	if err != nil {
		lg.Warn("ffmpeg transcode failed", zap.Uint64("video_id", job.VideoID), zap.Error(err), zap.String("stderr", stderr))
		if ff.IsPermanentTranscodeFailure(stderr) {
			failVideo(db, job.VideoID, strings.TrimSpace(stderr))
			cleanupPaths(job.RawPath, job.CoverPath, outMP4, coverOut, "")
			ackDelivery(lg, d)
			return
		}
		requeueOrFail(ctx, cfg, db, pubCh, lg, job, stderr, outMP4, coverOut)
		ackDelivery(lg, d)
		return
	}
	lg.Info("transcode ffmpeg done", zap.Uint64("video_id", job.VideoID))

	var finalCoverPath string
	var coverExt string
	if job.CoverPath != "" {
		finalCoverPath = job.CoverPath
		coverExt = strings.TrimPrefix(strings.ToLower(filepath.Ext(job.CoverPath)), ".")
		if coverExt == "jpeg" {
			coverExt = "jpg"
		}
	} else {
		// Default cover: captures a frame from the transcoded H.264 MP4 (more reliable than capturing from the source container).
		se, err := ff.ScreenshotJPEG(outMP4, coverOut, 1)
		if err != nil {
			lg.Warn("ffmpeg screenshot failed", zap.Error(err), zap.String("stderr", se))
			if ff.IsPermanentTranscodeFailure(se) {
				failVideo(db, job.VideoID, strings.TrimSpace(se))
				cleanupPaths(job.RawPath, job.CoverPath, outMP4, coverOut, "")
				ackDelivery(lg, d)
				return
			}
			requeueOrFail(ctx, cfg, db, pubCh, lg, job, se, outMP4, coverOut)
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
		if requeueOrFail(ctx, cfg, db, pubCh, lg, job, err.Error(), outMP4, coverOut, finalCoverPath) {
			// Retries still depend on RawPath / user cover: only delete regenerable intermediate artifacts (next round will re-transcode / re-capture).
			cleanupPaths(outMP4, coverOut)
		}
		ackDelivery(lg, d)
		return
	}
	if err := ossClient.UploadFile(coverKey, finalCoverPath); err != nil {
		lg.Error("oss upload cover", zap.Error(err))
		if requeueOrFail(ctx, cfg, db, pubCh, lg, job, err.Error(), outMP4, coverOut, finalCoverPath) {
			cleanupPaths(outMP4, coverOut)
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
	if err := db.Model(&video.Video{}).Where("id = ?", job.VideoID).Updates(updates).Error; err != nil {
		lg.Error("db update after transcode", zap.Error(err))
	} else if !cfg.VideoReviewRequired {
		if err := vsvc.PublishVideo(ctx, db, esc, lg, job.VideoID, nil); err != nil {
			lg.Error("publish video after transcode", zap.Error(err))
		}
	}
	cleanupPaths(job.RawPath, job.CoverPath, outMP4, coverOut, "")
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
		if err := pubCh.PublishWithContext(ctx, "", queue.TranscodeDeadQueue, false, false, amqp.Publishing{
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
func requeueOrFail(ctx context.Context, cfg *config.C, db *gorm.DB, pubCh transcodePublisher, lg *zap.Logger, job TranscodeJob, reason string, terminalLocalExtras ...string) bool {
	if job.RetryCount >= 3 {
		deadLetterTranscode(ctx, db, pubCh, lg, job, reason)
		failVideo(db, job.VideoID, reason)
		cleanupPaths(terminalLocalExtras...)
		lg.Error("transcode exhausted retries", zap.Uint64("video_id", job.VideoID))
		return false
	}
	wait := time.Duration(transcodeRetryBaseDelay.Nanoseconds() * int64(job.RetryCount+1))
	lg.Info("transcode retry scheduled", zap.Uint64("video_id", job.VideoID), zap.Duration("wait", wait), zap.Int("retry", job.RetryCount+1))
	time.Sleep(wait)
	job.RetryCount++
	body, _ := json.Marshal(job)
	if err := pubCh.PublishWithContext(ctx, "", queue.TranscodeQueue, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	}); err != nil {
		lg.Error("republish transcode job", zap.Error(err))
		deadLetterTranscode(ctx, db, pubCh, lg, job, reason)
		failVideo(db, job.VideoID, reason)
		cleanupPaths(terminalLocalExtras...)
		return false
	}
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

// handleTranscodeDeadLetter logs and acks a single dead letter, then marks the
// matching audit row processed so retention can eventually clean it up.
func handleTranscodeDeadLetter(d amqp.Delivery, lg *zap.Logger, db *gorm.DB) {
	var job TranscodeJob
	_ = json.Unmarshal(d.Body, &job)
	lg.Warn("transcode dead letter consumed",
		zap.Uint64("video_id", job.VideoID),
		zap.Int("retry_count", job.RetryCount),
	)
	_ = d.Ack(false)
	if db != nil {
		if err := db.Model(&video.TranscodeDeadLetter{}).
			Where("video_id = ? AND retry_count = ? AND processed_at IS NULL", job.VideoID, job.RetryCount).
			Order("id DESC").
			Limit(1).
			Update("processed_at", time.Now()).Error; err != nil {
			lg.Warn("mark transcode dead letter processed", zap.Uint64("video_id", job.VideoID), zap.Error(err))
		}
	}
}

const (
	transcodeDeadRetention     = 30 * 24 * time.Hour
	transcodeDeadCleanInterval = 24 * time.Hour
)

// StartTranscodeDeadRetention periodically archives resolved dead letters
// (processed or requeued) older than the retention window.
func StartTranscodeDeadRetention(ctx context.Context, db *gorm.DB, lg *zap.Logger) {
	if db == nil {
		return
	}
	cleanupTranscodeDeadLetters(db, time.Now().Add(-transcodeDeadRetention), lg)
	t := time.NewTicker(transcodeDeadCleanInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			cleanupTranscodeDeadLetters(db, time.Now().Add(-transcodeDeadRetention), lg)
		}
	}
}

// cleanupTranscodeDeadLetters deletes resolved dead letters (processed or
// requeued) older than the retention cutoff. It returns the deleted row count.
func cleanupTranscodeDeadLetters(db *gorm.DB, cutoff time.Time, lg *zap.Logger) int64 {
	// Rows that were consumed and never requeued are safe to archive: no
	// successor job references their raw media. Remove those files before
	// deleting the audit rows. Rows that were requeued may still be feeding a
	// successor job, so their files are left untouched.
	var archival []video.TranscodeDeadLetter
	if err := db.Where("processed_at IS NOT NULL AND processed_at < ? AND requeued_at IS NULL", cutoff).
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
		}
	}
	res := db.Where("processed_at IS NOT NULL AND processed_at < ?", cutoff).
		Or("requeued_at IS NOT NULL AND requeued_at < ?", cutoff).
		Delete(&video.TranscodeDeadLetter{})
	if res.Error != nil {
		lg.Warn("cleanup transcode dead letters", zap.Error(res.Error))
		return 0
	}
	if res.RowsAffected > 0 {
		lg.Info("cleaned transcode dead letters", zap.Int64("rows", res.RowsAffected))
	}
	return res.RowsAffected
}
