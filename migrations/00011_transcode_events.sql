-- +goose Up
CREATE TABLE IF NOT EXISTS `transcode_events` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `video_id` bigint NOT NULL,
  `job_id` varchar(64) NOT NULL DEFAULT '',
  `from_status` varchar(32) NOT NULL DEFAULT '',
  `to_status` varchar(32) NOT NULL DEFAULT '',
  `reason` varchar(1900) NOT NULL DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_transcode_events_video_id` (`video_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS `transcode_events`;
