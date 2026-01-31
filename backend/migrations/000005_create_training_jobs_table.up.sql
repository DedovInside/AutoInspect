CREATE TABLE training_jobs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id  UUID NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    base_model_id UUID NOT NULL REFERENCES models(id),  -- откуда начинаем
    
    status      VARCHAR(20) NOT NULL DEFAULT 'queued'
                CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
    
    -- Параметры обучения
    params_json JSONB NOT NULL,  -- {"epochs": 10, "batch_size": 16, "lr": 0.001, "optimizer": "adam"}
    
    -- Результат
    result_model_id UUID REFERENCES models(id),  -- новая обученная модель
    
    -- Логи и метрики
    logs_path   TEXT,  -- путь к логам в MinIO
    metrics_json JSONB,  -- промежуточные метрики по эпохам
    
    -- Временные метки
    created_at   TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Система
    worker_id    VARCHAR(100),  -- какой воркер обрабатывал
    error_message TEXT,
    
    -- Уведомления
    webhook_url  TEXT,
    notified_at  TIMESTAMPTZ
);

CREATE INDEX idx_training_jobs_dataset_id ON training_jobs(dataset_id);
CREATE INDEX idx_training_jobs_status ON training_jobs(status);
CREATE INDEX idx_training_jobs_created_at ON training_jobs(created_at DESC);

-- Примечание: updated_at не требуется, т.к. job не редактируется после создания