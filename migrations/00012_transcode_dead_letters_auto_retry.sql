-- +goose Up
ALTER TABLE `transcode_dead_letters`
  ADD COLUMN `auto_retry_count` bigint NOT NULL DEFAULT 0,
  ADD COLUMN `last_auto_retry_at` datetime(3) DEFAULT NULL;

-- +goose Down
ALTER TABLE `transcode_dead_letters`
  DROP COLUMN `last_auto_retry_at`,
  DROP COLUMN `auto_retry_count`;
