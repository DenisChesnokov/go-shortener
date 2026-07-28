CREATE TABLE IF NOT EXISTS shortener (
    short_url VARCHAR(255) PRIMARY KEY,
    original_url TEXT NOT NULL
);