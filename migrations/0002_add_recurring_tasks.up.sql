-- Task templates (recurrence rules)
CREATE TABLE IF NOT EXISTS task_templates (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    recurrence_type VARCHAR(50) NOT NULL,
    recurrence_params JSONB NOT NULL DEFAULT '{}',
    start_date DATE NOT NULL,
    end_date DATE,
    last_generated_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add new fields to tasks table for recurrence support
ALTER TABLE tasks ADD COLUMN template_id BIGINT REFERENCES task_templates(id) ON DELETE CASCADE;
ALTER TABLE tasks ADD COLUMN scheduled_date DATE;

-- Unique index to prevent duplicate tasks per template per date
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_template_id_scheduled_date ON tasks (template_id, scheduled_date);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_task_templates_recurrence_type ON task_templates (recurrence_type);
CREATE INDEX IF NOT EXISTS idx_task_templates_start_date ON task_templates (start_date);
CREATE INDEX IF NOT EXISTS idx_task_templates_end_date ON task_templates (end_date);
CREATE INDEX IF NOT EXISTS idx_tasks_scheduled_date ON tasks (scheduled_date);
