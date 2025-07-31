# Rancangan Skema Database Suar

Dokumen ini menguraikan rancangan skema database PostgreSQL untuk aplikasi Suar. Skema ini dirancang untuk mendukung fungsionalitas saat ini dan masa depan, dengan fokus pada normalisasi, skalabilitas, dan kejelasan.

## 1. Prinsip Desain

- **Normalisasi**: Untuk mengurangi redundansi data dan meningkatkan integritas data.
- **Keterbacaan**: Nama tabel dan kolom dibuat sejelas mungkin untuk mencerminkan tujuannya.
- **Ekstensibilitas**: Didesain untuk memudahkan penambahan fitur baru seperti *team*, *monitoring*, dll., di masa depan.

## 2. Diagram Hubungan Entitas (ERD) Konseptual

Berikut adalah diagram sederhana yang menggambarkan hubungan antar tabel utama:

```
+-----------+       +-------------------+
|   users   |       | request_history   |
+-----------+       +-------------------+
| id (PK)   |----<--| id (PK)           |
| email     |       | user_id (FK, NULL)|
| ...       |       | collection_id(FK,NULL)|
+-----------+       | ...               |
      |             +-------------------+
      |
      |       +-----------------------+
      +----<--|      collections      |
      |       +-----------------------+
      |       | id (PK)               |
      |       | user_id (FK)          |
      |       | ...                   |
      |       +-----------------------+
      |
      |       +-----------------------+
      +----<--|     environments      |
              +-----------------------+
              | id (PK)               |
              | user_id (FK)          |
              | ...                   |
              +-----------------------+
                      |
                      |       +---------------------------+
                      +----<--|   environment_variables   |
                              +---------------------------+
                              | id (PK)                   |
                              | environment_id (FK)       |
                              | ...                       |
                              +---------------------------+
```

## 3. Definisi Tabel

### Tabel: `users`

Menyimpan informasi kredensial dan profil pengguna.

| Nama Kolom      | Tipe Data          | Kendala/Catatan                               |
|-----------------|--------------------|-----------------------------------------------|
| `id`            | `SERIAL PRIMARY KEY` | Identifier unik untuk setiap pengguna.        |
| `full_name`     | `VARCHAR(255)`     | `NOT NULL`                                    |
| `email`         | `VARCHAR(255)`     | `UNIQUE NOT NULL`                             |
| `password_hash` | `VARCHAR(255)`     | `NOT NULL` - Kata sandi yang sudah di-hash.   |
| `created_at`    | `TIMESTAMPTZ`      | `DEFAULT CURRENT_TIMESTAMP`                   |
| `updated_at`    | `TIMESTAMPTZ`      | `DEFAULT CURRENT_TIMESTAMP`                   |

---

### Tabel: `request_history`

Mencatat setiap permintaan yang dibuat. Awalnya hanya untuk pengguna yang login, namun dirancang untuk bisa mendukung pengguna anonim di masa depan.

| Nama Kolom             | Tipe Data          | Kendala/Catatan                                                              |
|------------------------|--------------------|------------------------------------------------------------------------------|
| `id`                   | `SERIAL PRIMARY KEY` | Identifier unik untuk setiap catatan riwayat.                                |
| `user_id`              | `INTEGER`          | `NULL, REFERENCES users(id) ON DELETE SET NULL`. **Bisa NULL** untuk mendukung riwayat pengguna anonim di masa depan. Jika pengguna dihapus, riwayatnya menjadi anonim. |
| `collection_id`        | `INTEGER`          | `NULL, REFERENCES collections(id) ON DELETE SET NULL` - Opsional.            |
| `executed_at`          | `TIMESTAMPTZ`      | `DEFAULT CURRENT_TIMESTAMP` - Waktu permintaan dieksekusi.                   |
| `request_method`       | `VARCHAR(10)`      | `NOT NULL` - (e.g., 'GET', 'POST').                                          |
| `request_url`          | `TEXT`             | `NOT NULL` - URL target.                                                     |
| `request_headers`      | `JSONB`            | `NULL` - Header permintaan dalam format JSON.                                |
| `request_body`         | `TEXT`             | `NULL` - Body permintaan.                                                    |
| `response_status_code` | `INTEGER`          | `NULL` - Kode status dari respons.                                           |
| `response_headers`     | `JSONB`            | `NULL` - Header respons dalam format JSON.                                   |
| `response_body`        | `TEXT`             | `NULL` - Body respons.                                                       |
| `response_size`        | `BIGINT`           | `NULL` - Ukuran body respons dalam byte.                                     |
| `duration_ms`          | `INTEGER`          | `NULL` - Durasi permintaan dalam milidetik.                                  |

