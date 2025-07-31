# Panduan Implementasi Repository Layer

Dokumen ini menyediakan panduan langkah-demi-langkah untuk membuat **Repository Layer** (Lapisan Repositori) di proyek Suar Backend. Lapisan ini sangat penting untuk memisahkan logika bisnis dari logika akses data.

## 1. Apa itu Repository Layer?

Repository Layer adalah sebuah abstraksi yang berada di antara *Service Layer* dan *Database*. Tujuannya adalah untuk mengisolasi semua logika yang berhubungan langsung dengan database (kueri SQL) ke dalam satu tempat.

**Service Layer TIDAK boleh menulis SQL secara langsung.** Sebaliknya, ia akan memanggil method-method yang disediakan oleh Repository Layer, seperti `CreateUser()` atau `GetHistoryByUserID()`.

## 2. Keuntungan Menggunakan Repository Layer

-   **Pemisahan Tanggung Jawab (Separation of Concerns)**: Service layer fokus pada logika bisnis, repository layer fokus pada logika data.
-   **Testability**: Anda dapat dengan mudah membuat *mock* (tiruan) dari repository untuk menguji service layer tanpa perlu terhubung ke database sungguhan.
-   **Kemudahan Perawatan**: Jika Anda perlu mengubah cara data disimpan (misalnya, mengoptimalkan kueri SQL atau bahkan mengganti database), Anda hanya perlu mengubahnya di satu tempat (repository), tanpa menyentuh service layer.
-   **Kode Lebih Bersih**: Service layer menjadi lebih mudah dibaca karena tidak dicampuri oleh kode-kode SQL yang panjang.

## 3. Langkah-langkah Implementasi

### Langkah 1: Buat Direktori dan File yang Dibutuhkan

Buat direktori dan file-file berikut di dalam proyek Anda:

```
internal/
└── repository/
    ├── repository.go                 # Mendefinisikan interface dan struct utama
    ├── user_repository.go            # Implementasi untuk tabel 'users'
    └── request_history_repository.go # Implementasi untuk tabel 'request_history'
```

### Langkah 2: Definisikan Interface di `repository.go`

File ini akan menjadi pusat dari lapisan repositori kita. Ia mendefinisikan *interface* (kontrak) untuk setiap repositori dan sebuah *struct* utama untuk injeksi dependensi.

**Salin kode berikut ke `internal/repository/repository.go`:**

```go
package repository

import (
	"context"
	"database/sql"

	"github.com/suar-net/suar-be/internal/model" // Pastikan model diimpor
)

// IRepository adalah interface utama yang menggabungkan semua interface repository.
// Ini berguna untuk dependency injection dan mocking.	ype IRepository interface {
	User() IUserRepository
	RequestHistory() IRequestHistoryRepository
}

// Repository adalah struct yang menampung semua implementasi repositori.	ype Repository struct {
	user           IUserRepository
	requestHistory IRequestHistoryRepository
}

// NewRepository adalah constructor untuk membuat instance Repository baru.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		user:           NewUserRepository(db),
		requestHistory: NewRequestHistoryRepository(db),
	}
}

// User mengembalikan implementasi IUserRepository.
func (r *Repository) User() IUserRepository {
	return r.user
}

// RequestHistory mengembalikan implementasi IRequestHistoryRepository.
func (r *Repository) RequestHistory() IRequestHistoryRepository {
	return r.requestHistory
}

// IUserRepository mendefinisikan interface untuk operasi data pada pengguna.	ype IUserRepository interface {
	Create(ctx context.Context, user *model.User) (int, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

// IRequestHistoryRepository mendefinisikan interface untuk operasi data pada riwayat permintaan.	ype IRequestHistoryRepository interface {
	Create(ctx context.Context, history *model.RequestHistory) error
	GetByUserID(ctx context.Context, userID int) ([]*model.RequestHistory, error)
}
```

### Langkah 3: Implementasikan `user_repository.go`

File ini berisi implementasi konkret dari `IUserRepository`.

**Salin kode berikut ke `internal/repository/user_repository.go`:**

```go
package repository

import (
	"context"
	"database/sql"

	"github.com/suar-net/suar-be/internal/model"
)

// userRepository adalah implementasi dari IUserRepository.
type userRepository struct {
	db *sql.DB
}

// NewUserRepository adalah constructor untuk userRepository.
func NewUserRepository(db *sql.DB) IUserRepository {
	return &userRepository{db: db}
}

// Create menyisipkan pengguna baru ke dalam database dan mengembalikan ID-nya.
func (r *userRepository) Create(ctx context.Context, user *model.User) (int, error) {
	query := `
		INSERT INTO users (full_name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id`

	var userID int
	err := r.db.QueryRowContext(ctx, query, user.FullName, user.Email, user.PasswordHash).Scan(&userID)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

// GetByEmail mengambil pengguna dari database berdasarkan alamat email.
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, full_name, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1`

	var user model.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Pengguna tidak ditemukan, bukan error
		}
		return nil, err
	}

	return &user, nil
}
```

### Langkah 4: Implementasikan `request_history_repository.go`

File ini berisi implementasi konkret dari `IRequestHistoryRepository`.

**Salin kode berikut ke `internal/repository/request_history_repository.go`:**

```go
package repository

