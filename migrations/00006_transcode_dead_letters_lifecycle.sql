-- +goose Up
ALTER TABLE `transcode_dead_letters`
  ADD COLUMN `processed_at` datetime(3) DEFAULT NULL,
  ADD COLUMN `requeued_at` datetime(3) DEFAULT NULL,
  ADD COLUMN `requeued_count` bigint NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE `transcode_dead_letters`
  DROP COLUMN `requeued_count`,
  DROP COLUMN `requeued_at`,
  DROP COLUMN `processed_at`;
