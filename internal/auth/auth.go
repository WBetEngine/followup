package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"followup/internal/config"
	"followup/internal/models"

	"golang.org/x/crypto/bcrypt" // Import bcrypt

	"github.com/golang-jwt/jwt/v5"
)

var (
	// JWT secret key (DIHAPUS - akan menggunakan dari config)
	// jwtKey = []byte("rahasia_jwt_key_ganti_dengan_environment_variable")

	// Error definitions
	ErrInvalidCredentials = errors.New("username atau password tidak valid")
	ErrInvalidToken       = errors.New("token tidak valid")
	ErrExpiredToken       = errors.New("token sudah kadaluarsa")
)

// UserClaims struktur untuk JWT claims
type UserClaims struct {
	UserID   int             `json:"user_id"`
	Username string          `json:"username"`
	Role     models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// contextKey adalah tipe yang tidak diekspor untuk kunci konteks yang didefinisikan dalam paket ini.
// Menggunakan tipe khusus untuk kunci konteks adalah praktik yang baik untuk menghindari tabrakan kunci.
type contextKey string

// UserClaimsKey adalah kunci untuk UserClaims dalam sebuah context.Context.
const UserClaimsKey contextKey = "userClaims"

// GetClaimsFromContext mengambil UserClaims dari konteks.
// Mengembalikan nil jika tidak ditemukan atau jika tipenya salah.
func GetClaimsFromContext(ctx context.Context) *UserClaims {
	if claims, ok := ctx.Value(UserClaimsKey).(*UserClaims); ok {
		return claims
	}
	log.Println("[AUTH] UserClaimsKey not found in context or type assertion failed")
	return nil
}

// Authenticate memeriksa username dan password
func Authenticate(username, password string) (bool, error) {
	// TODO: Implementasi pengecekan dengan database
	// Untuk contoh, kita gunakan hardcoded values
	if username == "admin" && password == "admin123" {
		return true, nil
	}

	return false, ErrInvalidCredentials
}

// GenerateToken menghasilkan token JWT untuk user yang berhasil login
func GenerateToken(userID int, username string, role models.UserRole) (string, error) {
	cfg := config.GetConfig() // Ambil konfigurasi
	appJwtKey := []byte(cfg.Auth.JWTSecret)
	if len(appJwtKey) == 0 {
		log.Println("[AUTH] CRITICAL: JWTSecret is empty. Using a default insecure key. CONFIGURE JWT_SECRET ENV VAR!")
		// Fallback ke key default yang tidak aman JIKA konfigurasi kosong,
		// ini seharusnya tidak terjadi di produksi jika env var diset.
		appJwtKey = []byte("fallback_insecure_default_jwt_key_32_bytes_long")
	}

	expirationTime := time.Now().Add(time.Duration(cfg.Auth.JWTExpiration) * time.Hour)

	claims := UserClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(appJwtKey)
}

// ValidateToken memvalidasi token JWT
func ValidateToken(tokenString string) (*UserClaims, error) {
	cfg := config.GetConfig() // Ambil konfigurasi
	appJwtKey := []byte(cfg.Auth.JWTSecret)
	if len(appJwtKey) == 0 {
		log.Println("[AUTH] CRITICAL: JWTSecret for validation is empty. Token validation will likely fail or use insecure key.")
		// Jika JWTSecret kosong saat validasi, kemungkinan besar akan gagal jika token dibuat dengan secret non-empty
		// atau jika token dibuat dengan fallback key di GenerateToken, kita harus gunakan fallback yang sama.
		appJwtKey = []byte("fallback_insecure_default_jwt_key_32_bytes_long")
	}

	log.Printf("[AUTH] Attempting to validate token: %s\n", tokenString)
	claims := &UserClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("[AUTH] Unexpected signing method: %v\n", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return appJwtKey, nil
	})

	if err != nil {
		log.Printf("[AUTH] Error parsing or validating token: %v\n", err)
		if errors.Is(err, jwt.ErrTokenMalformed) {
			log.Println("[AUTH] Token is malformed.")
			return nil, ErrInvalidToken // Atau error spesifik untuk malformed token
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			log.Println("[AUTH] Token signature is invalid.")
			return nil, ErrInvalidToken
		} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			log.Println("[AUTH] Token is either expired or not valid yet.")
			return nil, ErrExpiredToken
		} else {
			log.Printf("[AUTH] Unknown error validating token: %v", err)
			return nil, ErrInvalidToken // Default error untuk kegagalan validasi lainnya
		}
	}

	if !token.Valid {
		log.Println("[AUTH] Token marked as invalid by library (post-error check).")
		return nil, ErrInvalidToken
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		log.Printf("[AUTH] Token is expired (checked via claims.ExpiresAt). Now: %s, ExpiresAt: %s", time.Now(), claims.ExpiresAt.Time)
		return nil, ErrExpiredToken
	}

	log.Printf("[AUTH] Token validated successfully for user: %s\n", claims.Username)
	return claims, nil
}

// GetUserFromRequest mendapatkan data user dari request (dari cookie atau header)
func GetUserFromRequest(r *http.Request) (*UserClaims, error) {
	log.Println("[AUTH] Attempting to get user from request...")
	cookie, err := r.Cookie("auth_token")
	if err == nil {
		log.Printf("[AUTH] Found 'auth_token' cookie: %s\n", cookie.Value)
		return ValidateToken(cookie.Value)
	}
	log.Printf("[AUTH] 'auth_token' cookie not found: %v\n", err)

	return nil, ErrInvalidToken
}

// IsAuthenticated memeriksa apakah request sudah terautentikasi
func IsAuthenticated(r *http.Request) bool {
	log.Println("[AUTH] Checking if request is authenticated...")
	claims, err := GetUserFromRequest(r)
	if err != nil {
		log.Printf("[AUTH] Authentication check failed: %v\n", err)
		return false
	}
	log.Printf("[AUTH] Authentication check successful for user: %s\n", claims.Username)
	return true
}

// IsPublicPath memeriksa apakah path adalah path publik yang tidak memerlukan autentikasi
func IsPublicPath(path string) bool {
	publicPaths := []string{
		"/login",
		"/static/",
		"/favicon.ico",
	}

	for _, p := range publicPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}

	return false
}

// Logout menghapus cookie autentikasi dan mengarahkan ke halaman login.
func Logout(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		// Secure: true, // Aktifkan di produksi jika menggunakan HTTPS
		// SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
	// http.Redirect(w, r, "/login", http.StatusSeeOther) // Redirect sebaiknya dilakukan oleh handler pemanggil
}

// HashPassword mengenkripsi password menggunakan bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14) // Cost factor 14
	return string(bytes), err
}

// CheckPasswordHash membandingkan password teks biasa dengan hash bcrypt.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CreateLoginSession membuat token JWT berdasarkan UserClaims dan mengaturnya sebagai cookie.
func CreateLoginSession(w http.ResponseWriter, claims *UserClaims) error {
	tokenString, err := GenerateToken(claims.UserID, claims.Username, claims.Role)
	if err != nil {
		return fmt.Errorf("gagal membuat token: %w", err)
	}

	cfg := config.GetConfig()
	expirationDuration := time.Duration(cfg.Auth.JWTExpiration) * time.Hour

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		Expires:  time.Now().Add(expirationDuration),
		MaxAge:   int(expirationDuration.Seconds()), // MaxAge dalam detik
		HttpOnly: true,
		Secure:   false, // TODO: Buat ini dapat dikonfigurasi berdasarkan environment (misal cfg.Server.Env == "production")
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}
