-- +migrate Up
CREATE TABLE car_service_reviews
(
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repair_request_id      UUID        NOT NULL UNIQUE REFERENCES repair_requests (id) ON DELETE CASCADE,
    car_service_profile_id UUID        NOT NULL REFERENCES car_service_profiles (id) ON DELETE CASCADE,
    user_id                UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    rating                 INTEGER     NOT NULL,
    author_name            VARCHAR(255),
    comment                TEXT,

    created_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_car_service_reviews_rating
        CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT chk_car_service_reviews_author_name_not_blank
        CHECK (author_name IS NULL OR char_length(trim(author_name)) > 0),
    CONSTRAINT chk_car_service_reviews_comment_not_blank
        CHECK (comment IS NULL OR char_length(trim(comment)) > 0)
);

CREATE INDEX idx_car_service_reviews_profile_created
    ON car_service_reviews (car_service_profile_id, created_at DESC);

CREATE INDEX idx_car_service_reviews_user_created
    ON car_service_reviews (user_id, created_at DESC);

CREATE INDEX idx_car_service_reviews_rating
    ON car_service_reviews (car_service_profile_id, rating);

CREATE TRIGGER update_car_service_reviews_updated_at
    BEFORE UPDATE ON car_service_reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE car_service_reviews IS 'Отзывы пользователей об автосервисах по завершённым ремонтным заявкам';
COMMENT ON COLUMN car_service_reviews.repair_request_id IS 'Ремонтная заявка, по которой оставлен отзыв. По одной заявке допускается только один отзыв';
COMMENT ON COLUMN car_service_reviews.author_name IS 'Имя автора, сохранённое на момент публикации отзыва';
