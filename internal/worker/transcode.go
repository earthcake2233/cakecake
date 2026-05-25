package worker

import (
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

	"minibili/internal/config"
	"minibili/internal/ffmpeg"
	"minibili/internal/logger"
	"minibili/internal/model"
	"minibili/internal/queue"
	"minibili/internal/search"
	"minibili/internal/service"
	"minibili/internal/storage"
)

// TranscodeJob is the JSON payload on the transcode queue.
type TranscodeJob struct {
	VideoID    uint64 `json:"video_id"`
	RawPath    string `json:"raw_path"`
	CoverPath  string `json:"cover_path,omitempty"`
	RetryCount int    `json:"retry_count"`
}

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
	lg := logger.L
	var job TranscodeJob
	if err := json.Unmarshal(d.Body, &job); err != nil {
		lg.Error("transcode bad json", zap.Error(err))
		_ = d.Ack(false)
		return
	}
	lg.Info("transcode job received", zap.Uint64("video_id", job.VideoID), zap.String("raw", job.RawPath))
	if ossClient == nil {
		lg.Error("oss not configured, failing job", zap.Uint64("video_id", job.VideoID))
		failVideo(db, job.VideoID, "OSS 未配置")
		cleanupPaths(job.RawPath, job.CoverPath, "", "", "")
		_ = d.Ack(false)
		return
	}

	outMP4 := filepath.Join(cfg.TempUploadDir, fmt.Sprintf("%d_out.mp4", job.VideoID))
	coverOut := filepath.Join(cfg.TempUploadDir, fmt.Sprintf("%d_cover.jpg", job.VideoID))
	_ = os.Remove(outMP4)
	_ = os.Remove(coverOut)

	lg.Info("transcode ffmpeg start", zap.Uint64("video_id", job.VideoID))
	stderr, err := ffmpeg.TranscodeToH264MP4(job.RawPath, outMP4)
	if err != nil {
		lg.Warn("ffmpeg transcode failed", zap.Uint64("video_id", job.VideoID), zap.Error(err), zap.String("stderr", stderr))
		if ffmpeg.IsPermanentTranscodeFailure(stderr) {
			failVideo(db, job.VideoID, strings.TrimSpace(stderr))
			cleanupPaths(job.RawPath, job.CoverPath, outMP4, coverOut, "")
			_ = d.Ack(false)
			return
		}
		requeueOrFail(ctx, cfg, db, pubCh, lg, job, stderr, outMP4, coverOut)
		_ = d.Ack(false)
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
		// 默认封面：对已转码的 H.264 MP4 截帧（比直接截原始容器更稳）
		se, err := ffmpeg.ScreenshotJPEG(outMP4, coverOut, 1)
		if err != nil {
			lg.Warn("ffmpeg screenshot failed", zap.Error(err), zap.String("stderr", se))
			if ffmpeg.IsPermanentTranscodeFailure(se) {
				failVideo(db, job.VideoID, strings.TrimSpace(se))
				cleanupPaths(job.RawPath, job.CoverPath, outMP4, coverOut, "")
				_ = d.Ack(false)
				return
			}
			requeueOrFail(ctx, cfg, db, pubCh, lg, job, se, outMP4, coverOut)
			_ = d.Ack(false)
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
			// 重试仍依赖 RawPath / 用户封面：只删可再生中间产物（下一轮会重新转码 / 截帧）
			cleanupPaths(outMP4, coverOut)
		}
		_ = d.Ack(false)
		return
	}
	if err := ossClient.UploadFile(coverKey, finalCoverPath); err != nil {
		lg.Error("oss upload cover", zap.Error(err))
		if requeueOrFail(ctx, cfg, db, pubCh, lg, job, err.Error(), outMP4, coverOut, finalCoverPath) {
			cleanupPaths(outMP4, coverOut)
		}
		_ = d.Ack(false)
		return
	}

	videoURL := cfg.OSSObjectURL(videoKey)
	coverURL := cfg.OSSObjectURL(coverKey)

	updates := map[string]interface{}{
		"video_url": videoURL,
		"cover_url": coverURL,
	}
	if cfg.VideoReviewRequired {
		updates["status"] = "pending_review"
	}
	if err := db.Model(&model.Video{}).Where("id = ?", job.VideoID).Updates(updates).Error; err != nil {
		lg.Error("db update after transcode", zap.Error(err))
	} else if !cfg.VideoReviewRequired {
		if err := service.PublishVideo(ctx, db, esc, lg, job.VideoID, nil); err != nil {
			lg.Error("publish video after transcode", zap.Error(err))
		}
	}
	cleanupPaths(job.RawPath, job.CoverPath, outMP4, coverOut, "")
	lg.Info("transcode completed", zap.Uint64("video_id", job.VideoID))
	_ = d.Ack(false)
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
	_ = db.Model(&model.Video{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      "failed",
		"fail_reason": truncate(msg, 1900),
	}).Error
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// requeueOrFail 在可重试时重新入队并返回 true（调用方须保留 RawPath / 用户封面）。
// 终局失败时返回 false，并已删除 RawPath、CoverPath 及 terminalLocalExtras 中的本地文件。
func requeueOrFail(ctx context.Context, cfg *config.C, db *gorm.DB, pubCh *amqp.Channel, lg *zap.Logger, job TranscodeJob, reason string, terminalLocalExtras ...string) bool {
	if job.RetryCount >= 3 {
		failVideo(db, job.VideoID, reason)
		cleanupPaths(append([]string{job.RawPath, job.CoverPath}, terminalLocalExtras...)...)
		lg.Error("transcode exhausted retries", zap.Uint64("video_id", job.VideoID))
		return false
	}
	wait := time.Duration(30*(job.RetryCount+1)) * time.Second
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
		failVideo(db, job.VideoID, reason)
		cleanupPaths(append([]string{job.RawPath, job.CoverPath}, terminalLocalExtras...)...)
		return false
	}
	return true
}
