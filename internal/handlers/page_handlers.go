package handlers

import (
	// Diperlukan jika getUserNameFromRequest atau fungsi lain masih menggunakannya
	"errors"
	"fmt"
	"net/http"

	// Diperlukan untuk LogoutHandler dan mungkin lainnya
	"followup/internal/auth" // Diperlukan untuk mengambil info user
	"followup/internal/models"

	// Jika masih menggunakan models.TemplateData
	"encoding/json"
	"followup/internal/render"
	"followup/internal/services" // Diperlukan untuk BrandService
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

const (
	DefaultUserPageLimit     = 25 // Default jumlah user per halaman
	DefaultFollowupPageLimit = 50 // Default jumlah followup per halaman (sesuai dokumentasi)
	MaxUserPageNavLinks      = 5  // Maksimum link navigasi halaman yang ditampilkan
	MaxFollowupPageNavLinks  = 5  // Bisa disamakan atau dibuat terpisah jika perlu
)

// PaginationData menyimpan informasi untuk rendering kontrol paginasi.
// Ini akan digunakan oleh calculatePagination helper.
type PaginationData struct {
	Pages            []int
	ShowPrevEllipsis bool
	ShowNextEllipsis bool
	CurrentPage      int
	TotalPages       int
	HasPreviousPage  bool
	HasNextPage      bool
	PreviousPage     int
	NextPage         int
}

// calculatePagination menghitung detail yang diperlukan untuk navigasi halaman.
func calculatePagination(currentPage, totalPages, maxNavLinks int) PaginationData {
	var pages []int
	startPage := int(math.Max(1, float64(currentPage-(maxNavLinks/2))))
	endPage := int(math.Min(float64(totalPages), float64(startPage+maxNavLinks-1)))

	if endPage-startPage+1 < maxNavLinks {
		if currentPage > (totalPages - maxNavLinks/2) { // Jika kita berada di akhir halaman
			startPage = int(math.Max(1, float64(totalPages-maxNavLinks+1)))
		} else { // Jika kita berada di awal halaman
			endPage = int(math.Min(float64(totalPages), float64(startPage+maxNavLinks-1)))
		}
	}
	// Koreksi jika startPage menjadi 0 atau negatif karena totalPages < maxNavLinks
	if startPage < 1 {
		startPage = 1
	}
	// Koreksi jika endPage melebihi totalPages setelah penyesuaian startPage
	if endPage > totalPages {
		endPage = totalPages
	}

	for i := startPage; i <= endPage; i++ {
		pages = append(pages, i)
	}

	return PaginationData{
		Pages:            pages,
		ShowPrevEllipsis: startPage > 1,
		ShowNextEllipsis: endPage < totalPages,
		CurrentPage:      currentPage,
		TotalPages:       totalPages,
		HasPreviousPage:  currentPage > 1,
		HasNextPage:      currentPage < totalPages,
		PreviousPage:     int(math.Max(1, float64(currentPage-1))),
		NextPage:         int(math.Min(float64(totalPages), float64(currentPage+1))),
	}
}

// PageHandler struct untuk menampung semua service yang dibutuhkan oleh page handlers.
type PageHandler struct {
	memberService   services.MemberService
	brandService    services.BrandService
	userService     services.UserServiceInterface
	followupService services.FollowupServiceInterface
}

// NewPageHandler membuat instance baru dari PageHandler.
func NewPageHandler(ms services.MemberService, bs services.BrandService, us services.UserServiceInterface, fs services.FollowupServiceInterface) *PageHandler {
	return &PageHandler{
		memberService:   ms,
		brandService:    bs,
		userService:     us,
		followupService: fs,
	}
}

// HomeHandler menangani halaman beranda (sebelumnya Home)
func (ph *PageHandler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	// Jika sudah login, redirect ke dashboard
	// Jika belum login, redirect ke halaman login
	// Logika ini mungkin perlu ditinjau ulang tergantung perilaku yang diinginkan
	// jika sudah login dan mengunjungi "/", apakah redirect ke dashboard atau tampilkan halaman khusus?
	// Untuk sekarang, konsisten dengan implementasi yang ada
	if auth.IsAuthenticated(r) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// LoginPageHandler menangani halaman login (sebelumnya LoginPage)
func (ph *PageHandler) LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	// Render template login.html (standalone)
	// Menggunakan map[string]interface{} agar konsisten dengan handler lain di page_handlers.go
	// dan juga untuk menghindari ketergantungan pada models.TemplateData jika memungkinkan
	data := map[string]interface{}{
		"Title": "Login",
	}
	render.Template(w, r, "login.html", data)
}

// LoginPostHandler menangani permintaan POST untuk login.
func (ph *PageHandler) LoginPostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			respondWithErrorJSON(w, "Gagal memproses form: "+err.Error(), http.StatusBadRequest)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == "" || password == "" {
			data := map[string]interface{}{
				"Title": "Login",
				"Error": "Username dan password tidak boleh kosong.",
			}
			w.WriteHeader(http.StatusBadRequest) // Set status code untuk input tidak valid
			render.Template(w, r, "login.html", data)
			return
		}

		userClaims, err := ph.userService.AuthenticateUser(username, password)
		if err != nil {
			log.Printf("Gagal autentikasi untuk user %s: %v", username, err)
			errorMessage := "Username atau password tidak valid."
			if errors.Is(err, auth.ErrInvalidCredentials) {
				// Pesan sudah sesuai
			} else {
				// Untuk error lain yang tidak terduga, bisa gunakan pesan yang lebih umum
				errorMessage = "Terjadi kesalahan internal saat login."
			}
			data := map[string]interface{}{
				"Title":    "Login",
				"Error":    errorMessage,
				"Username": username, // Untuk mengisi kembali field username
			}
			w.WriteHeader(http.StatusUnauthorized) // Set status code untuk autentikasi gagal
			render.Template(w, r, "login.html", data)
			return
		}

		err = auth.CreateLoginSession(w, userClaims)
		if err != nil {
			log.Printf("Gagal membuat sesi login untuk user %s: %v", username, err)
			data := map[string]interface{}{
				"Title":    "Login",
				"Error":    "Terjadi kesalahan saat mencoba login. Silakan coba lagi.",
				"Username": username,
			}
			w.WriteHeader(http.StatusInternalServerError) // Set status code untuk kesalahan server
			render.Template(w, r, "login.html", data)
			return
		}

		log.Printf("Login berhasil untuk user: %s, Role: %s", userClaims.Username, userClaims.Role)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// Fungsi helper untuk mengambil nama pengguna, dipindahkan dari handlers.go
func getUserNameFromRequest(r *http.Request) string {
	userName := "Guest"
	userClaims, err := auth.GetUserFromRequest(r)
	if err == nil && userClaims != nil {
		userName = userClaims.Username
	}
	return userName
}

// DashboardHandler menangani halaman dashboard (sebelumnya Dashboard)
func (ph *PageHandler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Dashboard",
		"Active":   "dashboard",
		"UserName": getUserNameFromRequest(r),
	}
	render.TemplateWithBase(w, r, "dashboard.html", data)
}

