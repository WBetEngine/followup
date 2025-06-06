package services

import (
	"context"
	"errors"
	"fmt"
	"followup/internal/auth"
	"followup/internal/models"
	"followup/internal/repository"
	"log"
	"strings"
	"time"
)

// UserServiceInterface mendefinisikan interface untuk layanan terkait User.
type UserServiceInterface interface {
	GetAllAssignableUsers() ([]models.UserBasicInfo, error)
	AuthenticateUser(username, password string) (*auth.UserClaims, error)
	GetMenuKeysForRole(role models.UserRole) []models.MenuKey
	GetAllUsers(page, limit int, searchTerm, roleFilter string) ([]models.User, int, error)
	CreateUser(username, name, email, password, confirmPassword, roleStr string) (*models.User, map[string]string, error)
	DeleteUser(id int64) error
	GetUserByID(id int64) (*models.User, error)
	UpdateUser(id int64, name, email, roleStr, password, confirmPassword string) (map[string]string, error)
	GetUsersAvailableForTeamMembership(ctx context.Context, searchTerm string, limit int) ([]models.UserBasicInfo, error)
	// Tambahkan fungsi layanan lain terkait user jika diperlukan
}

// userService adalah implementasi dari UserServiceInterface.
type userService struct {
	userRepo repository.UserRepositoryInterface
	teamRepo repository.TeamRepositoryInterface
}

// NewUserService membuat instance baru dari userService.
func NewUserService(userRepo repository.UserRepositoryInterface, teamRepo repository.TeamRepositoryInterface) UserServiceInterface {
	return &userService{userRepo: userRepo, teamRepo: teamRepo}
}

// GetAllAssignableUsers mengambil daftar semua pengguna yang dapat ditugaskan sebagai CRM.
func (s *userService) GetAllAssignableUsers() ([]models.UserBasicInfo, error) {
	users, err := s.userRepo.GetAllAssignableUsers()
	if err != nil {
		// Error sudah di-log di repository, bisa ditambahkan logging konteks service jika perlu
		return nil, fmt.Errorf("service failed to get assignable users: %w", err)
	}
	return users, nil
}

// AuthenticateUser memverifikasi kredensial pengguna dan mengembalikan UserClaims jika berhasil.
func (s *userService) AuthenticateUser(username, password string) (*auth.UserClaims, error) {
	user, err := s.userRepo.GetUserByUsername(username)
	if err != nil {
		// Log error sudah ada di repository, atau bisa ditambahkan di sini jika perlu konteks lebih.
		// Mengembalikan error yang lebih umum ke handler untuk keamanan.
		log.Printf("[AUTH_DEBUG] User not found in DB or DB error for username: '%s': %v", username, err) // Tambahan log jika user tidak ditemukan
		return nil, auth.ErrInvalidCredentials                                                            // Gunakan error dari paket auth
	}

	log.Printf("[AUTH_DEBUG] Attempting to authenticate user: '%s'", username)
	log.Printf("[AUTH_DEBUG] Password from form: '%s'", password)                        // Log password dari form
	log.Printf("[AUTH_DEBUG] Hash from DB for user '%s': '%s'", username, user.Password) // Log hash dari DB

	if !auth.CheckPasswordHash(password, user.Password) { // user.Password adalah hash dari DB
		log.Printf("[AUTH_DEBUG] Password check FAILED for user: '%s'", username) // Log jika gagal
		return nil, auth.ErrInvalidCredentials
	}
	log.Printf("[AUTH_DEBUG] Password check PASSED for user: '%s'", username) // Log jika berhasil

	// Jika kredensial valid, buat UserClaims.
	// Field Name pada UserClaims bersifat opsional dan tidak ada di definisi auth.UserClaims saat ini.
	// Jika Name diperlukan di JWT, definisi UserClaims di auth.go perlu disesuaikan.
	claims := &auth.UserClaims{
		UserID:   int(user.ID), // Konversi int64 ke int jika UserID di claims adalah int
		Username: user.Username,
		Role:     user.Role,
	}

	return claims, nil
}

