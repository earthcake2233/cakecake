-- +goose Up
CREATE TABLE IF NOT EXISTS `search_sync_outbox` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `entity_type` varchar(16) NOT NULL,
  `entity_id` bigint unsigned NOT NULL,
  `action` varchar(8) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `attempts` bigint NOT NULL DEFAULT 0,
  `next_retry_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_search_sync_status_retry` (`status`,`next_retry_at`),
  KEY `idx_search_sync_entity` (`entity_type`,`entity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS `search_sync_outbox`;
