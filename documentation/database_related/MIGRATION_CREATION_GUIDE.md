# Panduan Langkah-demi-Langkah: Membuat Migrasi Pertama

Dokumen ini memberikan panduan praktis untuk membuat file migrasi SQL pertama Anda untuk tabel `users` dan `request_history` menggunakan `golang-migrate/migrate`.

Ikuti langkah-langkah ini untuk memastikan proses yang bersih dan dapat diulang.

---

### Prasyarat: Instalasi `migrate` CLI

Pastikan Anda sudah menginstal `golang-migrate/migrate` CLI. Jika belum, jalankan perintah berikut di terminal Anda:

```sh
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Pastikan direktori `$GOPATH/bin` Anda ada di dalam `PATH` sistem Anda.

---

### Langkah 1: Buat Direktori Migrasi

Jika belum ada, buat direktori untuk menyimpan file-file migrasi Anda. Struktur yang disarankan adalah:

```sh
mkdir -p internal/database/migrations
```

### Langkah 2: Hasilkan File Migrasi Baru

Jalankan perintah berikut dari direktori root proyek Anda untuk membuat file migrasi pertama. Kita akan menamainya `create_users_and_history_tables`.

```sh
migrate create -ext sql -dir internal/database/migrations -seq create_users_and_history_tables
```

**Penjelasan Perintah:**
-   `create`: Perintah untuk membuat file migrasi baru.
-   `-ext sql`: Menentukan bahwa ekstensi file adalah `.sql`.
-   `-dir internal/database/migrations`: Menentukan lokasi file akan dibuat.
-   `-seq`: Menggunakan nomor urut (sekuensial) untuk penamaan file (misalnya, `000001_...`).

Setelah menjalankan perintah ini, Anda akan melihat dua file baru di dalam direktori `internal/database/migrations`:
1.  `000001_create_users_and_history_tables.up.sql`
2.  `000001_create_users_and_history_tables.down.sql`

### Langkah 3: Isi File Migrasi `.up.sql`

Buka file `...up.sql`. File ini berisi skrip untuk **menerapkan** perubahan Anda. Salin dan tempel kode SQL berikut ke dalamnya. Kode ini akan membuat tabel `users` dan `request_history` sesuai dengan skema yang telah kita setujui.

```sql
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

```

### Langkah 4: Isi File Migrasi `.down.sql`

Buka file `...down.sql`. File ini berisi skrip untuk **membatalkan** atau **mengembalikan** perubahan yang dibuat oleh file `.up.sql`. Ini sangat penting untuk keamanan dan reversibilitas.

Salin dan tempel kode SQL berikut. Perhatikan bahwa urutan `DROP` adalah kebalikan dari urutan `CREATE` untuk menghindari error karena *foreign key constraint*.

```sql
-- +migrate Down

-- Hapus dalam urutan terbalik untuk menghindari masalah ketergantungan (dependency issues).
DROP TABLE IF EXISTS request_history;
DROP TABLE IF EXISTS users;

-- Hapus fungsi yang tidak lagi digunakan.
DROP FUNCTION IF EXISTS update_updated_at_column();

```

### Langkah 5: Jalankan Migrasi (Untuk Anda Lakukan Sendiri)

Setelah kedua file tersebut disimpan, Anda siap untuk menjalankan migrasi. Buka terminal Anda dan jalankan perintah berikut, **pastikan untuk mengganti string koneksi database dengan milik Anda**.

```sh
migrate -database "postgres://USERNAME:PASSWORD@HOST:PORT/DATABASE_NAME?sslmode=disable" -path internal/database/migrations up
```

**Contoh String Koneksi Lokal:**
`postgres://postgres:mysecretpassword@localhost:5432/suar_db?sslmode=disable`

Setelah perintah ini dijalankan, alat `migrate` akan:
1.  Terhubung ke database Anda.
2.  Mengeksekusi skrip di dalam file `...up.sql`.
3.  Membuat tabel `schema_migrations` (jika belum ada) dan mencatat bahwa migrasi versi `1` telah berhasil diterapkan.

### Langkah 6: Verifikasi Hasil

Untuk memastikan semuanya berjalan lancar, hubungkan ke database Anda menggunakan `psql` atau alat GUI database favorit Anda dan periksa apakah tabel `users`, `request_history`, dan `schema_migrations` telah berhasil dibuat.

Anda sekarang telah berhasil membuat dan (nantinya) menjalankan migrasi pertama Anda!
