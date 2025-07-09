## 1. Panduan Implementasi Otorisasi Berbasis JWT

Dokumen ini menyediakan panduan teknis langkah demi langkah untuk mengimplementasikan otorisasi berbasis JSON Web Token (JWT) di aplikasi backend Suar.

## 2. Persiapan Teknis

Sebelum memulai implementasi, pastikan Anda memiliki hal-hal berikut:

### 2.1. Go Dependencies

Tambahkan dependensi berikut ke proyek Go Anda:

*   **JWT Library**: Untuk membuat dan memverifikasi JWT.
    ```bash
    go get github.com/golang-jwt/jwt/v5
    ```
*   **Bcrypt Library**: Untuk hashing password pengguna.
    ```bash
    go get golang.org/x/crypto/bcrypt
    ```
*   **Database Driver**: Pastikan Anda memiliki driver database yang sesuai (misalnya, `github.com/lib/pq` untuk PostgreSQL, `github.com/go-sql-driver/mysql` untuk MySQL).

### 2.2. Skema Database Pengguna

Buat tabel `users` di database Anda. Minimal, tabel ini harus memiliki kolom berikut:

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Atau INTEGER jika Anda tidak menggunakan UUID
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE, -- Opsional, tapi direkomendasikan
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 2.3. Variabel Lingkungan (Environment Variables)

Definisikan variabel lingkungan berikut untuk konfigurasi keamanan dan JWT:

*   `JWT_SECRET_KEY`: Kunci rahasia yang kuat dan acak (misalnya, 32-64 karakter) untuk menandatangani JWT. **Jaga kerahasiaan kunci ini.**
*   `ACCESS_TOKEN_EXPIRATION_MINUTES`: Durasi kedaluwarsa access token dalam menit (misalnya, `15` atau `30`).

## 3. Langkah-langkah Implementasi

### 3.1. Definisi Model (`internal/model/user.go`)

Buat atau modifikasi file `internal/model/user.go` untuk mendefinisikan struktur data pengguna dan DTOs untuk pendaftaran/login.

```go
package model

import "time"

// User represents the user model in the database.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Exclude from JSON output
	Email        string    `json:"email,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserRegisterRequest represents the DTO for user registration.
type UserRegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
	Email    string `json:"email" validate:"omitempty,email"`
}

// UserLoginRequest represents the DTO for user login.
type UserLoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// UserLoginResponse represents the DTO for user login response.
type UserLoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"` // Seconds until expiration
}

// Claims represents the JWT claims.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}
```

### 3.2. Implementasi Service Autentikasi (`internal/service/auth_service.go`)

Buat file `internal/service/auth_service.go` yang akan berisi logika bisnis untuk pendaftaran, login, dan manajemen JWT.

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"suar-be/internal/model" // Sesuaikan dengan path model Anda
)

// Custom errors for authentication service
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists = errors.New("username or email already exists")
	ErrTokenInvalid    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")
)

// AuthRepository defines the interface for user data persistence.
type AuthRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	// Anda bisa menambahkan GetUserByID jika diperlukan
}

// AuthService defines the interface for authentication operations.
type AuthService interface {
	Register(ctx context.Context, req *model.UserRegisterRequest) (*model.User, error)
	Login(ctx context.Context, req *model.UserLoginRequest) (*model.UserLoginResponse, error)
	GenerateAccessToken(userID, username string) (string, int64, error)
	ValidateAccessToken(tokenString string) (*model.Claims, error)
}

// authService implements AuthService.
type authService struct {
	repo AuthRepository
	jwtSecretKey []byte
	accessTokenExpirationMinutes int
	logger *log.Logger
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService(repo AuthRepository, logger *log.Logger) (AuthService, error) {
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET_KEY environment variable not set")
	}

	expMinutesStr := os.Getenv("ACCESS_TOKEN_EXPIRATION_MINUTES")
	expMinutes, err := strconv.Atoi(expMinutesStr)
	if err != nil || expMinutes <= 0 {
		logger.Printf("Warning: ACCESS_TOKEN_EXPIRATION_MINUTES not set or invalid, using default 15 minutes: %v", err)
		expMinutes = 15 // Default to 15 minutes
	}

	return &authService{
		repo: repo,
		jwtSecretKey: []byte(jwtSecret),
		accessTokenExpirationMinutes: expMinutes,
		logger: logger,
	}, nil
}

// Register handles user registration.
func (s *authService) Register(ctx context.Context, req *model.UserRegisterRequest) (*model.User, error) {
	// Check if username or email already exists
	existingUser, err := s.repo.GetUserByUsername(ctx, req.Username)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}
	// You might want to add a check for email existence too if email is unique

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		ID:           generateUUID(), // Implement a UUID generator or use database default
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Email:        req.Email,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login handles user login and generates an access token.
func (s *authService) Login(ctx context.Context, req *model.UserLoginRequest) (*model.UserLoginResponse, error) {
	user, err := s.repo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, expiresIn, err := s.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &model.UserLoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}, nil
}

// GenerateAccessToken creates a new JWT access token.
func (s *authService) GenerateAccessToken(userID, username string) (string, int64, error) {
	expirationTime := time.Now().Add(time.Duration(s.accessTokenExpirationMinutes) * time.Minute)
	expiresIn := expirationTime.Unix() - time.Now().Unix() // Seconds until expiration

	claims := &model.Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "suar-be", // Your application name
			Subject:   userID,
			Audience:  []string{"users"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecretKey)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresIn, nil
}

// ValidateAccessToken parses and validates a JWT access token.
func (s *authService) ValidateAccessToken(tokenString string) (*model.Claims, error) {
	claims := &model.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// Placeholder for UUID generation. In a real app, use a library like github.com/google/uuid
func generateUUID() string {
	return fmt.Sprintf("user-%d", time.Now().UnixNano())
}

// Implement AuthRepository (e.g., in internal/repository/user_repository.go)
// This is a placeholder for demonstration. You'll need a real database implementation.
type inMemoryAuthRepo struct {
	users map[string]*model.User
}

func NewInMemoryAuthRepo() AuthRepository {
	return &inMemoryAuthRepo{
		users: make(map[string]*model.User),
	}
}

func (r *inMemoryAuthRepo) CreateUser(ctx context.Context, user *model.User) error {
	if _, exists := r.users[user.Username]; exists {
		return ErrUserAlreadyExists
	}
	r.users[user.Username] = user
	return nil
}

func (r *inMemoryAuthRepo) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	user, exists := r.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}
```