// GetMenuKeysForRole mengembalikan daftar MenuKey yang diizinkan untuk peran tertentu.
// Ini adalah implementasi hardcoded dan bisa diperluas atau dipindahkan ke sistem berbasis DB nanti.
func (s *userService) GetMenuKeysForRole(role models.UserRole) []models.MenuKey {
	allMenus := models.AllMenuKeys() // Dapatkan semua menu key yang valid

	switch role {
	case models.SuperadminRole:
		return allMenus // Superadmin mendapatkan semua menu
	case models.AdminRole, models.AdministratorRole: // Administrator disamakan dengan Admin untuk saat ini
		return []models.MenuKey{
			models.MenuDashboard,
			models.MenuMemberList,
			models.MenuBrandList,
			models.MenuUploadMember,
			models.MenuUserList,
			models.MenuTeam,
			models.MenuFollowUp,
			models.MenuLangganan,
			// models.MenuInvalidNumber, // Contoh tidak diizinkan untuk Admin
			models.MenuDepositList,
			// models.MenuWithdrawalList, // Contoh tidak diizinkan untuk Admin
			// models.MenuWallet,         // Contoh tidak diizinkan untuk Admin
			models.MenuSettings,
			models.MenuBonusList,
		}
	case models.TelemarketingRole:
		return []models.MenuKey{
			models.MenuDashboard,
			models.MenuFollowUp,
			models.MenuLangganan,
			models.MenuInvalidNumber,
		}
	case models.CRMRole:
		return []models.MenuKey{
			models.MenuDashboard,
			models.MenuMemberList, // CRM mungkin perlu akses ke daftar member untuk melihat/mengelola info CRM mereka
			models.MenuFollowUp,
			// Mungkin perlu izin yang lebih granular di dalam halaman member untuk CRM
		}
	default:
		// Jika peran tidak dikenal, tidak ada menu yang diizinkan
		log.Printf("Peringatan: Peran tidak dikenal '%s' saat meminta izin menu.", role)
		return []models.MenuKey{}
	}
}

// GetAllUsers mengambil daftar semua pengguna dengan paginasi, pencarian, dan filter.
func (s *userService) GetAllUsers(page, limit int, searchTerm, roleFilter string) ([]models.User, int, error) {
	users, totalRecords, err := s.userRepo.GetAllUsers(page, limit, searchTerm, roleFilter)
	if err != nil {
		// Log error sudah terjadi di repository, bisa ditambahkan konteks service jika perlu
		log.Printf("Service error getting all users: %v", err) // Tambahkan log di service
		return nil, 0, fmt.Errorf("service failed to get all users: %w", err)
	}
	return users, totalRecords, nil
}

