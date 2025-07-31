# Panduan Integrasi Database PostgreSQL

Dokumen ini menyediakan panduan teknis langkah demi langkah untuk menghubungkan aplikasi backend Suar dengan database PostgreSQL.

## Langkah 1: Menambahkan Driver PostgreSQL

Pertama, tambahkan driver `pq` ke dalam dependensi proyek Anda menggunakan perintah berikut. Perintah ini akan secara otomatis memperbarui berkas `go.mod` dan `go.sum`.

```bash
go get github.com/lib/pq
```

## Langkah 2: Membuat Paket Konfigurasi

Untuk mengelola konfigurasi aplikasi secara terpusat (termasuk kredensial database), buat sebuah paket `config`.

1.  Buat direktori baru: `internal/config`
2.  Buat berkas baru: `internal/config/config.go`
3.  Isi berkas tersebut dengan kode berikut:

```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config menampung semua konfigurasi aplikasi.
type Config struct {
	Server ServerConfig
	DB     DBConfig
}

// ServerConfig menampung konfigurasi server HTTP.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DBConfig menampung konfigurasi koneksi database.
type DBConfig struct {
	Host    string
	Port    int
	User    string
	Pass    string
	Name    string
	SSLMode string
	DSN     string
}

// Load memuat konfigurasi dari environment variables.
func Load() (*Config, error) {
	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	dbConfig := DBConfig{
		Host:    getEnv("DB_HOST", "localhost"),
		Port:    dbPort,
		User:    getEnv("DB_USER", "postgres"),
		Pass:    getEnv("DB_PASS", "password"),
		Name:    getEnv("DB_NAME", "suar_db"),
		SSLMode: getEnv("DB_SSL_MODE", "disable"),
	}
	dbConfig.DSN = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Pass, dbConfig.Name, dbConfig.SSLMode,
	)

	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		DB: dbConfig,
	}, nil
}

// Helper untuk mendapatkan environment variable atau nilai default.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
```

## Langkah 3: Membuat Modul Koneksi Database

Buat paket terpisah untuk logika koneksi database agar kode tetap bersih.

1.  Buat direktori baru: `internal/database`
2.  Buat berkas baru: `internal/database/postgres.go`
3.  Isi berkas tersebut dengan kode berikut:

```go
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/suar-net/suar-be/internal/config"
)

// ConnectDB membuat dan mengembalikan koneksi ke database PostgreSQL.
func ConnectDB(cfg config.DBConfig) (*sql.DB, error) {
	// Membuka koneksi ke database
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi database: %w", err)
	}

	// Mengatur connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Memverifikasi koneksi dengan ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		db.Close() // Tutup koneksi jika ping gagal
		return nil, fmt.Errorf("gagal memverifikasi koneksi database: %w", err)
	}

	return db, nil
}
```

## Langkah 4: Integrasi di `cmd/api/main.go`

Sekarang, gunakan paket `config` dan `database` di dalam fungsi `main` untuk menginisialisasi koneksi saat aplikasi dimulai.

Ganti seluruh isi `cmd/api/main.go` dengan kode berikut:

```go
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/suar-net/suar-be/internal/config"
	"github.com/suar-net/suar-be/internal/database"
	"github.com/suar-net/suar-be/internal/handler"
	"github.com/suar-net/suar-be/internal/service"
)

func main() {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	// 1. Muat Konfigurasi
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Gagal memuat konfigurasi: %v", err)
	}

	// 2. Hubungkan ke Database
	db, err := database.ConnectDB(cfg.DB)
	if err != nil {
		logger.Fatalf("Gagal terhubung ke database: %v", err)
	}
	defer db.Close()
	logger.Println("Koneksi database berhasil.")

	// Initialize Services
	httpProxyService := service.NewHTTPProxyService()
	// Di masa depan, Anda akan menginisialisasi AuthRepository di sini dengan `db`
	// authRepo := repository.NewPostgresAuthRepository(db)
	// authService, err := service.NewAuthService(authRepo, logger) ...

	// Setup Router
	// Saat ini router belum memerlukan `db`, tapi kita siapkan untuk masa depan
	router := handler.SetupRouter(httpProxyService, db, logger)

	// Konfigurasi Server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Printf("Server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Tidak dapat menjalankan server di port %s: %v", cfg.Server.Port, err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Println("Mematikan server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server shutdown gagal: %v", err)
	}
	logger.Println("Server berhasil dimatikan.")
}
```

## Langkah 5: Membuat Health Check Endpoint

Untuk memverifikasi koneksi database dari luar, tambahkan sebuah *endpoint* `/healthcheck`.

Ganti seluruh isi `internal/handler/router.go` dengan kode berikut:

```go
package handler

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/suar-net/suar-be/internal/service"
)

// SetupRouter menginisialisasi dan mengonfigurasi router HTTP.
// Kita tambahkan `*sql.DB` sebagai parameter untuk di-pass ke handler.
func SetupRouter(
	httpProxyService service.HTTPProxyService,
	db *sql.DB, // Tambahkan parameter DB
	logger *log.Logger,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Initialize handlers
	httpProxyHandler := NewHTTPProxyHandler(httpProxyService, logger)

	r.Route("/api/v1", func(r chi.Router) {
		// Rute yang sudah ada
		r.Mount("/request", httpProxyHandler)

		// Rute baru untuk health check
		r.Get("/healthcheck", healthCheckHandler(db, logger))
	})

	return r
}

// healthCheckHandler membuat http.HandlerFunc untuk memeriksa status koneksi database.
func healthCheckHandler(db *sql.DB, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			logger.Printf("Health check gagal: koneksi database error: %v", err)
			data := map[string]string{
				"status":  "error",
				"message": "database connection failed",
			}
			respondWithError(w, http.StatusServiceUnavailable, data)
			return
		}

		data := map[string]string{
			"status":  "ok",
			"message": "database connection is healthy",
		}
		respondWithJSON(w, http.StatusOK, data)
	}
}
```

## Langkah 6: Konfigurasi Environment Variables

Aplikasi Anda sekarang membaca konfigurasi dari *environment variables*. Anda bisa membuat berkas `.env` di root proyek Anda untuk pengembangan lokal.

**Contoh berkas `.env`:**
```
# Konfigurasi Server
PORT=8080

# Konfigurasi Database PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASS=your_secret_password # Ganti dengan password Anda
DB_NAME=suar_db
DB_SSL_MODE=disable
```

**Penting**: Jangan pernah menyimpan kredensial asli di dalam kode atau di dalam Git. Gunakan *environment variables*.

## Kesimpulan

Setelah mengikuti semua langkah ini, aplikasi Anda akan:
1.  Memiliki driver PostgreSQL.
2.  Memuat konfigurasi dari *environment variables*.
3.  Terhubung ke database PostgreSQL saat startup.
4.  Menyediakan *endpoint* `GET /api/v1/healthcheck` untuk memverifikasi koneksi.

Anda sekarang siap untuk melanjutkan ke tahap implementasi *repository* dan *service* untuk fitur autentikasi.
