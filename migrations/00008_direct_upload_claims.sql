-- +goose Up
CREATE TABLE IF NOT EXISTS `direct_upload_claims` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `raw_key` varchar(255) NOT NULL,
  `user_id` bigint NOT NULL,
  `video_id` bigint NOT NULL DEFAULT 0,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_direct_upload_claims_raw_key` (`raw_key`),
  KEY `idx_direct_upload_claims_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS `direct_upload_claims`;
