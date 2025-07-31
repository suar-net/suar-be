-- +migrate Up

-- Fungsi untuk secara otomatis memperbarui kolom updated_at saat ada perubahan baris.
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Membuat tabel 'users'
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Menerapkan trigger ke tabel 'users'
CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Membuat tabel 'request_history'
CREATE TABLE request_history (
    id SERIAL PRIMARY KEY,
    -- user_id bisa NULL untuk fleksibilitas di masa depan (pengguna anonim).
    -- ON DELETE SET NULL berarti jika user dihapus, history-nya menjadi anonim.
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    executed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    request_method VARCHAR(10) NOT NULL,
    request_url TEXT NOT NULL,
    request_headers JSONB,
    request_body TEXT,
    response_status_code INTEGER,
    response_headers JSONB,
    response_body TEXT,
    response_size BIGINT,
    duration_ms INTEGER
);

-- Membuat indeks pada foreign key untuk performa query yang lebih cepat.
CREATE INDEX idx_request_history_user_id ON request_history(user_id);