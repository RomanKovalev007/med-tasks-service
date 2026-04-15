ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS date_scheduled_at  DATE,                                                       
    ADD COLUMN IF NOT EXISTS time_scheduled_at  TIME,                                                       
    ADD COLUMN IF NOT EXISTS is_periodicity     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS repeat_rule        JSONB;