---

### Tabel: `collections` (Fitur Eksklusif Pengguna Login)

Mengelompokkan beberapa permintaan ke dalam satu koleksi untuk organisasi.

| Nama Kolom   | Tipe Data          | Kendala/Catatan                                                       |
|--------------|--------------------|-----------------------------------------------------------------------|
| `id`         | `SERIAL PRIMARY KEY` | Identifier unik untuk setiap koleksi.                                 |
| `user_id`    | `INTEGER`          | `NOT NULL, REFERENCES users(id) ON DELETE CASCADE` - Pemilik koleksi. |
| `name`       | `VARCHAR(255)`     | `NOT NULL` - Nama koleksi (e.g., "User API").                         |
| `description`| `TEXT`             | `NULL` - Deskripsi opsional untuk koleksi.                            |
| `created_at` | `TIMESTAMPTZ`      | `DEFAULT CURRENT_TIMESTAMP`                                           |
| `updated_at` | `TIMESTAMPTZ`      | `DEFAULT CURRENT_TIMESTAMP`                                           |

---

### Tabel: `environments` (Fitur Eksklusif Pengguna Login)

Menyimpan grup variabel lingkungan yang dapat digunakan kembali.

| Nama Kolom   | Tipe Data          | Kendala/Catatan                                                       |
|--------------|--------------------|-----------------------------------------------------------------------|
| `id`         | `SERIAL PRIMARY KEY` | Identifier unik untuk setiap lingkungan.                              |
| `user_id`    | `INTEGER`          | `NOT NULL, REFERENCES users(id) ON DELETE CASCADE` - Pemilik lingkungan. |
| `name`       | `VARCHAR(255)`     | `NOT NULL` - Nama lingkungan (e.g., "Staging", "Production").         |
| `created_at` | `TIMESTAMPTZ`      | `DEFAULT CURRENT_TIMESTAMP`                                           |
| `updated_at` | `TIMESTAMPTZ`      | `DEFAULT CURRENT_TIMESTAMP`                                           |

---

### Tabel: `environment_variables` (Fitur Eksklusif Pengguna Login)

Menyimpan pasangan kunci-nilai untuk setiap lingkungan.

| Nama Kolom        | Tipe Data          | Kendala/Catatan                                                              |
|-------------------|--------------------|------------------------------------------------------------------------------|
| `id`              | `SERIAL PRIMARY KEY` | Identifier unik.                                                             |
| `environment_id`  | `INTEGER`          | `NOT NULL, REFERENCES environments(id) ON DELETE CASCADE` - Lingkungan induk. |
| `variable_key`    | `VARCHAR(255)`     | `NOT NULL` - Nama variabel.                                                  |
| `encrypted_value` | `TEXT`             | `NOT NULL` - Nilai variabel yang dienkripsi.                                 |
| `created_at`      | `TIMESTAMPTZ`      | `DEFAULT CURRENT_TIMESTAMP`                                                  |
| `updated_at`      | `TIMESTAMPTZ`      | `DEFAULT CURRENT_TIMESTAMP`                                                  |
|                   |                    | `UNIQUE (environment_id, variable_key)` - Kunci harus unik per lingkungan.   |

## 4. Pertimbangan Tambahan

- **Indeks**: Indeks perlu ditambahkan pada kolom *foreign key* (seperti `user_id`, `collection_id`) dan kolom yang sering di-query (seperti `email` pada tabel `users`) untuk meningkatkan performa.
- **Enkripsi**: Nilai pada `environment_variables` harus dienkripsi saat disimpan (at-rest) untuk keamanan.
- **Trigger untuk `updated_at`**: Sebuah fungsi trigger di PostgreSQL sebaiknya dibuat untuk secara otomatis memperbarui kolom `updated_at` setiap kali sebuah baris diubah.