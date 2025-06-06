package router

import (
	"database/sql" // Diperlukan untuk *sql.DB
	"net/http"

	// "followup/internal/auth" // auth.Logout sekarang dipanggil via pageHandler.LogoutHandler
	"followup/internal/handlers"
	authMiddleware "followup/internal/middleware/auth" // Alias untuk paket middleware auth DIKEMBALIKAN
	"followup/internal/repository"                     // Diperlukan untuk NewUserRepository
	"followup/internal/services"                       // Diperlukan untuk services

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// SetupRouter mengkonfigurasi rute aplikasi
func SetupRouter(
	db *sql.DB, // Tambahkan *sql.DB untuk inisialisasi repo
	brandSvc services.BrandService, // Tetap dibutuhkan oleh PageHandler
	brandActHandler *handlers.BrandHandler, // Untuk aksi brand CRUD yang mungkin belum jadi method PageHandler
	memberSvc services.MemberService, // Tetap dibutuhkan oleh PageHandler
	// uploadExcelHandler *handlers.UploadExcelHandler, // Jika ini adalah struct, harus diinisialisasi dan di-pass
	// loginHandler *handlers.LoginHandler // Jika login handler memerlukan dependencies, inject di sini
) http.Handler {
	r := chi.NewRouter()

	// Middleware global
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.StripSlashes)

	// Inisialisasi Repositories
	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	followupRepo := repository.NewFollowupRepository(db) // Tambahkan FollowupRepository

	// Inisialisasi Services
	userSvc := services.NewUserService(userRepo, teamRepo)
	teamSvc := services.NewTeamService(teamRepo, userRepo)
	followupSvc := services.NewFollowupService(followupRepo) // Tambahkan FollowupService

	// Team Dependencies
	// teamRepo sudah diinisialisasi di atas
	// teamSvc memerlukan UserRepository untuk beberapa validasi, jadi kita pass userRepo
	teamHandler := handlers.NewTeamHandler(teamSvc) // Inisialisasi TeamHandler

	// Inisialisasi PageHandler dengan semua service
	pageHandler := handlers.NewPageHandler(memberSvc, brandSvc, userSvc, followupSvc)

	// Rute untuk file statis
	fileServer := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Rute publik/tidak memerlukan autentikasi
	r.Group(func(r chi.Router) {
		r.Get("/", pageHandler.HomeHandler)           // Menggunakan method dari PageHandler
		r.Get("/login", pageHandler.LoginPageHandler) // Menggunakan method dari PageHandler
		// Menggunakan LoginPostHandler yang baru dari PageHandler
		r.Post("/login", pageHandler.LoginPostHandler())
	})

	// Rute yang memerlukan autentikasi
	r.Group(func(r chi.Router) {
		authMW := authMiddleware.Middleware(userSvc) // Membuat instance middleware dengan userService
		r.Use(authMW)                                // Menggunakan instance middleware

		// Dashboard
		r.Get("/dashboard", pageHandler.DashboardHandler) // Menggunakan method dari PageHandler

		// Brand Page (display) dan Actions
		r.Route("/brand", func(r chi.Router) {
			r.Get("/", pageHandler.BrandPageHandler()) // Menggunakan method dari PageHandler
			// brandActHandler adalah struct terpisah untuk API CRUD brand, jadi pemanggilannya tetap
			r.Post("/add", brandActHandler.AddBrand)
			r.Post("/edit/{id}", brandActHandler.UpdateBrand)
			r.Post("/delete/{id}", brandActHandler.DeleteBrand)
		})

		// Member
		r.Route("/member", func(r chi.Router) {
			r.Get("/", pageHandler.MemberListHandler()) // Menggunakan method dari PageHandler
			// r.Get("/{id}", pageHandler.MemberDetailHandler()) // Jika MemberDetailHandler juga method PageHandler
			r.Post("/", pageHandler.MemberCreateHandler()) // Menggunakan method dari PageHandler
			// Path untuk update telepon adalah /member/{memberId}/phone
			r.Put("/{memberId}/phone", pageHandler.MemberUpdateHandler())
			// r.Delete("/{memberId}", pageHandler.MemberDeleteHandler()) // Jika MemberDeleteHandler juga method PageHandler

			// Rute baru untuk update CRM
			r.Put("/{memberId}/crm", pageHandler.UpdateMemberCRMHandler()) // Rute baru
		})

		// Upload Excel
		r.Route("/upload", func(r chi.Router) {
			r.Get("/excel", pageHandler.UploadExcelPageHandler())
			r.Post("/excel", handlers.UploadExcelHandler)
			r.Post("/excel/import", handlers.ImportExcelHandler)
		})

		// Rute untuk Manajemen User
		r.Route("/user", func(r chi.Router) {
			r.Get("/", pageHandler.UserHandler)                           // Menampilkan daftar user & form filter
			r.Get("/form/add", pageHandler.ShowUserFormHandler)           // Menampilkan form tambah user
			r.Get("/form/edit/{userID}", pageHandler.ShowUserFormHandler) // Rute untuk menampilkan form edit user
			r.Post("/create", pageHandler.CreateUserHandler)              // Memproses pembuatan user baru
			r.Post("/update/{userID}", pageHandler.UpdateUserHandler)     // Rute untuk memproses pembaruan user
			r.Post("/delete/{userID}", pageHandler.DeleteUserHandler)     // Rute untuk hapus user
		})

		// Rute untuk Halaman Manajemen Tim
		r.Get("/team", pageHandler.TeamPageHandler) // Menampilkan halaman utama tim

		// Rute API untuk Tim
		r.Route("/api/teams", func(r chi.Router) {
			r.Post("/", teamHandler.CreateTeamHandler)           // POST /api/teams (Buat tim baru)
			r.Get("/", teamHandler.ListTeamsHandler)             // GET /api/teams (Daftar tim dengan paginasi & search)
			r.Get("/{teamID}", teamHandler.GetTeamHandler)       // GET /api/teams/{teamID} (Detail tim)
			r.Put("/{teamID}", teamHandler.UpdateTeamHandler)    // PUT /api/teams/{teamID} (Update tim)
			r.Delete("/{teamID}", teamHandler.DeleteTeamHandler) // DELETE /api/teams/{teamID} (Hapus tim)

			r.Get("/{teamID}/members", teamHandler.GetTeamMembersHandler)               // GET /api/teams/{teamID}/members (Daftar anggota tim)
			r.Post("/{teamID}/members", teamHandler.AddTeamMemberHandler)               // POST /api/teams/{teamID}/members (Tambah anggota)
			r.Delete("/{teamID}/members/{userID}", teamHandler.RemoveTeamMemberHandler) // DELETE /api/teams/{teamID}/members/{userID} (Hapus anggota)
		})

		// Rute API tambahan untuk mendapatkan user yang bisa ditugaskan
		r.Route("/api/users", func(r chi.Router) {
			// Endpoint ini mungkin lebih baik di user handler jika general,
			// tapi jika spesifik untuk tim, bisa tetap di teamHandler atau service.
			// Saya taruh di teamHandler untuk saat ini karena service-nya sudah ada di sana.
			r.Get("/assignable-to-team", teamHandler.GetAssignableUsersForTeamHandler)   // GET /api/users/assignable-to-team
			r.Get("/assignable-as-admin", teamHandler.GetAssignableAdminsForTeamHandler) // GET /api/users/assignable-as-admin
		})

		// Rute Halaman Lainnya
		r.Get("/followup", pageHandler.FollowupPageHandler)
		r.Get("/langganan", pageHandler.LanggananHandler)
		r.Get("/deposit", pageHandler.DepositHandler)
		r.Get("/bonus", pageHandler.BonusHandler)
		r.Get("/setting", pageHandler.SettingHandler)

		// Logout
		// Menggunakan method dari PageHandler karena LogoutHandler di page_handlers.go sudah di-refactor.
		r.Get("/logout", pageHandler.LogoutHandler)
	})

	// Anda bisa mengaktifkan ini jika ingin semua routing error ditangani oleh PageHandler.
	// r.NotFound(pageHandler.NotFoundHandler)
	// r.MethodNotAllowed(pageHandler.MethodNotAllowedHandler)

	return r
}
