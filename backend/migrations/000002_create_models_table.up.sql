CREATE TABLE models (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version     VARCHAR(50) UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,  -- "VW Polo 5 Base Model"
    
    -- Хранилище
    weights_path TEXT NOT NULL,  -- MinIO path
    config_path  TEXT,           -- путь к конфигу модели
    
    -- Метаданные
    car_make     VARCHAR(100),   -- "Volkswagen"
    car_model    VARCHAR(100),   -- "Polo 5"
    
    -- Статус
    status       VARCHAR(20) DEFAULT 'training' 
                 CHECK (status IN ('training', 'ready', 'active', 'deprecated')),
    active       BOOLEAN DEFAULT FALSE,
    
    -- Метрики качества (из training)
    metrics_json JSONB,  -- {"accuracy": 0.95, "mAP": 0.89, "loss": 0.12}
    
    -- История
    parent_model_id UUID REFERENCES models(id),  -- для доменной адаптации
    trained_at      TIMESTAMPTZ,
    description     TEXT,
    created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by      UUID REFERENCES users(id)  -- кто создал (admin/owner)
);

-- Только одна активная модель
CREATE UNIQUE INDEX idx_models_active ON models(active) WHERE active = TRUE;
CREATE INDEX idx_models_status ON models(status);
CREATE INDEX idx_models_car_model ON models(car_model);

-- Trigger для автоматического updated_at (используем функцию из первой миграции)
CREATE TRIGGER update_models_updated_at
    BEFORE UPDATE ON models
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();