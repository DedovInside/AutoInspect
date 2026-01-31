CREATE TABLE analyses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    status          VARCHAR(20) NOT NULL DEFAULT 'queued' 
                    CHECK (status IN ('queued', 'processing', 'completed', 'failed', 'cancelled')),
    
    -- Хранение изображения
    image_key       VARCHAR(500) NOT NULL,     -- увеличил до 500
    image_metadata  JSONB,                     -- {"size": 1024000, "format": "jpg", "dimensions": {"width": 1920, "height": 1080}}
    
    -- ML модель
    model_version   VARCHAR(50) NOT NULL,      -- обязательное поле
    model_id        UUID REFERENCES models(id), -- ссылка на конкретную модель
    
    -- Результаты анализа
    result_json     JSONB,
    /*
    Структура result_json:
    {
      "view_angle": "front|rear|side_left|side_right",
      "defects": [
        {
          "id": "defect_1",
          "part_name": "front_bumper",
          "part_id": "bumper_01",
          "defect_type": "scratch|dent|crack|broken_glass",
          "severity": "minor|major",
          "bbox": {"x": 100, "y": 200, "width": 50, "height": 60},
          "mask": "base64_encoded_or_url",
          "confidence": 0.95,
          "recommended_action": "paint|replace"
        }
      ],
      "summary": {
        "total_defects": 3,
        "critical_count": 1,
        "estimated_cost": null  // на будущее
      }
    }
    */
    
    -- Ошибки
    error_message   TEXT,
    error_code      VARCHAR(50),  -- для классификации ошибок
    retry_count     INTEGER DEFAULT 0,  -- для повторных попыток
    
    -- Временные метки
    created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    queued_at       TIMESTAMPTZ,    -- когда попал в очередь
    processing_at   TIMESTAMPTZ,    -- когда начал обработку
    completed_at    TIMESTAMPTZ,    -- когда завершил
    
    -- Метрики
    processing_time_ms BIGINT,
    queue_wait_time_ms BIGINT,     -- время ожидания в очереди
    
    -- Опционально: webhook для уведомлений
    webhook_url     TEXT,
    webhook_sent_at TIMESTAMPTZ
);

-- Индексы
CREATE INDEX idx_analyses_user_id ON analyses(user_id);
CREATE INDEX idx_analyses_status ON analyses(status);
CREATE INDEX idx_analyses_created_at ON analyses(created_at DESC);
CREATE INDEX idx_analyses_user_created ON analyses(user_id, created_at DESC);  -- композитный для истории
CREATE INDEX idx_analyses_result_json ON analyses USING GIN (result_json);

-- Trigger для автоматического updated_at
CREATE TRIGGER update_analyses_updated_at
    BEFORE UPDATE ON analyses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Для поиска по типу дефектов (пример JSONB запроса)
-- SELECT * FROM analyses WHERE result_json @> '{"defects": [{"defect_type": "scratch"}]}'];