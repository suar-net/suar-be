
# Panduan Penggunaan Raw SQL dengan PostgreSQL di Go

Dokumen ini adalah panduan teknis untuk berinteraksi dengan database PostgreSQL menggunakan raw SQL di dalam proyek Go. Panduan ini ditujukan untuk developer yang terbiasa dengan ORM dan ingin beralih ke pendekatan yang lebih fundamental dan terkontrol.

## 1. Filosofi: `database/sql` dan Driver

Go memiliki pendekatan dua lapis untuk interaksi database:

1.  **`database/sql`**: Ini adalah paket di dalam *standard library* Go. Paket ini menyediakan *interface* generik dan abstrak untuk berinteraksi dengan database SQL. Ia tidak tahu cara berkomunikasi dengan PostgreSQL, MySQL, atau database spesifik lainnya. Tugasnya adalah menyediakan API yang konsisten untuk semua operasi database (query, transaksi, dll).
2.  **Database Driver**: Ini adalah "penerjemah" yang mengimplementasikan *interface* dari `database/sql` dan menangani komunikasi spesifik dengan jenis database tertentu. Untuk PostgreSQL, driver modern dan yang paling direkomendasikan adalah **`pgx`**.

Anda akan selalu menggunakan fungsi-fungsi dari paket `database/sql` di dalam kode aplikasi Anda, dan mengimpor driver `pgx` agar `database/sql` tahu cara berbicara dengan PostgreSQL.

## 2. Koneksi ke Database

Koneksi ke database adalah langkah pertama. Koneksi ini idealnya dibuat sekali saat aplikasi pertama kali berjalan dan kemudian digunakan kembali di seluruh aplikasi (konsep ini disebut *connection pool*).

```go
// internal/database/postgres.go

package database

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Driver PostgreSQL (pgx)
)

// ConnectDB menginisialisasi dan mengembalikan koneksi ke database PostgreSQL.
func ConnectDB(dsn string) (*sql.DB, error) {
	// sql.Open tidak langsung membuat koneksi. Ia hanya menyiapkan objek *sql.DB.
	// Koneksi sebenarnya dibuat saat pertama kali dibutuhkan (lazy connection).
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	// Sangat penting untuk mengatur parameter connection pool.
	db.SetMaxOpenConns(25) // Jumlah maksimum koneksi yang terbuka
	db.SetMaxIdleConns(25) // Jumlah maksimum koneksi yang idle
	db.SetConnMaxLifetime(5 * time.Minute) // Waktu maksimum koneksi dapat digunakan kembali

	// Gunakan PingContext untuk memverifikasi bahwa koneksi ke database berhasil dibuat.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
```

**DSN (Data Source Name)** adalah string yang berisi informasi koneksi. Formatnya:
`"postgres://[user]:[password]@[host]:[port]/[dbname]?sslmode=disable"`

## 3. Operasi CRUD dengan Raw SQL

Ini adalah inti dari interaksi database. Kunci utamanya adalah **selalu gunakan placeholders (`$1`, `$2`, dst.) untuk menyisipkan data ke dalam query**. Jangan pernah menggunakan `fmt.Sprintf` atau konkatenasi string untuk membangun query dengan input dari pengguna, karena ini akan membuka celah keamanan **SQL Injection**.

### a. Menjalankan Perintah Tanpa Hasil (INSERT, UPDATE, DELETE)

Gunakan `db.ExecContext()` untuk perintah yang tidak mengembalikan baris data.

**Contoh: INSERT**
```go
func CreateUser(ctx context.Context, db *sql.DB, email string, name string) (int64, error) {
	// Perhatikan penggunaan placeholder $1, $2
	query := "INSERT INTO users (email, name, created_at) VALUES ($1, $2, NOW())"

	// ExecContext mengirim query ke database
	result, err := db.ExecContext(ctx, query, email, name)
	if err != nil {
		return 0, err
	}

	// Anda bisa mendapatkan jumlah baris yang terpengaruh
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
```

### b. Mengambil Satu Baris Data (Single Row Query)

Gunakan `db.QueryRowContext()` untuk query yang Anda harapkan mengembalikan **tepat satu baris**.