// MemberListHandler menangani halaman daftar member.
func (ph *PageHandler) MemberListHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Ambil parameter dari query URL
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")
		searchTerm := r.URL.Query().Get("search")
		filterBrand := r.URL.Query().Get("brand_name") // Sesuaikan dengan nama param di frontend/template
		filterStatus := r.URL.Query().Get("status")
		sortBy := r.URL.Query().Get("sort_by")
		sortOrder := r.URL.Query().Get("sort_order")

		page, _ := strconv.Atoi(pageStr)
		limit, _ := strconv.Atoi(limitStr)
		if page == 0 {
			page = 1
		}
		if limit == 0 {
			limit = 50
		} // Default limit

		members, totalRecords, totalPages, err := ph.memberService.GetAllMembers(page, limit, searchTerm, filterBrand, filterStatus, sortBy, sortOrder)
		if err != nil {
			log.Printf("Error getting members for page: %v", err)
			// Tampilkan halaman error atau pesan error di halaman member
			http.Error(w, "Gagal memuat data member: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Ambil daftar brand untuk filter dropdown
		brandsForFilter, errBrand := ph.brandService.GetAllBrandsWithMemberCount("")
		if errBrand != nil {
			log.Printf("Error getting brands for filter in member list: %v", errBrand)
			brandsForFilter = []map[string]interface{}{}
		}

		// Ambil daftar user yang bisa ditugaskan sebagai CRM
		crmAssignees, errCRM := ph.userService.GetAllAssignableUsers()
		if errCRM != nil {
			log.Printf("Error getting CRM assignees: %v", errCRM)
			crmAssignees = []models.UserBasicInfo{} // default ke kosong jika error
		}

		// Buat daftar nomor halaman untuk paginasi
		var paginationPages []int
		maxPagesToShow := 5 // Jumlah tombol halaman yang ditampilkan (misal: 1 2 3 4 5 ... atau ... 3 4 5 6 7 ...)
		startPage := int(math.Max(1, float64(page-(maxPagesToShow/2))))
		endPage := int(math.Min(float64(totalPages), float64(startPage+maxPagesToShow-1)))
		if endPage-startPage+1 < maxPagesToShow && startPage > 1 {
			startPage = int(math.Max(1, float64(endPage-maxPagesToShow+1)))
		}
		for i := startPage; i <= endPage; i++ {
			paginationPages = append(paginationPages, i)
		}

		data := map[string]interface{}{
			"Title":            "Daftar Member",
			"Active":           "member",
			"UserName":         getUserNameFromRequest(r),
			"Members":          members,
			"TotalRecords":     totalRecords,
			"TotalPages":       totalPages,
			"CurrentPage":      page,
			"Limit":            limit,
			"SearchTerm":       searchTerm,
			"FilterBrand":      filterBrand,
			"FilterStatus":     filterStatus,
			"SortBy":           sortBy,
			"SortOrder":        sortOrder,
			"BrandsForFilter":  brandsForFilter, // Untuk dropdown filter brand
			"CRMAssignees":     crmAssignees,    // Data CRM untuk dropdown
			"PaginationPages":  paginationPages,
			"ShowPrevEllipsis": startPage > 1,
			"ShowNextEllipsis": endPage < totalPages,
		}
		render.TemplateWithBase(w, r, "member.html", data)
	}
}

