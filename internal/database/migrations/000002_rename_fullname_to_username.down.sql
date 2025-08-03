-- +migrate Down
ALTER TABLE users RENAME COLUMN username TO full_name;