**Catatan**: Anda perlu mengimplementasikan `AuthRepository` yang sebenarnya untuk berinteraksi dengan database Anda (misalnya, menggunakan `database/sql` atau ORM seperti GORM). Contoh `inMemoryAuthRepo` di atas hanya untuk demonstrasi.

### 3.3. Implementasi Handler Autentikasi (`internal/handler/auth_handler.go`)

Buat file `internal/handler/auth_handler.go` untuk menangani permintaan HTTP terkait autentikasi.

```go
package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"suar-be/internal/model" // Sesuaikan dengan path model Anda
	"suar-be/internal/service" // Sesuaikan dengan path service Anda
)

// AuthHandler defines the HTTP handlers for authentication.
type AuthHandler struct {
	authService service.AuthService
	logger      *log.Logger
}

// NewAuthHandler creates a new instance of AuthHandler.
func NewAuthHandler(authService service.AuthService, logger *log.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register handles user registration requests.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.UserRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Basic validation (more comprehensive validation should be done with go-playground/validator)
	if req.Username == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	user, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			respondWithError(w, http.StatusConflict, err.Error())
			return
		}
		h.logger.Printf("Error registering user: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to register user")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]string{"message": "User registered successfully", "user_id": user.ID})
}

// Login handles user login requests.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Basic validation
	if req.Username == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	resp, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			respondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		h.logger.Printf("Error logging in user: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to login user")
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

// Helper functions (respondWithError, respondWithJSON) should be in a common utility file
// or directly in the handler package if they are only used here.
// For consistency, you might already have them in http_proxy_handler.go or validator.go.
// If not, define them here or create a new `util.go` file in `internal/handler`.

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
```

### 3.4. Middleware Autentikasi (`internal/handler/auth_middleware.go`)

Buat file `internal/handler/auth_middleware.go` untuk middleware yang akan memverifikasi JWT pada setiap permintaan terautentikasi.

```go
package handler

import (
	"context"
	"log"
	"net/http"
	"strings"

	"suar-be/internal/service" // Sesuaikan dengan path service Anda
)

// ContextKey represents the key for storing user information in context.
type ContextKey string

const (
	// UserContextKey is the key used to store authenticated user information in the request context.
	UserContextKey ContextKey = "user"
)

// AuthMiddleware provides JWT authentication middleware.
type AuthMiddleware struct {
	authService service.AuthService
	logger      *log.Logger
}

// NewAuthMiddleware creates a new instance of AuthMiddleware.
func NewAuthMiddleware(authService service.AuthService, logger *log.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		logger:      logger,
	}
}

// Authenticate is a middleware that validates the JWT from the Authorization header.
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			respondWithError(w, http.StatusUnauthorized, "Authorization header must be in Bearer format")
			return
		}

		tokenString := parts[1]
		claims, err := m.authService.ValidateAccessToken(tokenString)
		if err != nil {
			if err == service.ErrTokenExpired {
				respondWithError(w, http.StatusUnauthorized, "Token expired")
				return
			}
			m.logger.Printf("Invalid token: %v", err)
			respondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		// Store user information in the request context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext extracts user claims from the request context.
func GetUserFromContext(ctx context.Context) (*service.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*service.Claims)
	return claims, ok
}
```