// LanggananHandler menangani halaman langganan (sebelumnya Langganan)
func (ph *PageHandler) LanggananHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Manajemen Langganan",
		"Active":   "langganan",
		"UserName": getUserNameFromRequest(r),
	}
	render.TemplateWithBase(w, r, "langganan.html", data)
}

// FollowupHandler menangani halaman followup (sebelumnya Followup)
func (ph *PageHandler) FollowupHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Manajemen Followup",
		"Active":   models.MenuFollowUp, // Menggunakan MenuKey
		"UserName": getUserNameFromRequest(r),
	}
	render.TemplateWithBase(w, r, "followup.html", data)
}

// UserHandler menangani halaman user (sebelumnya User)
func (ph *PageHandler) UserHandler(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	searchTerm := r.URL.Query().Get("search")
	roleFilter := r.URL.Query().Get("role")

	page, _ := strconv.Atoi(pageStr)
	if page == 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit == 0 {
		limit = DefaultUserPageLimit
	}

	users, totalRecords, err := ph.userService.GetAllUsers(page, limit, searchTerm, roleFilter)
	if err != nil {
		log.Printf("Error getting users: %v", err)
		http.Error(w, "Gagal memuat data pengguna", http.StatusInternalServerError)
		return
	}

	totalPages := 0
	if limit > 0 && totalRecords > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(limit)))
	}

	pagination := calculatePagination(page, totalPages, MaxUserPageNavLinks)

	data := map[string]interface{}{
		"Title":          "Manajemen User",
		"Active":         models.MenuUserList,
		"UserName":       getUserNameFromRequest(r),
		"Users":          users,
		"TotalRecords":   totalRecords,
		"PaginationData": pagination, // Menggunakan struct PaginationData
		"Limit":          limit,
		"SearchTerm":     searchTerm,
		"RoleFilter":     roleFilter,
		"AllRoles":       models.GetValidRoles(),
	}

	if r.Header.Get("HX-Request") == "true" || r.URL.Query().Get("fragment") == "true" {
		// Jika ini adalah request HTMX (misalnya dari paginasi atau filter) ATAU request JS dengan fragment=true,
		// render hanya bagian tabel
		render.Template(w, r, "user.html#user_list_content", data)
	} else {
		// Jika ini adalah request halaman penuh, render seluruh halaman
		render.TemplateWithBase(w, r, "user.html", data)
	}
}

// ShowUserFormHandler menangani permintaan untuk menampilkan form tambah/edit user.
// Untuk kasus "tambah user", tidak ada user ID, jadi form akan kosong.
// Untuk kasus "edit user", user ID akan ada, dan form akan diisi dengan data user tersebut.
func (ph *PageHandler) ShowUserFormHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	isEditMode := false
	var userIDInt64 int64

	formData := make(map[string]interface{})
	pageTitle := "Tambah User Baru"

	if userIDStr != "" {
		var errParse error
		userIDInt64, errParse = strconv.ParseInt(userIDStr, 10, 64)
		if errParse != nil {
			log.Printf("Error parsing userID '%s' for edit user form: %v", userIDStr, errParse)
			w.WriteHeader(http.StatusBadRequest)
			// Optional: render error message or trigger client-side message
			fmt.Fprintln(w, "<div class='p-4 text-sm text-red-700 bg-red-100 rounded'>User ID tidak valid.</div>") // Placeholder response
			return
		}
		isEditMode = true
		pageTitle = "Edit User"

		userToEdit, errService := ph.userService.GetUserByID(userIDInt64)
		if errService != nil {
			if errors.Is(errService, models.ErrUserNotFound) {
				log.Printf("User with ID %d not found for edit.", userIDInt64)
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, "<div class='p-4 text-sm text-red-700 bg-red-100 rounded'>User dengan ID %d tidak ditemukan.</div>", userIDInt64) // Placeholder
			} else {
				log.Printf("Error fetching user with ID %d for edit: %v", userIDInt64, errService)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "<div class='p-4 text-sm text-red-700 bg-red-100 rounded'>Gagal memuat data user. Silakan coba lagi.</div>") // Placeholder
			}
			return
		}

		formData["Username"] = userToEdit.Username
		formData["Name"] = userToEdit.Name
		formData["Email"] = userToEdit.Email
		formData["Role"] = userToEdit.Role.String() // Kirim sebagai string untuk perbandingan di template
	}

	validRoles := models.GetValidRoles()
	assignableRoles := []models.UserRole{}
	for _, role := range validRoles {
		// Superadmin tidak bisa di-assign atau diedit menjadi superadmin via form ini
		// Jika user yang diedit adalah superadmin, perannya tidak akan muncul di dropdown,
		// sehingga tidak bisa diubah dari superadmin ke peran lain melalui form ini.
		// Ini adalah perilaku yang diinginkan untuk proteksi superadmin.
		if role != models.SuperadminRole {
			assignableRoles = append(assignableRoles, role)
		}
	}

	pageData := map[string]interface{}{
		"Title":            pageTitle,
		"IsEditMode":       isEditMode,
		"UserID":           userIDInt64, // Akan 0 jika bukan mode edit
		"AllRoles":         assignableRoles,
		"FormData":         formData,
		"ValidationErrors": make(map[string]string),
		"FormError":        "",
	}

	render.Template(w, r, "user.html#user_form_content", map[string]interface{}{"Data": pageData})
}

