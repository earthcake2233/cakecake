-- +goose Up
CREATE TABLE IF NOT EXISTS `transcode_job_dedup` (
  `job_id` varchar(64) NOT NULL,
  `retry_count` bigint NOT NULL,
  `video_id` bigint NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`job_id`,`retry_count`),
  KEY `idx_transcode_job_dedup_video_id` (`video_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS `transcode_job_dedup`;