import (
	"context"
	"database/sql"

	"github.com/suar-net/suar-be/internal/model"
)

// requestHistoryRepository adalah implementasi dari IRequestHistoryRepository.
type requestHistoryRepository struct {
	db *sql.DB
}

// NewRequestHistoryRepository adalah constructor untuk requestHistoryRepository.
func NewRequestHistoryRepository(db *sql.DB) IRequestHistoryRepository {
	return &requestHistoryRepository{db: db}
}

// Create menyisipkan catatan riwayat baru ke dalam database.
func (r *requestHistoryRepository) Create(ctx context.Context, history *model.RequestHistory) error {
	query := `
		INSERT INTO request_history (user_id, request_method, request_url, request_headers, request_body, response_status_code, response_headers, response_body, response_size, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		history.UserID,
		history.RequestMethod,
		history.RequestURL,
		history.RequestHeaders,
		history.RequestBody,
		history.ResponseStatusCode,
		history.ResponseHeaders,
		history.ResponseBody,
		history.ResponseSize,
		history.DurationMs,
	)

	return err
}

// GetByUserID mengambil semua riwayat permintaan untuk pengguna tertentu.
func (r *requestHistoryRepository) GetByUserID(ctx context.Context, userID int) ([]*model.RequestHistory, error) {
	query := `
		SELECT id, user_id, executed_at, request_method, request_url, request_headers, request_body, response_status_code, response_headers, response_body, response_size, duration_ms
		FROM request_history
		WHERE user_id = $1
		ORDER BY executed_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []*model.RequestHistory
	for rows.Next() {
		var history model.RequestHistory
		if err := rows.Scan(
			&history.ID,
			&history.UserID,
			&history.ExecutedAt,
			&history.RequestMethod,
			&history.RequestURL,
			&history.RequestHeaders,
			&history.RequestBody,
			&history.ResponseStatusCode,
			&history.ResponseHeaders,
			&history.ResponseBody,
			&history.ResponseSize,
			&history.DurationMs,
		); err != nil {
			return nil, err
		}
		histories = append(histories, &history)
	}

	return histories, nil
}
```

### Langkah 5: Definisikan Model

Pastikan Anda memiliki struct `User` dan `RequestHistory` di dalam paket `internal/model`. Anda mungkin perlu membuat file baru atau memodifikasi yang sudah ada.

**Contoh `internal/model/domain.go`:**

```go
package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           int       `json:"id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Jangan pernah kirim hash ke klien
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RequestHistory struct {
	ID                 int             `json:"id"`
	UserID             *int            `json:"user_id"` // Pointer agar bisa NULL
	ExecutedAt         time.Time       `json:"executed_at"`
	RequestMethod      string          `json:"request_method"`
	RequestURL         string          `json:"request_url"`
	RequestHeaders     json.RawMessage `json:"request_headers"`
	RequestBody        *string         `json:"request_body"`
	ResponseStatusCode *int            `json:"response_status_code"`
	ResponseHeaders    json.RawMessage `json:"response_headers"`
	ResponseBody       *string         `json:<em>"response_body"`
	ResponseSize       *int64          `json:"response_size"`
	DurationMs         *int            `json:"duration_ms"`
}
```

### Langkah 6: Integrasi dengan `main.go`

Terakhir, inisialisasi repositori di `main.go` dan siapkan untuk di-inject ke dalam service layer Anda (yang akan menjadi langkah selanjutnya).

```go
// di dalam func main() di cmd/api/main.go

// ... setelah koneksi db

// Inisialisasi Repository Layer
repo := repository.NewRepository(db)
logger.Println("Repository layer initialized")

// Selanjutnya, Anda akan meng-inject `repo` ini ke dalam service Anda, contoh:
// authService := service.NewAuthService(repo, logger)
// historyService := service.NewHistoryService(repo, logger)

// Dan kemudian service di-inject ke handler:
// authHandler := handler.NewAuthHandler(authService, logger)

// ... dan seterusnya
```

Dengan mengikuti panduan ini, Anda akan memiliki lapisan repositori yang kuat dan terstruktur dengan baik, yang akan membuat sisa pengembangan proyek menjadi jauh lebih mudah dan bersih.
