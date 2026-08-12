-- +goose Up
CREATE TABLE IF NOT EXISTS `transcode_dead_letters` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `video_id` bigint unsigned NOT NULL,
  `reason` varchar(1900) NOT NULL DEFAULT '',
  `retry_count` bigint NOT NULL DEFAULT 0,
  `payload_json` text,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_transcode_dead_letters_video_id` (`video_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose Down
DROP TABLE IF EXISTS `transcode_dead_letters`;