// CreateUser menangani logika bisnis untuk membuat pengguna baru.
func (s *userService) CreateUser(username, name, email, password, confirmPassword, roleStr string) (*models.User, map[string]string, error) {
	validationErrors := make(map[string]string)

	// Validasi input dasar
	if strings.TrimSpace(username) == "" {
		validationErrors["username"] = "Username tidak boleh kosong."
	}
	if strings.TrimSpace(name) == "" {
		validationErrors["name"] = "Nama lengkap tidak boleh kosong."
	}
	if strings.TrimSpace(email) == "" {
		validationErrors["email"] = "Email tidak boleh kosong."
	}
	if password == "" {
		validationErrors["password"] = "Password tidak boleh kosong."
	} else if len(password) < 6 { // Contoh validasi panjang password minimal
		validationErrors["password"] = "Password minimal 6 karakter."
	}
	if password != confirmPassword {
		validationErrors["confirm_password"] = "Konfirmasi password tidak cocok."
	}

	parsedRole, errRole := models.ParseUserRole(roleStr)
	if errRole != nil {
		validationErrors["role"] = "Peran tidak valid."
	} else if parsedRole == models.SuperadminRole {
		// Validasi tambahan: Jangan izinkan pembuatan superadmin baru melalui form ini secara langsung
		// Ini sudah ditangani di handler dengan tidak menampilkan opsi, tapi sebagai lapisan pertahanan tambahan.
		// Aturan "hanya satu superadmin" akan diimplementasikan lebih lanjut.
		// validasi SuperadminExists dipindahkan ke bawah setelah validasi dasar lainnya
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors, nil // Tidak ada error service, hanya error validasi
	}

	// Pemeriksaan SuperadminExists dilakukan di sini setelah validasi dasar lolos
	if parsedRole == models.SuperadminRole {
		exists, errCheck := s.userRepo.SuperadminExists()
		if errCheck != nil {
			log.Printf("Service error checking if superadmin exists: %v", errCheck)
			return nil, nil, fmt.Errorf("gagal memverifikasi status superadmin: %w", errCheck)
		}
		if exists {
			return nil, nil, models.ErrSuperadminAlreadyExists // Kembalikan error spesifik
		}
	}

	hashedPassword, errHash := auth.HashPassword(password)
	if errHash != nil {
		log.Printf("Error hashing password for user %s: %v", username, errHash)
		return nil, nil, fmt.Errorf("gagal memproses password: %w", errHash)
	}

	now := time.Now()
	newUser := models.User{
		Username:  username,
		Name:      name,
		Email:     email,
		Password:  hashedPassword, // Menggunakan field Password sesuai model Anda
		Role:      parsedRole,
		CreatedAt: now,
		UpdatedAt: now,
	}

	errRepo := s.userRepo.CreateUser(&newUser) // repo.CreateUser akan mengisi ID, CreatedAt, UpdatedAt pada newUser
	if errRepo != nil {
		if errors.Is(errRepo, models.ErrDuplicateUsername) {
			validationErrors["username"] = "Username sudah terdaftar."
			return nil, validationErrors, nil
		} else if errors.Is(errRepo, models.ErrDuplicateEmail) {
			validationErrors["email"] = "Email sudah terdaftar."
			return nil, validationErrors, nil
		}
		log.Printf("Repository error creating user '%s': %v", username, errRepo)
		return nil, nil, fmt.Errorf("gagal menyimpan pengguna: %w", errRepo)
	}

	return &newUser, nil, nil // Sukses
}

// DeleteUser menangani logika bisnis untuk menghapus pengguna.
func (s *userService) DeleteUser(id int64) error {
	err := s.userRepo.DeleteUser(id)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			// Bisa langsung return error ini jika handler ingin tahu secara spesifik
			return models.ErrUserNotFound
		}
		// Untuk error lain dari repository
		log.Printf("Service error deleting user with ID %d: %v", id, err)
		return fmt.Errorf("gagal menghapus pengguna di service: %w", err)
	}
	return nil
}

// GetUserByID mengambil data pengguna berdasarkan ID.
func (s *userService) GetUserByID(id int64) (*models.User, error) {
	user, err := s.userRepo.GetUserByID(id)
	if err != nil {
		// Log error bisa ditambahkan di sini jika perlu, tapi repository sudah log
		// Kembalikan error sebagaimana adanya, termasuk models.ErrUserNotFound
		return nil, err
	}
	return user, nil
}

