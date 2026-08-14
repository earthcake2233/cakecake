-- +goose Up
ALTER TABLE `videos` CHANGE COLUMN `draft_raw_path` `draft_raw_key` VARCHAR(1024) NULL;
ALTER TABLE `videos` CHANGE COLUMN `draft_cover_path` `draft_cover_key` VARCHAR(1024) NULL;

-- +goose Down
ALTER TABLE `videos` CHANGE COLUMN `draft_raw_key` `draft_raw_path` VARCHAR(1024) NULL;
ALTER TABLE `videos` CHANGE COLUMN `draft_cover_key` `draft_cover_path` VARCHAR(1024) NULL;
