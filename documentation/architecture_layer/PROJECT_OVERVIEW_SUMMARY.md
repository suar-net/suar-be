# Ringkasan Proyek Suar Backend untuk Asisten LLM

Dokumen ini menyediakan ringkasan teknis tingkat tinggi dari proyek Suar Backend. Tujuannya adalah untuk memberikan pemahaman yang cepat dan komprehensif tentang arsitektur, fungsionalitas, dan komponen kunci kepada Large Language Model (LLM) yang membantu pengembangan.

## 1. Ringkasan Proyek

**Suar Backend** adalah sebuah layanan **proxy HTTP** yang ditulis dalam **Go (Golang)**, yang dirancang untuk menjadi platform pengujian dan manajemen API. Fungsionalitas intinya adalah menerima permintaan HTTP, menjalankannya atas nama klien, dan menyimpan riwayat permintaan untuk pengguna yang terautentikasi. Proyek ini dibangun dengan arsitektur berlapis (layered architecture) yang bersih untuk memastikan pemisahan tanggung jawab, kemudahan perawatan, dan testability.

## 2. Fungsionalitas Inti & Direncanakan

-   **Proxy HTTP**: Klien mengirim permintaan `POST` ke endpoint `/api/v1/request` dengan body JSON yang mendefinisikan `method`, `url`, `headers`, dan `body` dari permintaan yang ingin dijalankan.
-   **Respons Terstruktur**: Layanan mengembalikan respons terstruktur yang berisi `status_code`, `headers`, `body`, `duration`, dan detail lainnya dari server target.
-   **Pemeriksaan Kesehatan (Health Check)**: Menyediakan endpoint `GET /api/v1/healthcheck` untuk memverifikasi status layanan dan konektivitas database.
-   **Manajemen Pengguna (Direncanakan)**: Fitur untuk registrasi dan login pengguna.
-   **Riwayat Permintaan (Direncanakan)**: Pengguna yang login dapat melihat riwayat permintaan yang telah mereka buat.

## 3. Arsitektur Aplikasi (Layered Architecture)

Proyek ini mengikuti arsitektur berlapis yang jelas untuk memisahkan tanggung jawab:

-   **Handler Layer (`internal/handler`)**: Bertanggung jawab untuk semua interaksi HTTP. Menerima permintaan, mem-parsing JSON, melakukan validasi input, dan memanggil Service Layer. Tidak berisi logika bisnis.
-   **Service Layer (`internal/service`)**: Berisi logika bisnis inti. Menerima data dari Handler, melakukan orkestrasi, dan memanggil Repository Layer untuk mengakses data. Lapisan ini tidak berinteraksi langsung dengan database.
-   **Repository Layer (`internal/repository`)**: Berfungsi sebagai jembatan antara Service Layer dan database. Mengabstraksikan semua kueri SQL. Ini adalah satu-satunya lapisan yang boleh berinteraksi langsung dengan database.
-   **Model Layer (`internal/model`)**: Mendefinisikan struct data Go, baik untuk DTO (Data Transfer Object) yang digunakan antar lapisan maupun untuk model domain yang merepresentasikan entitas database.
-   **Database Layer (`internal/database`)**: Mengelola koneksi ke database dan mengeksekusi file migrasi skema.
-   **Config Layer (`internal/config`)**: Bertanggung jawab untuk memuat dan menyediakan konfigurasi aplikasi dari environment variables.

## 4. Struktur Direktori Kunci

-   `cmd/api/main.go`: Titik masuk (entrypoint) aplikasi. Menginisialisasi semua komponen (logger, config, DB, repository, services, handlers) dan memulai server HTTP.
-   `internal/handler/`: Kode yang berhubungan dengan penanganan HTTP.
-   `internal/service/`: Logika bisnis aplikasi.
-   `internal/repository/`: Implementasi kueri database (SQL).
-   `internal/database/`: Koneksi database dan file migrasi (`migrations/`).
-   `internal/model/`: Definisi struct Go (`User`, `RequestHistory`, `DTORequest`, dll.).
-   `internal/config/`: Manajemen konfigurasi.
-   `documentation/`: Semua dokumentasi proyek.

## 5. Alur Data (Contoh: Menyimpan Riwayat setelah Request)

1.  Permintaan masuk ke `main.go` dan diarahkan oleh `chi.Router` ke `HTTPProxyHandler`.
2.  `HTTPProxyHandler` mem-parsing DTO dan memanggil `HTTPProxyService.ProcessRequest()`.
3.  `HTTPProxyService` mengeksekusi permintaan ke server eksternal.
4.  Setelah mendapatkan respons, `HTTPProxyService` memanggil method di service lain, misalnya `HistoryService.CreateEntry(ctx, historyData)`.
5.  `HistoryService` memanggil `Repository.RequestHistory().Create(ctx, historyModel)`.
6.  `RequestHistoryRepository` mengeksekusi kueri `INSERT` SQL ke tabel `request_history`.
7.  Hasil dikembalikan ke atas melalui rantai panggilan, dan `HTTPProxyHandler` mengirim respons akhir ke klien.

## 6. Fitur Keamanan Utama

-   **Perlindungan SSRF (Server-Side Request Forgery)**: Memblokir permintaan ke alamat IP privat.
-   **Filter Header**: Menghapus header sensitif (`Authorization`, `Cookie`) dari permintaan keluar.
-   **Pembatasan Ukuran Body Respons**: Mencegah serangan DoS melalui konsumsi memori.
-   **Manajemen Timeout**: Mencegah sumber daya server tergantung pada koneksi yang lambat.

## 7. Dependensi Utama

-   `github.com/go-chi/chi/v5`: Router HTTP.
-   `github.com/jackc/pgx/v5/stdlib`: Driver PostgreSQL.
-   `github.com/go-playground/validator/v10`: Validasi struct.
-   `github.com/golang-migrate/migrate/v4`: Untuk migrasi skema database.
