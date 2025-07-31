# Panduan Migrasi Database dengan golang-migrate

Dokumen ini menjelaskan pendekatan yang direkomendasikan untuk mengelola perubahan skema database di proyek Suar Backend menggunakan alat `golang-migrate/migrate`.

## 1. Apa itu Migrasi Database?

Migrasi database adalah cara mengelola perubahan inkremental dan reversibel pada skema database Anda dari waktu ke waktu. Anggap saja ini sebagai **sistem kontrol versi (seperti Git) untuk database Anda**.

Setiap kali Anda perlu mengubah struktur database (membuat tabel, menambah kolom, membuat indeks, dll.), Anda membuat sebuah *file migrasi*. File ini berisi skrip SQL untuk menerapkan (`up`) dan membatalkan (`down`) perubahan tersebut.

## 2. Mengapa Menggunakan Sistem Migrasi?

Meskipun kita tidak menggunakan ORM, pendekatan ini memberikan keuntungan besar:

-   **Versioning & Sejarah**: Menciptakan jejak audit yang jelas tentang bagaimana skema database berevolusi.
-   **Konsistensi**: Memastikan semua lingkungan (lokal, staging, production) memiliki skema database yang identik dan sinkron.
-   **Automasi**: Memudahkan proses setup dan deployment. Pengembang baru atau pipeline CI/CD dapat menyiapkan database ke versi terbaru dengan satu perintah.
-   **Reversibilitas**: Jika terjadi masalah setelah deployment, Anda dapat dengan mudah kembali (rollback) ke versi skema sebelumnya.
-   **Sumber Kebenaran Tunggal (Single Source of Truth)**: File migrasi di dalam repositori Git menjadi satu-satunya sumber kebenaran untuk struktur database, bukan database yang ada di mesin lokal seseorang.

## 3. Alat yang Direkomendasikan: `golang-migrate/migrate`

`golang-migrate/migrate` adalah alat baris perintah (CLI) yang sangat populer dan andal di ekosistem Go. Alat ini tidak terikat pada framework tertentu dan bekerja langsung dengan file SQL, yang sangat cocok untuk proyek kita.

### 3.1. Instalasi

Anda dapat menginstal CLI-nya menggunakan Go:

```sh
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Pastikan `$GOPATH/bin` ada di dalam `PATH` sistem Anda untuk menjalankan perintah `migrate` dari mana saja.

## 4. Langkah-langkah Implementasi

### 4.1. Struktur Direktori

Kita akan menyimpan semua file migrasi di dalam direktori berikut untuk menjaga kerapian:

```
internal/
└── database/
    └── migrations/
        ├── 000001_create_initial_tables.down.sql
        ├── 000001_create_initial_tables.up.sql
        └── ...
```

### 4.2. Membuat File Migrasi Baru

Untuk membuat file migrasi baru, kita gunakan perintah `migrate create`. Perintah ini akan menghasilkan dua file: satu untuk `up` dan satu untuk `down`.

```sh
migrate create -ext sql -dir internal/database/migrations -seq create_initial_tables
```

-   `-ext sql`: Menentukan ekstensi file adalah `.sql`.
-   `-dir ...`: Menentukan direktori tempat menyimpan file.
-   `-seq`: Menggunakan nomor sekuensial (000001, 000002, dst.).
-   `create_initial_tables`: Deskripsi singkat tentang tujuan migrasi ini.

Perintah di atas akan menghasilkan file:
-   `internal/database/migrations/000001_create_initial_tables.up.sql`
-   `internal/database/migrations/000001_create_initial_tables.down.sql`

### 4.3. Menulis Kode SQL Migrasi (Contoh)

Sekarang, kita isi file-file tersebut dengan SQL yang sebenarnya.

**Contoh 1: Membuat tabel `users` dan `request_history`**

`000001_create_initial_tables.up.sql`:

```sql
-- Fungsi untuk auto-update kolom updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Tabel Users
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Tabel Request History
CREATE TABLE request_history (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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

-- Indeks untuk foreign key
CREATE INDEX idx_request_history_user_id ON request_history(user_id);
```

`000001_create_initial_tables.down.sql` (untuk membatalkan):

```sql
-- Hapus dalam urutan terbalik untuk menghindari masalah foreign key
DROP TABLE IF EXISTS request_history;
DROP TABLE IF EXISTS users;

-- Hapus fungsi trigger
DROP FUNCTION IF EXISTS update_updated_at_column();
```

## 5. Menjalankan Migrasi

Untuk menerapkan migrasi ke database Anda, gunakan perintah `migrate up`.

```sh
migrate -database "postgres://user:password@localhost:5432/suar_db?sslmode=disable" -path internal/database/migrations up
```

-   `-database`: String koneksi ke database PostgreSQL Anda. **PENTING**: Gunakan variabel lingkungan untuk ini, jangan di-hardcode.
-   `-path`: Path ke direktori file migrasi Anda.
-   `up`: Perintah untuk menerapkan semua migrasi yang belum dijalankan.

Alat `migrate` akan membuat tabel di dalam database Anda bernama `schema_migrations` untuk melacak versi migrasi mana yang sudah diterapkan.

### Membatalkan Migrasi

Untuk membatalkan migrasi terakhir, gunakan `down 1`:

```sh
migrate -database "..." -path internal/database/migrations down 1
```

## 6. Kesimpulan

Mengadopsi sistem migrasi sejak awal akan sangat membantu dalam menjaga proyek tetap terorganisir, dapat diandalkan, dan mudah untuk dikembangkan baik secara solo maupun dalam tim. Ini adalah investasi kecil dalam praktik rekayasa perangkat lunak yang baik yang akan membuahkan hasil besar di masa depan.
