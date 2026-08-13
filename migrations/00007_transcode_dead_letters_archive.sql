-- +goose Up
ALTER TABLE `transcode_dead_letters`
  ADD COLUMN `archived_at` datetime(3) DEFAULT NULL;

-- +goose Down
ALTER TABLE `transcode_dead_letters`
  DROP COLUMN `archived_at`;
