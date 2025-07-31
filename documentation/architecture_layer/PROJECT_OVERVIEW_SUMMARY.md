# Ringkasan Proyek Suar Backend untuk Asisten LLM

Dokumen ini menyediakan ringkasan teknis tingkat tinggi dari proyek Suar Backend. Tujuannya adalah untuk memberikan pemahaman yang cepat dan komprehensif tentang arsitektur, fungsionalitas, dan komponen kunci kepada Large Language Model (LLM) yang membantu pengembangan.

## 1. Ringkasan Proyek

**Suar Backend** adalah sebuah layanan **proxy HTTP** yang ditulis dalam **Go (Golang)**. Fungsionalitas intinya adalah menerima deskripsi permintaan HTTP dalam format JSON, menjalankannya atas nama klien, dan mengembalikan respons dari server target, juga dalam format JSON. Proyek ini dibangun dengan arsitektur berlapis (layered architecture) yang bersih untuk memastikan pemisahan tanggung jawab, kemudahan perawatan, dan testability.

## 2. Fungsionalitas Inti

-   **Proxy HTTP**: Klien mengirim permintaan `POST` ke endpoint `/api/v1/request` dengan body JSON yang mendefinisikan `method`, `url`, `headers`, dan `body` dari permintaan yang ingin dijalankan.
-   **Respons Terstruktur**: Layanan mengembalikan respons terstruktur yang berisi `status_code`, `headers`, `body`, `duration`, dan detail lainnya dari server target.
-   **Pemeriksaan Kesehatan (Health Check)**: Menyediakan endpoint `GET /api/v1/healthcheck` untuk memverifikasi status layanan dan konektivitasnya ke database.

## 3. Arsitektur Aplikasi (Layered Architecture)

Proyek ini mengikuti arsitektur berlapis yang jelas:

-   **Handler Layer (`internal/handler`)**: Bertanggung jawab untuk semua hal yang berkaitan dengan HTTP. Menerima permintaan, mem-parsing JSON, melakukan validasi input awal, dan memanggil Service Layer. Lapisan ini tidak berisi logika bisnis.
-   **Service Layer (`internal/service`)**: Berisi logika bisnis inti. Menerima DTO dari Handler, melakukan validasi bisnis yang kompleks (seperti perlindungan SSRF), mengeksekusi permintaan ke server eksternal, dan mengembalikan hasilnya.
-   **Model/DTO Layer (`internal/model`)**: Mendefinisikan objek transfer data (`DTORequest`, `DTOResponse`) yang digunakan untuk komunikasi antar lapisan.
-   **Database Layer (`internal/database`)**: Mengabstraksi koneksi dan interaksi dengan database. Saat ini digunakan oleh `HealthHandler` untuk memeriksa konektivitas.
-   **Config Layer (`internal/config`)**: Bertanggung jawab untuk memuat dan menyediakan konfigurasi aplikasi dari environment variables (termasuk yang dari file `.env`).

## 4. Struktur Direktori Kunci

-   `cmd/api/main.go`: Titik masuk (entrypoint) aplikasi. Menginisialisasi semua komponen (logger, config, DB, services, handlers) dan memulai server HTTP.
-   `internal/handler/`: Berisi semua kode yang berhubungan dengan penanganan HTTP.
    -   `router.go`: Mengkonfigurasi rute HTTP menggunakan `chi` dan me-mount semua handler.
    -   `http_proxy_handler.go`: Handler untuk fungsionalitas proxy utama.
    -   `health_handler.go`: Handler untuk endpoint health check.
    -   `response.go`: Fungsi utilitas untuk membuat respons JSON yang konsisten (`respondWithJson`, `respondWithError`).
    -   `validator.go`: Logika untuk validasi DTO menggunakan `go-playground/validator`.
-   `internal/service/`: Berisi logika bisnis.
    -   `http_proxy_service.go`: Implementasi inti dari logika proxy, termasuk keamanan.
    -   `errors.go`: Mendefinisikan error kustom (`ErrInvalidInput`, `ErrRequestTimeout`).
-   `internal/database/`: Mengelola koneksi database.
    -   `postgres.go`: Menginisialisasi koneksi ke PostgreSQL menggunakan driver `pgx`.
-   `internal/model/api.go`: Mendefinisikan struct `DTORequest` dan `DTOResponse`.
-   `internal/config/config.go`: Mendefinisikan struct konfigurasi dan memuatnya.
-   `documentation/`: Berisi semua dokumentasi proyek.

## 5. Alur Data (Contoh: `POST /api/v1/request`)

1.  Permintaan masuk ke `main.go` dan diarahkan oleh `chi.Router` di `router.go`.
2.  `HTTPProxyHandler.ServeHTTP` di `http_proxy_handler.go` menerima permintaan.
3.  Body JSON di-decode menjadi struct `model.DTORequest`.
4.  Struct divalidasi menggunakan `validator`.
5.  `HTTPProxyHandler` memanggil `HTTPProxyService.ProcessRequest()` dengan DTO tersebut.
6.  `HTTPProxyService` melakukan validasi keamanan (terutama SSRF), membuat permintaan HTTP keluar, dan mengeksekusinya.
7.  Respons dari server target dibaca dan dikonversi menjadi `model.DTOResponse`.
8.  `DTOResponse` dikembalikan ke `HTTPProxyHandler`.
9.  `HTTPProxyHandler` menggunakan `respondWithJson` dari `response.go` untuk mengirim respons akhir ke klien.

## 6. Fitur Keamanan Utama

-   **Perlindungan SSRF (Server-Side Request Forgery)**: Service layer melakukan DNS lookup pada URL target dan memblokir permintaan ke alamat IP privat (RFC1918) atau loopback.
-   **Filter Header**: Header sensitif seperti `Authorization` dan `Cookie` dihapus dari permintaan keluar untuk mencegah kebocoran kredensial.
-   **Pembatasan Ukuran Body Respons**: Ukuran body respons dari server target dibatasi untuk mencegah serangan DoS melalui konsumsi memori yang berlebihan.
-   **Timeout**: Permintaan keluar memiliki timeout yang dapat dikonfigurasi untuk mencegah sumber daya server tergantung pada koneksi yang lambat.

## 7. Dependensi Utama

-   `github.com/go-chi/chi/v5`: Router HTTP yang ringan dan cepat.
-   `github.com/jackc/pgx/v5/stdlib`: Driver PostgreSQL modern dan berkinerja tinggi.
-   `github.com/go-playground/validator/v10`: Untuk validasi struct DTO.
-   `github.com/joho/godotenv`: Untuk memuat environment variables dari file `.env` selama development.