// CreateUserHandler menangani pembuatan pengguna baru.
func (ph *PageHandler) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form for create user: %v", err)
		respondWithErrorJSON(w, "Gagal memproses form. Silakan coba lagi.", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")
	roleStr := r.FormValue("role")

	createdUser, validationErrors, errService := ph.userService.CreateUser(username, name, email, password, confirmPassword, roleStr)

	if len(validationErrors) > 0 {
		log.Printf("CreateUser: Validation errors for user '%s': %v", username, validationErrors)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity) // 422
		errEncode := json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Periksa kembali input Anda.",
			"errors":  validationErrors,
		})
		if errEncode != nil {
			log.Printf("CRITICAL: Failed to encode JSON response for validation errors (user: %s): %v", username, errEncode)
		}
		return
	}

	if errService != nil {
		log.Printf("Service error creating user '%s': %v", username, errService)
		if errors.Is(errService, models.ErrSuperadminAlreadyExists) {
			respondWithErrorJSON(w, models.ErrSuperadminAlreadyExists.Error(), http.StatusConflict) // 409
		} else if errors.Is(errService, models.ErrDuplicateUsername) || errors.Is(errService, models.ErrDuplicateEmail) {
			respondWithErrorJSON(w, errService.Error(), http.StatusConflict) // 409
		} else {
			respondWithErrorJSON(w, "Gagal membuat pengguna: "+errService.Error(), http.StatusInternalServerError)
		}
		return
	}

	if createdUser == nil { // Ini seharusnya tidak terjadi jika tidak ada error sebelumnya
		log.Printf("CRITICAL: CreateUser service returned nil user without any errors for username: %s", username)
		respondWithErrorJSON(w, "Terjadi kesalahan internal yang tidak terduga saat membuat pengguna.", http.StatusInternalServerError)
		return
	}

	log.Printf("User '%s' (ID: %d) berhasil dibuat.", createdUser.Username, createdUser.ID)
	respondWithSuccessJSON(w, fmt.Sprintf("User '%s' berhasil ditambahkan!", createdUser.Username), http.StatusCreated, map[string]interface{}{"userID": createdUser.ID})
}

// DeleteUserHandler menangani permintaan untuk menghapus pengguna.
func (ph *PageHandler) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("Error parsing userID '%s' for delete user: %v", userIDStr, err)
		// Jika ini adalah request HTMX, mungkin ingin mengirim respons yang bisa ditampilkan di UI
		// Untuk sekarang, kirim error standar.
		w.WriteHeader(http.StatusBadRequest)
		// Di sini bisa ditambahkan HX-Trigger untuk menampilkan pesan error di frontend jika diperlukan
		// w.Header().Set("HX-Trigger", `{"showMessage": {"level": "error", "message": "User ID tidak valid."}}`)
		// http.Error(w, "User ID tidak valid", http.StatusBadRequest) // Atau render pesan error
		return
	}

	err = ph.userService.DeleteUser(userID)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			log.Printf("User with ID %d not found for deletion.", userID)
			w.WriteHeader(http.StatusNotFound) // 404 Not Found
			// Bisa ditambahkan HX-Trigger untuk pesan error
			// w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showMessage": {"level": "error", "message": "User dengan ID %d tidak ditemukan."}}`, userID))
			// http.Error(w, "User tidak ditemukan", http.StatusNotFound)
		} else {
			log.Printf("Error deleting user with ID %d: %v", userID, err)
			w.WriteHeader(http.StatusInternalServerError) // 500 Internal Server Error
			// w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showMessage": {"level": "error", "message": "Gagal menghapus user: %s"}}`, err.Error()))
			// http.Error(w, "Gagal menghapus user", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("User with ID %d successfully deleted.", userID)
	// Set HX-Trigger untuk me-refresh tabel dan mungkin menampilkan pesan sukses
	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showMessage": {"level": "success", "message": "User ID %d berhasil dihapus!"}, "userTableRefresh": true}`, userID))
	w.WriteHeader(http.StatusNoContent) // 204 No Content, karena tabel akan di-refresh oleh trigger
}

