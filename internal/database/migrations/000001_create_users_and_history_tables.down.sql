-- +migrate Down

-- Hapus dalam urutan terbalik untuk menghindari masalah ketergantungan (dependency issues).
DROP TABLE IF EXISTS request_history;
DROP TABLE IF EXISTS users;

-- Hapus fungsi yang tidak lagi digunakan.
DROP FUNCTION IF EXISTS update_updated_at_column();