-- +goose Up
CREATE TABLE IF NOT EXISTS `transcode_outbox` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `job_id` varchar(64) NOT NULL,
  `video_id` bigint NOT NULL,
  `payload` text NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `attempts` bigint NOT NULL DEFAULT 0,
  `next_retry_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `sent_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_transcode_outbox_job_id` (`job_id`),
  KEY `idx_transcode_outbox_video_id` (`video_id`),
  KEY `idx_transcode_outbox_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS `transcode_outbox`;