// UpdateUserHandler menangani permintaan untuk memperbarui pengguna yang ada.
func (ph *PageHandler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		log.Printf("Error parsing userID '%s' for update user: %v", userIDStr, err)
		respondWithErrorJSON(w, "User ID tidak valid.", http.StatusBadRequest)
		return
	}

	var requestPayload struct {
		Name            string `json:"name"`
		Email           string `json:"email"`
		Role            string `json:"role"`
		Password        string `json:"password,omitempty"`
		ConfirmPassword string `json:"confirm_password,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestPayload); err != nil {
		log.Printf("Error decoding request body for update user ID %d: %v", userID, err)
		respondWithErrorJSON(w, "Request body tidak valid.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	validationErrors, serviceErr := ph.userService.UpdateUser(userID,
		requestPayload.Name,
		requestPayload.Email,
		requestPayload.Role,
		requestPayload.Password,
		requestPayload.ConfirmPassword,
	)

	if validationErrors != nil {
		// Kirim error validasi sebagai JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity) // 422
		json.NewEncoder(w).Encode(map[string]interface{}{"errors": validationErrors})
		return
	}

	if serviceErr != nil {
		if errors.Is(serviceErr, models.ErrUserNotFound) {
			log.Printf("User with ID %d not found for update.", userID)
			respondWithErrorJSON(w, "User tidak ditemukan.", http.StatusNotFound)
		} else {
			log.Printf("Service error updating user ID %d: %v", userID, serviceErr)
			respondWithErrorJSON(w, "Gagal memperbarui pengguna: "+serviceErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	log.Printf("User with ID %d successfully updated.", userID)
	respondWithSuccessJSON(w, "Pengguna berhasil diperbarui.", http.StatusOK, nil)
}

// DepositHandler menangani halaman deposit (sebelumnya Deposit)
func (ph *PageHandler) DepositHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Manajemen Deposit",
		"Active":   "deposit",
		"UserName": getUserNameFromRequest(r),
	}
	render.TemplateWithBase(w, r, "deposit.html", data)
}

// BonusHandler menangani halaman bonus (sebelumnya Bonus)
func (ph *PageHandler) BonusHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Manajemen Bonus",
		"Active":   "bonus",
		"UserName": getUserNameFromRequest(r),
	}
	render.TemplateWithBase(w, r, "bonus.html", data)
}

// SettingHandler menangani halaman setting (sebelumnya Setting)
func (ph *PageHandler) SettingHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Pengaturan Sistem",
		"Active":   "setting",
		"UserName": getUserNameFromRequest(r),
	}
	render.TemplateWithBase(w, r, "setting.html", data)
}

// NotFoundHandler menangani 404 errors (sebelumnya NotFound)
func (ph *PageHandler) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	data := map[string]interface{}{
		"Title": "Halaman Tidak Ditemukan",
	}
	render.Template(w, r, "404.html", data) // Menggunakan render.Template karena ini halaman error standalone dan ada di root templates, bukan pages
}

// MethodNotAllowedHandler menangani 405 errors (sebelumnya MethodNotAllowed)
func (ph *PageHandler) MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	data := map[string]interface{}{
		"Title": "Metode Tidak Diizinkan",
	}
	// Sebaiknya ada template khusus 405, tapi untuk sekarang menggunakan 501.html seperti di kode awal
	// atau bisa juga menggunakan pages/405.html jika ada. Di sini mengacu ke root templates.
	render.Template(w, r, "501.html", data)
}

// respondWithErrorJSON adalah helper untuk mengirim error sebagai JSON.
func respondWithErrorJSON(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": message,
	})
}

// respondWithSuccessJSON adalah helper untuk mengirim sukses sebagai JSON.
func respondWithSuccessJSON(w http.ResponseWriter, message string, statusCode int, data interface{}) {
	payload := map[string]interface{}{
		"success": true,
		"message": message,
	}
	if data != nil {
		payload["data"] = data
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// MemberDetailHandler menangani halaman detail member
func MemberDetailHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implementasi detail member
	respondWithErrorJSON(w, "Halaman belum diimplementasikan", http.StatusNotImplemented)
}

// MemberCreateHandler menangani pembuatan member baru (API JSON)
func (ph *PageHandler) MemberCreateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var member models.MemberData
		if err := json.NewDecoder(r.Body).Decode(&member); err != nil {
			respondWithErrorJSON(w, "Request body tidak valid: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Validasi dasar (bisa diperluas)
		if member.Username == "" || member.BrandName == "" || member.PhoneNumber == "" {
			respondWithErrorJSON(w, "Username, BrandName, dan PhoneNumber wajib diisi", http.StatusBadRequest)
			return
		}

		// Jika Email kosong tapi MembershipEmail ada (dari form), gunakan MembershipEmail
		if member.Email == "" && member.MembershipEmail != "" {
			member.Email = member.MembershipEmail
		}

		// Logika untuk menentukan status "New Deposit" atau "Redeposit" saat create manual
		if member.Saldo == "" || member.Saldo == "0" {
			member.Status = "New Deposit"
		} else {
			// Jika saldo ada dan bukan 0, bisa dianggap "Redeposit" atau status lain sesuai kebutuhan
			member.Status = "Redeposit"
		}

		createdMemberID, err := ph.memberService.CreateMember(&member)
		if err != nil {
			if strings.Contains(err.Error(), "username sudah terdaftar") || strings.Contains(err.Error(), "nomor telepon sudah terdaftar") {
				respondWithErrorJSON(w, err.Error(), http.StatusConflict)
			} else {
				respondWithErrorJSON(w, "Gagal membuat member: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		respondWithSuccessJSON(w, "Member berhasil ditambahkan", http.StatusCreated, map[string]interface{}{"memberId": createdMemberID})
	}
}

// MemberUpdateHandler menangani update member (API JSON untuk No. Telepon)
func (ph *PageHandler) MemberUpdateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		memberIDStr := chi.URLParam(r, "memberId")
		memberID, err := strconv.Atoi(memberIDStr)
		if err != nil {
			respondWithErrorJSON(w, "Member ID tidak valid", http.StatusBadRequest)
			return
		}

		var payload struct {
			PhoneNumber string `json:"phoneNumber"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondWithErrorJSON(w, "Request body tidak valid: "+err.Error(), http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(payload.PhoneNumber) == "" {
			respondWithErrorJSON(w, "Nomor telepon baru tidak boleh kosong", http.StatusBadRequest)
			return
		}

		err = ph.memberService.UpdateMemberPhoneNumber(memberID, payload.PhoneNumber)
		if err != nil {
			// Anda mungkin perlu error type yang lebih spesifik dari service, seperti services.ErrMemberNotFound
			if strings.Contains(err.Error(), "tidak ditemukan") { // Periksa pesan error sementara
				respondWithErrorJSON(w, "Member tidak ditemukan", http.StatusNotFound)
			} else if strings.Contains(err.Error(), "nomor telepon sudah terdaftar") {
				respondWithErrorJSON(w, err.Error(), http.StatusConflict)
			} else {
				respondWithErrorJSON(w, "Gagal memperbarui nomor telepon: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		respondWithSuccessJSON(w, "Nomor telepon berhasil diperbarui", http.StatusOK, nil)
	}
}

// MemberDeleteHandler menangani penghapusan member (API JSON)
func (ph *PageHandler) MemberDeleteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// memberIDStr := chi.URLParam(r, "id")
		// Logika untuk menghapus member akan ditambahkan di sini nanti
		respondWithErrorJSON(w, "Fitur hapus member belum diimplementasikan", http.StatusNotImplemented)
	}
}

// BrandPageHandler (mengganti nama dari BrandHandler func untuk menghindari konflik dengan struct BrandHandler)
// menangani halaman brand dan menampilkan daftar brand.
func (ph *PageHandler) BrandPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		searchTerm := r.URL.Query().Get("search")
		successMsg := r.URL.Query().Get("success")
		errorMsg := r.URL.Query().Get("error")

		brandsWithCount, err := ph.brandService.GetAllBrandsWithMemberCount(searchTerm)
		if err != nil {
			log.Printf("Error getting brands with count for page: %v", err)
			// Jika terjadi error saat mengambil data, setidaknya tampilkan halaman dengan pesan error
			errorMsg = "Gagal memuat data brand: " + err.Error()
			brandsWithCount = []map[string]interface{}{} // Kirim slice kosong agar template tidak error
		}

		data := map[string]interface{}{
			"Title":      "Manajemen Brand",
			"Active":     "brand",
			"UserName":   getUserNameFromRequest(r),
			"Brands":     brandsWithCount,
			"SearchTerm": searchTerm,
			"SuccessMsg": successMsg,
			"ErrorMsg":   errorMsg,
		}
		render.TemplateWithBase(w, r, "brand.html", data)
	}
}

