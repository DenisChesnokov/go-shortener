-- 1. Сначала удаляем старый бизнес-PK
ALTER TABLE shortener DROP CONSTRAINT IF EXISTS shortener_pkey;

-- 2. Добавляем новый суррогатный PK
ALTER TABLE shortener ADD COLUMN id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY;

-- 3. Меняем тип original_url на VARCHAR
ALTER TABLE shortener ALTER COLUMN original_url TYPE VARCHAR(2048);

-- 4. short_url становится UNIQUE
ALTER TABLE shortener ADD CONSTRAINT uniq_short_url UNIQUE (short_url);