-- +goose Up

-- Сначала обновляем существующие записи: если transcription NULL, устанавливаем пустую строку
-- (это временное значение, так как мы не можем знать реальную транскрипцию)
UPDATE pronunciations
SET transcription = ''
WHERE transcription IS NULL;

-- Теперь делаем transcription обязательным
ALTER TABLE pronunciations
ALTER COLUMN transcription SET NOT NULL;

-- Делаем audio_url опциональным (убираем NOT NULL)
ALTER TABLE pronunciations
ALTER COLUMN audio_url DROP NOT NULL;

COMMENT ON COLUMN pronunciations.transcription IS 'Транскрипция в формате IPA (обязательное поле)';
COMMENT ON COLUMN pronunciations.audio_url IS 'URL аудио файла с произношением (опциональное поле)';

-- +goose Down

-- Возвращаем audio_url как обязательное
ALTER TABLE pronunciations
ALTER COLUMN audio_url SET NOT NULL;

-- Возвращаем transcription как опциональное
ALTER TABLE pronunciations
ALTER COLUMN transcription DROP NOT NULL;











