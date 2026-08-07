-- +goose Up
-- Model-generated follow-up question chips for AI assistant replies (JSON array).
ALTER TABLE `dm_messages` ADD COLUMN `suggestions` TEXT NULL;

-- +goose Down
ALTER TABLE `dm_messages` DROP COLUMN `suggestions`;
