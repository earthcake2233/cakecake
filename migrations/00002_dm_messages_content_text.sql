-- +goose Up
-- Enlarge dm_messages.content from VARCHAR(500) to TEXT so long AI assistant
-- replies are not truncated at 500 characters.
ALTER TABLE `dm_messages` MODIFY COLUMN `content` TEXT NOT NULL;

-- +goose Down
ALTER TABLE `dm_messages` MODIFY COLUMN `content` VARCHAR(500) NOT NULL;
