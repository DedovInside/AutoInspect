CREATE TABLE datasets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Тип датасета
    dataset_type VARCHAR(50) DEFAULT 'user_upload'
                 CHECK (dataset_type IN ('user_upload', 'public', 'generated')),
    
    -- Статус
    status      VARCHAR(20) DEFAULT 'pending' 
                CHECK (status IN ('pending', 'processing', 'ready', 'failed')),
    
    -- Хранилище
    file_key    VARCHAR(500),  -- zip в MinIO
    total_size_bytes BIGINT,
    
    -- Метаданные разметки
    annotation_format VARCHAR(50),  -- "COCO", "YOLO", "custom"
    images_count      INTEGER DEFAULT 0,
    annotations_count INTEGER DEFAULT 0,
    classes_json      JSONB,  -- список классов дефектов
    
    -- Для какой марки
    car_make     VARCHAR(100),
    car_model    VARCHAR(100),
    
    -- История
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    validated_at TIMESTAMPTZ  -- когда прошла валидацию
);

CREATE INDEX idx_datasets_owner_id ON datasets(owner_id);
CREATE INDEX idx_datasets_status ON datasets(status);
CREATE INDEX idx_datasets_car_model ON datasets(car_model);

-- Trigger для автоматического updated_at
CREATE TRIGGER update_datasets_updated_at
    BEFORE UPDATE ON datasets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();