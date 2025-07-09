# Ringkasan Proyek: Suar - Backend Klien HTTP Berbasis Web

## Pendahuluan
Suar adalah klien HTTP berbasis web yang dirancang untuk menyederhanakan proses pembuatan permintaan HTTP. Repositori ini berisi komponen backend dari aplikasi Suar, yang dibangun dengan Go. Tujuan utama backend adalah bertindak sebagai proxy yang kuat dan fleksibel, menangani berbagai jenis permintaan HTTP yang dimulai oleh frontend dan meneruskannya ke tujuan yang dimaksudkan.

## Fungsionalitas Inti
Tanggung jawab utama backend adalah menerima konfigurasi permintaan HTTP dari frontend, mengeksekusi permintaan ini, dan mengembalikan respons. Ini melibatkan:
- **Request Proxying**: Meneruskan permintaan masuk ke API atau layanan eksternal.
- **Dynamic Request Handling**: Mendukung berbagai metode HTTP (GET, POST, PUT, DELETE, dll.), header kustom, parameter kueri, dan body permintaan.
- **Response Handling**: Menangkap dan mengembalikan respons HTTP lengkap, termasuk kode status, header, dan body.
- **Error Management**: Menangani kesalahan jaringan, timeout, dan kesalahan spesifik API dengan baik.

## Tumpukan Teknologi
- **Bahasa**: Go
- **Kerangka Kerja Web**: [Chi](https://go-chi.io/) - Router yang ringan, idiomatik, dan dapat disusun untuk membangun layanan HTTP Go.
- **Manajemen Dependensi**: Go Modules
- **Validasi**: [go-playground/validator](https://github.com/go-playground/validator) - Untuk validasi payload permintaan.

## Struktur Proyek
Proyek ini mengikuti struktur modular yang bersih untuk memastikan pemeliharaan dan skalabilitas. Direktori utama meliputi:
- `cmd/api`: Berisi titik masuk aplikasi utama.
- `internal/handler`: Menampung handler HTTP yang bertanggung jawab untuk memproses permintaan masuk dan berinteraksi dengan layanan.
- `internal/service`: Berisi logika bisnis inti, seperti layanan proxy HTTP.
- `internal/model`: Mendefinisikan struktur data (DTOs, model) yang digunakan di seluruh aplikasi.
- `documentation`: Berisi berbagai dokumentasi proyek, termasuk arsitektur, panduan API, dan instruksi pengaturan.

## Cara Menjalankan Aplikasi (Pengembangan)
1.  **Clone repositori**: `git clone https://github.com/suar-net/suar-be.git`
2.  **Navigasi ke direktori proyek**: `cd suar-be`
3.  **Jalankan aplikasi**: `go run cmd/api/main.go`
    *   Server akan dimulai pada `http://localhost:8080` (atau port yang ditentukan oleh variabel lingkungan `PORT`).

## Endpoint API
Saat ini, endpoint API utama adalah:
-   `POST /api/v1/request`: Digunakan untuk memproxy permintaan HTTP. Body permintaan diharapkan berisi konfigurasi untuk permintaan HTTP keluar.
