ALTER TABLE shortener ADD CONSTRAINT shortener_pkey PRIMARY KEY (short_url);
ALTER TABLE shortener DROP CONSTRAINT uniq_short_url;
ALTER TABLE shortener ALTER COLUMN original_url TYPE TEXT;
ALTER TABLE shortener DROP COLUMN id;