// UploadExcelPageHandler menangani halaman upload Excel
func (ph *PageHandler) UploadExcelPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		brandsForDropdown, err := ph.brandService.GetAllBrandsWithMemberCount("") // Ambil semua brand, tanpa search term
		if err != nil {
			log.Printf("Error getting brands for dropdown in upload page: %v", err)
			// Tetap tampilkan halaman meski gagal load brand, dengan dropdown kosong atau pesan error
			// Untuk sekarang, kita kirim slice kosong saja
			brandsForDropdown = []map[string]interface{}{}
		}

		data := map[string]interface{}{
			"Title":             "Upload Data Member",
			"Active":            "upload_excel",
			"UserName":          getUserNameFromRequest(r),
			"BrandsForDropdown": brandsForDropdown, // Kirim data brand ke template
		}
		render.TemplateWithBase(w, r, "upload/excel.html", data)
	}
}

// LogoutHandler menangani proses logout.
func (ph *PageHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	auth.Logout(w, r) // Menggunakan fungsi Logout yang sudah ada
	http.Redirect(w, r, "/login?logout=success", http.StatusSeeOther)
}

// UpdateMemberCRMHandler menangani permintaan untuk memperbarui CRM seorang member.
func (ph *PageHandler) UpdateMemberCRMHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		memberIDStr := chi.URLParam(r, "memberId")
		memberID, err := strconv.Atoi(memberIDStr)
		if err != nil {
			respondWithErrorJSON(w, "ID member tidak valid", http.StatusBadRequest)
			return
		}

		var requestBody struct {
			// Mengubah field agar sesuai dengan payload dari frontend
			CRMUsername *string `json:"crm_username"`
		}

		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			respondWithErrorJSON(w, "Request body tidak valid: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Untuk otorisasi, kita bisa periksa peran di service layer untuk konsistensi.
		// Untuk sementara, kita bisa biarkan otorisasi di sini atau pindahkan.
		// Jika service sudah menangani validasi user CRM, pengecekan di sini mungkin tidak perlu.
		// Mari kita sederhanakan dan andalkan validasi di service.

		// Memanggil service dengan signature yang sudah benar.
		err = ph.memberService.UpdateMemberCRM(memberID, requestBody.CRMUsername)
		if err != nil {
			if strings.Contains(err.Error(), "member tidak ditemukan") {
				respondWithErrorJSON(w, "Gagal memperbarui CRM: Member tidak ditemukan", http.StatusNotFound)
			} else if strings.Contains(err.Error(), "user CRM tidak ditemukan") || strings.Contains(err.Error(), "bukan CRM atau Telemarketing") {
				// Memberikan pesan error yang lebih spesifik dari service
				respondWithErrorJSON(w, err.Error(), http.StatusBadRequest)
			} else {
				log.Printf("Error updating member CRM (ID: %d): %v", memberID, err)
				respondWithErrorJSON(w, "Gagal memperbarui CRM: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		var crmAssignedMessage string
		if requestBody.CRMUsername != nil && *requestBody.CRMUsername != "" {
			crmAssignedMessage = *requestBody.CRMUsername
		} else {
			crmAssignedMessage = "Tidak Ditugaskan"
		}
		respondWithSuccessJSON(w, fmt.Sprintf("CRM untuk member ID %d berhasil diperbarui menjadi '%s'", memberID, crmAssignedMessage), http.StatusOK, nil)
	}
}

// TeamPageHandler menampilkan halaman manajemen tim.
func (ph *PageHandler) TeamPageHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := auth.GetUserFromRequest(r)
	if err != nil || currentUser == nil {
		// Seharusnya sudah ditangani oleh middleware, tapi sebagai fallback
		http.Redirect(w, r, "/login?error=unauthorized", http.StatusSeeOther)
		return
	}

	// Otorisasi: Pastikan pengguna memiliki izin untuk mengakses halaman ini
	if !(currentUser.Role == models.SuperadminRole || currentUser.Role == models.AdminRole) { // Sesuaikan peran yang diizinkan
		log.Printf("User %s (Role: %s) unauthorized to access /team page.", currentUser.Username, currentUser.Role)
		// Tampilkan halaman error atau redirect
		w.WriteHeader(http.StatusForbidden) // Set status code Forbidden
		render.TemplateWithBase(w, r, "error.html", &render.TemplateData{
			Title:    "Akses Ditolak",
			Error:    "Anda tidak memiliki izin untuk mengakses halaman ini.", // Menggunakan field Error
			UserName: currentUser.Username,                                    // Sertakan username untuk layout dasar jika diperlukan
			Active:   "error_page",                                            // Tandai sebagai halaman error
		})
		return
	}

	pageData := map[string]interface{}{
		"Title":    "Manajemen Tim",
		"Active":   "team_management", // Untuk menandai menu aktif di sidebar
		"UserName": currentUser.Username,
		"UserRole": currentUser.Role.String(),
	}

	render.TemplateWithBase(w, r, "team.html", &render.TemplateData{
		Title:    "Manajemen Tim",
		Active:   "team_management",
		Data:     pageData,
		UserName: currentUser.Username, // Pastikan ini juga di-pass ke AddDefaultData jika perlu
	})
}

// FollowupPageHandler menangani halaman followup.
func (ph *PageHandler) FollowupPageHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, errAuth := auth.GetUserFromRequest(r) // errAuth ditangkap
	if errAuth != nil || currentUser == nil {
		log.Printf("FollowupPageHandler: User tidak terautentikasi atau gagal mendapatkan user: %v", errAuth)
		http.Redirect(w, r, "/login?error=unauthorized", http.StatusSeeOther)
		return
	}

	if currentUser.Role == models.AdministratorRole {
		log.Printf("User %s (Role: %s) tidak diizinkan mengakses halaman Followup.", currentUser.Username, currentUser.Role)
		render.TemplateWithBase(w, r, "error.html", &render.TemplateData{
			Title:    "Akses Ditolak",
			Error:    "Anda tidak memiliki izin untuk mengakses halaman ini.",
			UserName: currentUser.Username,
			Active:   string(models.MenuFollowUp), // Diubah ke string
		})
		return
	}

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	searchTerm := r.URL.Query().Get("search")
	statusFilter := r.URL.Query().Get("status")
	brandFilterStr := r.URL.Query().Get("brand_id")

	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = DefaultFollowupPageLimit
	}

	var brandIDFilter int64
	if brandFilterStr != "" {
		brandIDFilter, _ = strconv.ParseInt(brandFilterStr, 10, 64)
	}

	followupFilters := models.FollowupFilters{
		SearchTerm: searchTerm,
		BrandID:    brandIDFilter,
	}
	if statusFilter != "" {
		followupFilters.Status = strings.Split(statusFilter, ",")
	}

	// Menggunakan currentUser (yang bertipe *auth.UserClaims) langsung
	followups, totalRecords, errService := ph.followupService.GetAllFollowups(followupFilters, page, limit, currentUser)
	if errService != nil {
		log.Printf("FollowupPageHandler: Error getting followup data: %v", errService)
		// Menampilkan pesan error yang lebih baik di halaman yang sama
		dataError := map[string]interface{}{
			"Title":                "Manajemen Followup",
			"Active":               string(models.MenuFollowUp),
			"UserName":             currentUser.Username,
			"UserRole":             currentUser.Role.String(),
			"Followups":            []models.FollowupListItem{}, // list kosong
			"TotalRecords":         0,
			"PaginationData":       calculatePagination(1, 0, MaxFollowupPageNavLinks), // Paginasi kosong
			"Limit":                limit,
			"Filters":              followupFilters,
			"PageError":            "Gagal memuat data followup. Silakan coba lagi nanti.", // Pesan error untuk UI
			"IsNewDepositFiltered": false,                                                  // Default jika terjadi error
			"IsRedepositFiltered":  false,                                                  // Default jika terjadi error
			"IsPendingFiltered":    false,                                                  // Default jika terjadi error
		}
		render.TemplateWithBase(w, r, "followup.html", dataError)
		return
	}

	totalPages := 0
	if limit > 0 && totalRecords > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(limit)))
	}

	pagination := calculatePagination(page, totalPages, MaxFollowupPageNavLinks)

	// Menentukan status filter mana yang aktif untuk template
	isNewDepositFiltered := false
	isRedepositFiltered := false
	isPendingFiltered := false
	for _, s := range followupFilters.Status {
		if s == models.StatusFollowupNewDeposit {
			isNewDepositFiltered = true
		}
		if s == models.StatusFollowupRedeposit {
			isRedepositFiltered = true
		}
		if s == models.StatusFollowupPending { // Menggunakan konstanta dari models
			isPendingFiltered = true
		}
	}

	data := map[string]interface{}{
		"Title":                "Manajemen Followup",
		"Active":               string(models.MenuFollowUp),
		"UserName":             currentUser.Username,
		"UserRole":             currentUser.Role.String(),
		"Followups":            followups,
		"TotalRecords":         totalRecords,
		"PaginationData":       pagination,
		"Limit":                limit,
		"Filters":              followupFilters,
		"IsNewDepositFiltered": isNewDepositFiltered,
		"IsRedepositFiltered":  isRedepositFiltered,
		"IsPendingFiltered":    isPendingFiltered,
	}

	if r.Header.Get("HX-Request") == "true" || r.URL.Query().Get("fragment") == "true" {
		render.Template(w, r, "followup.html#followup_list_content", data)
	} else {
		render.TemplateWithBase(w, r, "followup.html", data)
	}
}