### 3.5. Integrasi Router (`internal/handler/router.go` dan `cmd/api/main.go`)

Modifikasi `internal/handler/router.go` untuk menambahkan rute autentikasi dan menerapkan middleware.

```go
package handler

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"suar-be/internal/service" // Sesuaikan dengan path service Anda
)

// SetupRouter initializes and configures the HTTP router.
func SetupRouter(
	httpProxyService service.HTTPProxyService,
	authService service.AuthService, // Tambahkan AuthService
	logger *log.Logger,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // WARNING: For production, restrict to specific frontend domains
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// Initialize handlers
	httpProxyHandler := NewHTTPProxyHandler(httpProxyService, logger)
	authHandler := NewAuthHandler(authService, logger) // Inisialisasi AuthHandler
	authMiddleware := NewAuthMiddleware(authService, logger) // Inisialisasi AuthMiddleware

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes (no authentication required)
		r.Post("/request", httpProxyHandler.ServeHTTP) // Existing proxy endpoint

		// Authentication routes
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// Protected routes (authentication required)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate) // Apply authentication middleware
			// Example of a protected route:
			r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
				claims, ok := GetUserFromContext(r.Context())
				if !ok {
					respondWithError(w, http.StatusInternalServerError, "User context not found")
					return
				}
				respondWithJSON(w, http.StatusOK, map[string]string{
					"message":  "You accessed a protected route!",
					"user_id":  claims.UserID,
					"username": claims.Username,
				})
			})
			// Add your request history endpoints here, e.g.:
			// r.Get("/history", historyHandler.GetHistory)
			// r.Post("/history", historyHandler.SaveHistory)
		})
	})

	return r
}
```

Dan modifikasi `cmd/api/main.go` untuk menginisialisasi `AuthService` dan `AuthRepository`.

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"suar-be/internal/handler"
	"suar-be/internal/service"
)

func main() {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Initialize HTTP Proxy Service
	httpProxyService := service.NewHTTPProxyService()

	// Initialize Auth Repository (using in-memory for demonstration, replace with real DB repo)
	authRepo := service.NewInMemoryAuthRepo() // Replace with your actual database repository

	// Initialize Auth Service
	authService, err := service.NewAuthService(authRepo, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize auth service: %v", err)
	}

	// Setup Router
	router := handler.SetupRouter(httpProxyService, authService, logger) // Pass authService

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Could not listen on port %s: %v", port, err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server shutdown failed: %v", err)
	}
	logger.Println("Server gracefully stopped.")
}
```

## 4. Pertimbangan Keamanan Tambahan

*   **HTTPS**: Selalu gunakan HTTPS di lingkungan produksi untuk melindungi JWT saat transit.
*   **Token Expiration**: Access token harus memiliki masa berlaku yang singkat (misalnya, 15-30 menit) untuk membatasi jendela waktu di mana token yang dicuri dapat digunakan.
*   **Refresh Tokens (Opsional)**: Untuk pengalaman pengguna yang lebih baik, pertimbangkan untuk mengimplementasikan refresh token. Refresh token adalah token jangka panjang yang digunakan untuk mendapatkan access token baru tanpa perlu login ulang. Ini memerlukan penyimpanan refresh token di database dan validasi tambahan.
*   **Penyimpanan Token di Frontend**: Instruksikan frontend untuk menyimpan access token dengan aman. `HttpOnly cookie` adalah pilihan yang lebih aman untuk mencegah serangan XSS, meskipun `localStorage` juga sering digunakan dengan pertimbangan keamanan yang cermat.
*   **Rate Limiting**: Terapkan rate limiting pada endpoint login dan pendaftaran untuk mencegah serangan brute-force.
*   **Validasi Input**: Gunakan `go-playground/validator` secara ekstensif pada DTOs `UserRegisterRequest` dan `UserLoginRequest` untuk memastikan input yang valid dan mencegah serangan injeksi atau data yang tidak valid.
*   **Error Handling**: Pastikan pesan error yang dikembalikan ke klien tidak terlalu detail dan tidak membocorkan informasi sensitif internal.

Dengan panduan ini, Anda memiliki kerangka kerja yang solid untuk mengimplementasikan otorisasi berbasis JWT di proyek Suar.
