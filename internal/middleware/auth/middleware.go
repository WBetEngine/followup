package auth

import (
	"context"
	"log"
	"net/http"

	"followup/internal/auth"
	"followup/internal/services"
)

// contextKey adalah tipe lokal untuk kunci konteks di paket middleware ini.
type contextKey string

// AllowedMenuKeysKey adalah kunci untuk daftar MenuKey yang diizinkan dalam context.
const AllowedMenuKeysKey contextKey = "allowedMenuKeys"

// Middleware membuat middleware autentikasi yang juga menangani izin menu.
func Middleware(userService services.UserServiceInterface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip autentikasi untuk path publik
			if auth.IsPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Coba dapatkan user claims dari request
			claims, err := auth.GetUserFromRequest(r) // Fungsi ini sudah melakukan logging internal
			if err != nil {
				// Jika ada error (token tidak valid, kadaluarsa, atau tidak ada), redirect ke login.
				// Logging tambahan di middleware bisa membantu menelusuri alur dari sisi middleware.
				log.Printf("[AUTH_MIDDLEWARE] Pengguna belum terautentikasi untuk path %s: %v. Mengarahkan ke /login.", r.URL.Path, err)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Jika berhasil, tambahkan claims ke context request
			// Handler selanjutnya dapat mengakses claims ini menggunakan auth.GetClaimsFromContext(r.Context())
			ctx := context.WithValue(r.Context(), auth.UserClaimsKey, claims)

			// Dapatkan dan tambahkan izin menu ke context
			// Pastikan claims.Role adalah tipe yang benar (models.UserRole) yang diterima oleh GetMenuKeysForRole
			allowedMenus := userService.GetMenuKeysForRole(claims.Role)
			ctx = context.WithValue(ctx, AllowedMenuKeysKey, allowedMenus)
			log.Printf("[AUTH_MIDDLEWARE] User: %s, Role: %s, AllowedMenus: %v untuk path: %s", claims.Username, claims.Role, allowedMenus, r.URL.Path)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
