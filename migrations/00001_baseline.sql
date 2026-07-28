-- +goose Up
-- Baseline migration: all 42 domain tables already created by GORM AutoMigrate
-- (tracked as V1-V19 in schema_versions table). This file is a marker so
-- goose knows the baseline exists. Future migrations (V20+) go here as SQL.

-- +goose Down
-- Baseline cannot be rolled back. Schema changes before this point are
-- tracked in schema_versions and managed by GORM's AutoMigrateAll.