// UpdateUser menangani logika bisnis untuk memperbarui pengguna.
func (s *userService) UpdateUser(id int64, name, email, roleStr, password, confirmPassword string) (map[string]string, error) {
	validationErrors := make(map[string]string)

	// Validasi input dasar
	if strings.TrimSpace(name) == "" {
		validationErrors["name"] = "Nama lengkap tidak boleh kosong."
	}
	if strings.TrimSpace(email) == "" {
		validationErrors["email"] = "Email tidak boleh kosong."
	}

	parsedRole, errRole := models.ParseUserRole(roleStr)
	if errRole != nil {
		validationErrors["role"] = "Peran tidak valid."
	} else if parsedRole == models.SuperadminRole {
		// Mencegah perubahan peran menjadi Superadmin melalui form ini.
		// Jika user yang diedit adalah Superadmin, perannya juga tidak boleh diubah dari sini.
		// Logika ini perlu diperkuat dengan memeriksa peran user yang ada.
		validationErrors["role"] = "Peran Superadmin tidak dapat diatur atau diubah melalui form ini."
	}

	// Validasi password hanya jika diisi
	if password != "" {
		if len(password) < 6 {
			validationErrors["password"] = "Password minimal 6 karakter."
		}
		if password != confirmPassword {
			validationErrors["confirm_password"] = "Konfirmasi password tidak cocok."
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors, nil // Tidak ada error service, hanya error validasi
	}

	// Ambil data user yang ada
	existingUser, err := s.userRepo.GetUserByID(id)
	if err != nil {
		// Jika user tidak ditemukan, ini adalah error service
		if errors.Is(err, models.ErrUserNotFound) {
			return nil, models.ErrUserNotFound
		}
		log.Printf("Service error getting user by ID %d for update: %v", id, err)
		return nil, fmt.Errorf("gagal mengambil data pengguna untuk pembaruan: %w", err)
	}

	// Mencegah perubahan peran dari atau ke Superadmin
	if (existingUser.Role == models.SuperadminRole && parsedRole != models.SuperadminRole) ||
		(existingUser.Role != models.SuperadminRole && parsedRole == models.SuperadminRole) {
		validationErrors["role"] = "Peran Superadmin tidak dapat diubah atau ditetapkan melalui form ini."
		return validationErrors, nil
	}

	// Update field
	existingUser.Name = name
	existingUser.Email = email
	existingUser.Role = parsedRole
	existingUser.UpdatedAt = time.Now()

	if password != "" {
		hashedPassword, errHash := auth.HashPassword(password)
		if errHash != nil {
			log.Printf("Error hashing password for user update %s: %v", existingUser.Username, errHash)
			return nil, fmt.Errorf("gagal memproses password baru: %w", errHash)
		}
		existingUser.Password = hashedPassword // Ini akan di-pass ke repo untuk diupdate
	} else {
		existingUser.Password = "" // Pastikan repo tahu untuk tidak update password jika kosong
	}

	errRepo := s.userRepo.UpdateUser(existingUser)
	if errRepo != nil {
		if errors.Is(errRepo, models.ErrDuplicateEmail) {
			validationErrors["email"] = "Email sudah terdaftar untuk pengguna lain."
			return validationErrors, nil
		} else if errors.Is(errRepo, models.ErrUserNotFound) {
			// Ini seharusnya sudah ditangani oleh GetUserByID di atas, tapi sebagai jaring pengaman
			return nil, models.ErrUserNotFound
		}
		log.Printf("Repository error updating user '%s': %v", existingUser.Username, errRepo)
		return nil, fmt.Errorf("gagal memperbarui pengguna: %w", errRepo)
	}

	return nil, nil // Sukses
}

// GetUsersAvailableForTeamMembership mengambil user yang bisa ditambahkan sebagai anggota tim.
// Kriteria: Belum ada di tim manapun (baik sebagai member atau admin), dan memiliki peran yang sesuai (CRM, Telemarketing).
func (s *userService) GetUsersAvailableForTeamMembership(ctx context.Context, searchTerm string, limit int) ([]models.UserBasicInfo, error) {
	// Daftar peran yang diizinkan menjadi anggota tim biasa
	rolesToInclude := []models.UserRole{models.CRMRole, models.TelemarketingRole}

	// Kita tidak perlu excludeUserIDs di sini karena GetUsersNotYetInTeam sudah menyaring berdasarkan
	// apakah user sudah ada di team_members atau sebagai admin di teams.
	return s.teamRepo.GetUsersNotYetInTeam(nil, rolesToInclude, searchTerm, limit)
}

// GetUsersAvailableForTeamAdmin mengambil user yang bisa dijadikan admin tim.
// ... existing code ...