**Contoh: SELECT by ID**
```go
type User struct {
	ID        int
	Email     string
	Name      string
	CreatedAt time.Time
}

func GetUserByID(ctx context.Context, db *sql.DB, id int) (*User, error) {
	query := "SELECT id, email, name, created_at FROM users WHERE id = $1"
	
	row := db.QueryRowContext(ctx, query, id)

	var user User
	// .Scan() akan memetakan hasil kolom ke dalam variabel Go.
	// Urutan variabel di .Scan() harus sama persis dengan urutan kolom di SELECT.
	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if err != nil {
		// Ini adalah error yang sangat umum!
		// Terjadi jika query tidak menemukan baris sama sekali.
		// Anda harus menanganinya secara eksplisit.
		if err == sql.ErrNoRows {
			return nil, errors.New("user tidak ditemukan")
		}
		// Error lain yang mungkin terjadi
		return nil, err
	}

	return &user, nil
}
```

### c. Mengambil Banyak Baris Data (Multiple Rows Query)

Gunakan `db.QueryContext()` untuk query yang bisa mengembalikan banyak baris. Pola ini sedikit lebih kompleks.

**Contoh: SELECT All Users**
```go
func GetAllUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	query := "SELECT id, email, name, created_at FROM users ORDER BY created_at DESC"

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	// PENTING! Selalu tutup rows untuk melepaskan koneksi kembali ke pool.
	defer rows.Close()

	var users []User // Slice untuk menampung hasil

	// Lakukan iterasi per baris
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	// Setelah loop selesai, selalu periksa apakah ada error selama iterasi.
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
```

## 4. Transaksi (Transactions)

Transaksi digunakan untuk menjalankan serangkaian perintah sebagai satu unit atomik. Jika salah satu perintah gagal, semua perintah sebelumnya akan dibatalkan (`ROLLBACK`). Jika semua berhasil, perubahan akan disimpan (`COMMIT`).

Ini sangat penting untuk menjaga integritas data.

```go
func TransferBalance(ctx context.Context, db *sql.DB, fromAccountID, toAccountID int, amount float64) error {
	// 1. Mulai transaksi
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// 2. Defer Rollback. Jika terjadi panic atau error, transaksi akan dibatalkan.
	// Jika Commit() berhasil, Rollback() tidak akan berpengaruh.
	defer tx.Rollback()

	// 3. Jalankan perintah di dalam transaksi menggunakan `tx`, bukan `db`
	_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", amount, fromAccountID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", amount, toAccountID)
	if err != nil {
		return err
	}

	// 4. Jika semua perintah berhasil, commit transaksi
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
```

## 5. Praktik Terbaik dan Jebakan Umum

*   **SQL Injection**: Saya ulangi lagi: **JANGAN PERNAH** gunakan `fmt.Sprintf("SELECT * FROM users WHERE id = %s", userID)`. **SELALU** gunakan placeholders (`$1`, `$2`, ...).
*   **Menangani Nilai NULL**: Kolom database bisa bernilai `NULL`. Tipe data standar Go (seperti `string` atau `int`) tidak bisa merepresentasikan `NULL`. Jika Anda mencoba `.Scan()` nilai `NULL` ke `string`, Anda akan mendapat error. Gunakan tipe data khusus dari `database/sql` seperti `sql.NullString`, `sql.NullInt64`, `sql.NullTime`, dll.
    ```go
    var middleName sql.NullString
    err := row.Scan(&firstName, &middleName)
    // ...
    if middleName.Valid {
        // Gunakan middleName.String
    } else {
        // Kolom ini NULL
    }
    ```
*   **`defer rows.Close()`**: Lupa menutup `rows` pada query multi-baris adalah penyebab umum kebocoran koneksi (*connection leak*). Koneksi tidak akan kembali ke *pool* dan aplikasi Anda bisa kehabisan koneksi.
*   **`sql.ErrNoRows`**: Selalu periksa error ini secara spesifik saat menggunakan `QueryRowContext`. Ini bukan error fatal, melainkan kondisi "tidak ditemukan" yang harus ditangani.

Dengan memahami prinsip-prinsip ini, Anda memiliki fondasi yang kuat untuk menggunakan raw SQL secara efektif dan aman di proyek Go Anda.
