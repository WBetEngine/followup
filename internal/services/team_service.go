package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"followup/internal/models"
	"followup/internal/repository"
	"log"
	"strings"
	"time"
)

// TeamServiceInterface defines the interface for team management business logic.
type TeamServiceInterface interface {
	CreateTeam(ctx context.Context, name string, description sql.NullString, adminUserID int64) (*models.Team, map[string]string, error)
	GetTeamDetails(ctx context.Context, teamID int64) (*models.TeamWithDetails, error)
	ListTeams(ctx context.Context, searchTerm string, page, limit int) ([]models.Team, int, error)
	UpdateTeam(ctx context.Context, teamID int64, name *string, description *sql.NullString, newAdminUserID *int64) (*models.Team, map[string]string, error)
	DeleteTeam(ctx context.Context, teamID int64) error

	AddMemberToTeam(ctx context.Context, teamID, userID int64) (*models.TeamMemberDetail, map[string]string, error)
	RemoveMemberFromTeam(ctx context.Context, teamID, userID int64) error
	GetTeamMembers(ctx context.Context, teamID int64) ([]models.TeamMemberDetail, error)

	GetUsersAvailableForTeamMembership(ctx context.Context, searchTerm string, limit int) ([]models.UserBasicInfo, error)
	GetUsersAvailableForTeamAdmin(ctx context.Context, currentAdminIDToExclude int64, searchTerm string, limit int) ([]models.UserBasicInfo, error)
}

type teamService struct {
	teamRepo repository.TeamRepositoryInterface
	userRepo repository.UserRepositoryInterface // Untuk validasi user, dll.
}

// NewTeamService creates a new instance of TeamService.
func NewTeamService(teamRepo repository.TeamRepositoryInterface, userRepo repository.UserRepositoryInterface) TeamServiceInterface {
	return &teamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

// CreateTeam handles the business logic for creating a new team.
func (s *teamService) CreateTeam(ctx context.Context, name string, description sql.NullString, adminUserID int64) (*models.Team, map[string]string, error) {
	validationErrors := make(map[string]string)

	if strings.TrimSpace(name) == "" {
		validationErrors["name"] = "Nama tim tidak boleh kosong."
	}
	if len(name) > 100 {
		validationErrors["name"] = "Nama tim maksimal 100 karakter."
	}
	if description.Valid && len(description.String) > 500 {
		validationErrors["description"] = "Deskripsi tim maksimal 500 karakter."
	}
	if adminUserID <= 0 {
		validationErrors["admin_user_id"] = "Admin tim harus dipilih."
	}

	// Periksa apakah nama tim sudah ada
	existingTeamByName, err := s.teamRepo.GetTeamByName(name)
	if err != nil && !errors.Is(err, models.ErrTeamNotFound) {
		log.Printf("Error checking team name '%s': %v", name, err)
		return nil, nil, fmt.Errorf("gagal memvalidasi nama tim: %w", err)
	}
	if existingTeamByName != nil {
		validationErrors["name"] = "Nama tim sudah digunakan."
	}

	// Periksa apakah adminUserID valid dan bisa jadi admin
	adminUser, err := s.userRepo.GetUserByID(adminUserID)
	if err != nil || adminUser == nil {
		if errors.Is(err, models.ErrUserNotFound) || adminUser == nil {
			validationErrors["admin_user_id"] = "Pengguna calon admin tidak ditemukan."
		} else {
			log.Printf("Error fetching admin user ID %d: %v", adminUserID, err)
			return nil, nil, fmt.Errorf("gagal memvalidasi admin: %w", err)
		}
	} else {
		// Validasi peran admin (contoh: hanya role 'admin' atau 'superadmin' yang boleh jadi admin tim)
		if adminUser.Role != models.AdminRole && adminUser.Role != models.SuperadminRole {
			validationErrors["admin_user_id"] = fmt.Sprintf("Pengguna dengan peran '%s' tidak bisa menjadi admin tim.", adminUser.Role.String())
		}
		// Periksa apakah user ini sudah menjadi admin di tim lain
		isAlreadyAdmin, existingTeamID, errCheckAdmin := s.teamRepo.CheckUserIsAdmin(adminUserID)
		if errCheckAdmin != nil {
			log.Printf("Error checking if user %d is already an admin: %v", adminUserID, errCheckAdmin)
			return nil, nil, fmt.Errorf("gagal memeriksa status admin pengguna: %w", errCheckAdmin)
		}
		if isAlreadyAdmin {
			validationErrors["admin_user_id"] = fmt.Sprintf("Pengguna ini sudah menjadi admin untuk tim lain (ID Tim: %d).", existingTeamID)
		}
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors, nil
	}

	team := &models.Team{
		Name:        name,
		Description: description,
		AdminUserID: adminUserID,
	}

	teamID, err := s.teamRepo.CreateTeam(team)
	if err != nil {
		// Repository sudah melakukan logging, service bisa menambahkan konteks jika perlu
		return nil, nil, fmt.Errorf("gagal menyimpan tim baru: %w", err)
	}
	team.ID = teamID

	// Admin utama otomatis menjadi anggota tim tersebut juga.
	// Constraint unique user_id di team_members akan dicek di sini.
	_, errMember := s.teamRepo.AddMember(teamID, adminUserID)
	if errMember != nil {
		// Jika admin sudah ada di tim lain (karena constraint unique user_id), ini adalah masalah data inkonsisten
		// karena pengecekan isAlreadyAdmin di atas seharusnya sudah menangkap ini jika admin_user_id di teams unik per user.
		// Atau, jika admin tersebut ditambahkan sebagai member biasa di tim lain.
		// Untuk saat ini, kita log error ini dan lanjutkan. Atau bisa juga rollback pembuatan tim.
		// Skenario ini lebih mungkin terjadi jika validasi CheckUserIsAdmin tidak cukup ketat atau ada race condition.
		log.Printf("Warning/Error: Gagal menambahkan admin (UID: %d) sebagai anggota tim (TID: %d) setelah tim dibuat: %v", adminUserID, teamID, errMember)
		// Tergantung kebijakan, mungkin ingin mengembalikan error di sini dan menghapus tim yang baru dibuat.
		// Untuk sekarang, kita anggap pembuatan tim sukses, tapi anggota admin mungkin gagal ditambah jika sudah di tim lain.
		// Ini mengindikasikan bahwa CheckUserIsAdmin dan GetTeamMembershipByUserID mungkin perlu disinkronkan.
		// Jika GetTeamMembershipByUserID menunjukkan user sudah di tim, maka dia tidak bisa jadi admin tim BARU.
	}

	return team, nil, nil
}

func (s *teamService) GetTeamDetails(ctx context.Context, teamID int64) (*models.TeamWithDetails, error) {
	details, err := s.teamRepo.GetTeamByID(teamID) // GetTeamByID di repo sudah mengembalikan TeamWithDetails
	if err != nil {
		if errors.Is(err, models.ErrTeamNotFound) {
			return nil, models.ErrTeamNotFound
		}
		log.Printf("Error getting team details for ID %d from repository: %v", teamID, err)
		return nil, fmt.Errorf("gagal mengambil detail tim: %w", err)
	}
	return details, nil
}

func (s *teamService) ListTeams(ctx context.Context, searchTerm string, page, limit int) ([]models.Team, int, error) {
	return s.teamRepo.GetTeamsPage(searchTerm, page, limit)
}

func (s *teamService) UpdateTeam(ctx context.Context, teamID int64, name *string, description *sql.NullString, newAdminUserID *int64) (*models.Team, map[string]string, error) {
	validationErrors := make(map[string]string)

	existingTeamWithDetails, err := s.teamRepo.GetTeamByID(teamID)
	if err != nil {
		if errors.Is(err, models.ErrTeamNotFound) {
			return nil, nil, models.ErrTeamNotFound
		}
		log.Printf("UpdateTeam: Error fetching team ID %d for update: %v", teamID, err)
		return nil, nil, fmt.Errorf("gagal mengambil data tim untuk pembaruan: %w", err)
	}

	teamToUpdate := existingTeamWithDetails.Team // Ambil struct Team dari TeamWithDetails

	hasChanges := false
	if name != nil && *name != teamToUpdate.Name {
		if strings.TrimSpace(*name) == "" {
			validationErrors["name"] = "Nama tim tidak boleh kosong."
		} else if len(*name) > 100 {
			validationErrors["name"] = "Nama tim maksimal 100 karakter."
		} else {
			// Periksa keunikan nama jika diubah
			otherTeam, errCheckName := s.teamRepo.GetTeamByName(*name)
			if errCheckName != nil && !errors.Is(errCheckName, models.ErrTeamNotFound) {
				log.Printf("UpdateTeam: Error checking new team name '%s': %v", *name, errCheckName)
				return nil, nil, fmt.Errorf("gagal memvalidasi nama tim baru: %w", errCheckName)
			}
			if otherTeam != nil && otherTeam.ID != teamID {
				validationErrors["name"] = "Nama tim sudah digunakan."
			}
			teamToUpdate.Name = *name
			hasChanges = true
		}
	}

	if description != nil {
		// Cek apakah deskripsi benar-benar berubah (termasuk dari NULL ke non-NULL atau sebaliknya)
		if (description.Valid != teamToUpdate.Description.Valid) || (description.Valid && teamToUpdate.Description.Valid && description.String != teamToUpdate.Description.String) {
			if description.Valid && len(description.String) > 500 {
				validationErrors["description"] = "Deskripsi tim maksimal 500 karakter."
			} else {
				teamToUpdate.Description = *description
				hasChanges = true
			}
		}
	}

	currentAdminID := teamToUpdate.AdminUserID
	if newAdminUserID != nil && *newAdminUserID != currentAdminID {
		if *newAdminUserID <= 0 {
			validationErrors["admin_user_id"] = "ID Admin baru tidak valid."
		} else {
			adminUser, errAdmin := s.userRepo.GetUserByID(*newAdminUserID)
			if errAdmin != nil || adminUser == nil {
				validationErrors["admin_user_id"] = "Pengguna calon admin baru tidak ditemukan."
			} else {
				if adminUser.Role != models.AdminRole && adminUser.Role != models.SuperadminRole {
					validationErrors["admin_user_id"] = fmt.Sprintf("Pengguna '%s' dengan peran '%s' tidak bisa menjadi admin tim.", adminUser.Username, adminUser.Role.String())
				}
				isAlreadyAdmin, otherTeamID, errCheck := s.teamRepo.CheckUserIsAdmin(*newAdminUserID)
				if errCheck != nil {
					return nil, nil, fmt.Errorf("gagal memeriksa status admin pengguna baru: %w", errCheck)
				}
				if isAlreadyAdmin && otherTeamID != teamID {
					validationErrors["admin_user_id"] = "Pengguna ini sudah menjadi admin untuk tim lain."
				}
				teamToUpdate.AdminUserID = *newAdminUserID
				hasChanges = true
			}
		}
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors, nil
	}

	if !hasChanges {
		return &teamToUpdate, nil, nil // Tidak ada perubahan, kembalikan data yang ada
	}

	errUpdate := s.teamRepo.UpdateTeam(&teamToUpdate)
	if errUpdate != nil {
		return nil, nil, fmt.Errorf("gagal memperbarui tim: %w", errUpdate)
	}

	// Jika admin berubah, kelola keanggotaan admin lama dan baru.
	if newAdminUserID != nil && *newAdminUserID != currentAdminID {
		// 1. Hapus admin lama dari keanggotaan tim ini (jika ada)
		// Tidak masalah jika admin lama tidak ditemukan sebagai member, bisa jadi memang tidak pernah/sudah dihapus.
		errRemoveOld := s.teamRepo.RemoveMember(teamID, currentAdminID)
		if errRemoveOld != nil {
			// Log warning, tapi jangan gagalkan seluruh operasi update tim hanya karena ini
			log.Printf("Warning: Gagal menghapus admin lama (UID: %d) dari anggota tim (TID: %d) saat update admin: %v", currentAdminID, teamID, errRemoveOld)
		}

		// 2. Tambahkan admin baru sebagai anggota tim ini
		// Ini penting agar konsisten dengan CreateTeam dimana admin juga adalah member.
		// teamRepo.AddMember harusnya idempotent atau menangani jika sudah ada (meski idealnya tidak terjadi jika logika benar)
		// atau memiliki constraint UNIQUE di DB pada (team_id, user_id).
		_, errAddNew := s.teamRepo.AddMember(teamID, *newAdminUserID)
		if errAddNew != nil {
			// Jika gagal menambahkan admin baru sebagai anggota, ini bisa jadi masalah.
			// Misalnya, jika user_id sudah ada di team_members untuk tim lain dan ada constraint global unique user_id.
			// Validasi sebelumnya (isAlreadyAdmin) seharusnya sudah mencegah user menjadi admin di >1 tim.
			// Validasi GetTeamMembershipByUserID di AddMemberToTeam (yang akan kita perbaiki) akan mencegah user jadi member di >1 tim.
			log.Printf("Warning/Error: Gagal menambahkan admin baru (UID: %d) sebagai anggota tim (TID: %d) setelah update admin: %v", *newAdminUserID, teamID, errAddNew)
			// Pertimbangkan apakah ini harus menjadi error fatal yang mengembalikan error ke user.
			// Untuk saat ini, kita log dan lanjutkan.
		}
	}

	// Ambil data tim yang sudah diperbarui (termasuk admin username baru jika berubah)
	updatedTeamDetails, errFetch := s.teamRepo.GetTeamByID(teamID)
	if errFetch != nil {
		log.Printf("Warning: Gagal mengambil detail tim setelah update (TID: %d): %v", teamID, errFetch)
		return &teamToUpdate, nil, nil // Kembalikan data sebelum fetch jika gagal
	}

	return &updatedTeamDetails.Team, nil, nil
}

func (s *teamService) DeleteTeam(ctx context.Context, teamID int64) error {
	// Validasi apakah tim ada sebelum menghapus
	_, err := s.teamRepo.GetTeamByID(teamID)
	if err != nil {
		if errors.Is(err, models.ErrTeamNotFound) {
			return models.ErrTeamNotFound
		}
		return fmt.Errorf("gagal memeriksa tim sebelum penghapusan: %w", err)
	}
	// Repository akan menghapus tim. Anggota akan terhapus karena ON DELETE CASCADE.
	return s.teamRepo.DeleteTeam(teamID)
}

func (s *teamService) AddMemberToTeam(ctx context.Context, teamID, userID int64) (*models.TeamMemberDetail, map[string]string, error) {
	validationErrors := make(map[string]string)

	// 1. Validasi dasar: Tim ada, User ada, User punya peran yang sesuai
	team, err := s.teamRepo.GetTeamByID(teamID)
	if err != nil {
		if errors.Is(err, models.ErrTeamNotFound) {
			// Tidak perlu error validasi spesifik, handler akan return 404 jika team tidak ditemukan via URL
			return nil, nil, models.ErrTeamNotFound
		}
		log.Printf("AddMemberToTeam: Error fetching team ID %d: %v", teamID, err)
		return nil, nil, fmt.Errorf("gagal mengambil data tim: %w", err)
	}

	userToAdd, err := s.userRepo.GetUserByID(userID)
	if err != nil || userToAdd == nil {
		if errors.Is(err, models.ErrUserNotFound) || userToAdd == nil {
			validationErrors["user_id"] = "Pengguna yang akan ditambahkan tidak ditemukan."
		} else {
			log.Printf("AddMemberToTeam: Error fetching user ID %d: %v", userID, err)
			return nil, nil, fmt.Errorf("gagal mengambil data pengguna: %w", err)
		}
	} else {
		// Peran yang diizinkan untuk menjadi anggota tim (bukan admin)
		if userToAdd.Role != models.CRMRole && userToAdd.Role != models.TelemarketingRole {
			validationErrors["user_id"] = fmt.Sprintf("Pengguna dengan peran '%s' tidak dapat ditambahkan sebagai anggota tim.", userToAdd.Role.String())
		}
	}

	// 2. Validasi Aturan Bisnis:
	//    - User tidak boleh admin tim ini (sudah otomatis jika admin == userToAdd.ID)
	//    - User tidak boleh admin tim LAIN
	//    - User tidak boleh sudah menjadi anggota tim LAIN
	//    - User tidak boleh sudah menjadi anggota tim INI (akan dicegah oleh AddMember jika ada constraint UNIQUE (team_id, user_id))

	if userToAdd != nil { // Hanya lanjutkan jika user valid
		if team.Team.AdminUserID == userID {
			validationErrors["user_id"] = "Pengguna ini sudah menjadi admin untuk tim ini."
		}

		// Cek apakah user ini adalah admin di tim lain
		isOtherAdmin, otherAdminTeamID, errCheckAdmin := s.teamRepo.CheckUserIsAdmin(userID)
		if errCheckAdmin != nil {
			log.Printf("AddMemberToTeam: Error checking if user %d is admin: %v", userID, errCheckAdmin)
			return nil, nil, fmt.Errorf("gagal memeriksa status admin pengguna: %w", errCheckAdmin)
		}
		if isOtherAdmin { // Tidak perlu cek otherAdminTeamID != teamID karena admin tim ini sudah dicek di atas
			validationErrors["user_id"] = fmt.Sprintf("Pengguna ini sudah menjadi admin untuk tim lain (ID Tim: %d). Tidak dapat ditambahkan sebagai anggota.", otherAdminTeamID)
		}

		// Cek apakah user ini sudah menjadi anggota di tim lain (atau tim ini, yang akan ditangani oleh repo.AddMember)
		// GetTeamMembershipByUserID akan mengembalikan ErrTeamMemberNotFound jika user bukan member di tim manapun.
		membership, errCheckMembership := s.teamRepo.GetTeamMembershipByUserID(userID)
		if errCheckMembership != nil && !errors.Is(errCheckMembership, models.ErrMemberNotFoundInTeam) {
			log.Printf("AddMemberToTeam: Error checking team membership for user %d: %v", userID, errCheckMembership)
			return nil, nil, fmt.Errorf("gagal memeriksa keanggotaan tim pengguna: %w", errCheckMembership)
		}
		if membership != nil { // Jika ditemukan membership
			if membership.TeamID == teamID {
				// Ini berarti user sudah menjadi anggota tim ini.
				// teamRepo.AddMember juga akan menangani ini (misalnya dengan error jika ada constraint unique).
				// Bisa ditambahkan pesan error spesifik di sini atau biarkan repository yang menangani.
				validationErrors["user_id"] = "Pengguna ini sudah menjadi anggota di tim ini."
			} else {
				// User adalah anggota tim lain.
				validationErrors["user_id"] = fmt.Sprintf("Pengguna ini sudah menjadi anggota di tim lain (ID Tim: %d).", membership.TeamID)
			}
		}
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors, nil
	}

	// Jika semua validasi lolos, tambahkan anggota
	var addMemberErr error // Deklarasikan variabel error baru untuk menghindari konflik scope
	_, addMemberErr = s.teamRepo.AddMember(teamID, userID)
	if addMemberErr != nil {
		if errors.Is(addMemberErr, models.ErrUserAlreadyInTeam) {
			validationErrors["user_id"] = "Pengguna ini sudah menjadi anggota di tim ini."
			return nil, validationErrors, nil
		}
		// Tangani error lain dari AddMember, e.g., DB error, fk constraint violation
		log.Printf("AddMemberToTeam: Error adding member (UID: %d) to team (TID: %d): %v", userID, teamID, addMemberErr)
		return nil, nil, fmt.Errorf("gagal menambahkan anggota ke tim: %w", addMemberErr)
	}

	// Ambil detail member yang baru ditambahkan untuk dikembalikan
	// Ini bisa dilakukan dengan query baru atau jika AddMember mengembalikan cukup info.
	// Untuk sekarang, kita buat asumsi perlu query baru.
	// Namun, TeamRepository.GetMembersByTeamID mengembalikan []models.TeamMemberDetail
	// Kita perlu satu fungsi untuk GetMemberByTeamAndUser atau GetMemberByID
	// Untuk sementara, kita kembalikan struct dasar.

	// Idealnya, kita ingin mengembalikan detail user yang baru ditambahkan.
	// Kita sudah punya userToAdd. Kita bisa buat TeamMemberDetail dari situ.
	newMemberDetail := &models.TeamMemberDetail{
		TeamID:       teamID,
		UserID:       userID,
		Username:     userToAdd.Username,
		UserFullName: userToAdd.Name,
		UserRole:     userToAdd.Role,
		JoinedAt:     time.Now(), // Sebaiknya ini dari DB jika AddMember mengisinya
		// TeamName dan TeamAdminUsername tidak relevan di sini, atau bisa diisi jika perlu
	}
	// Jika AddMember tidak mengisi CreatedAt, maka JoinedAt di atas adalah perkiraan.
	// Lebih baik jika GetMembersByTeamID atau GetTeamMemberByID dipanggil.
	// Tapi untuk menghindari kompleksitas, kita gunakan ini dulu.

	return newMemberDetail, nil, nil
}

func (s *teamService) RemoveMemberFromTeam(ctx context.Context, teamID, userID int64) error {
	// 1. Cek apakah tim ada
	team, errTeam := s.teamRepo.GetTeamByID(teamID) // Validasi tim ada
	if errTeam != nil {
		if errors.Is(errTeam, models.ErrTeamNotFound) {
			return models.ErrTeamNotFound
		}
		return fmt.Errorf("gagal memeriksa tim: %w", errTeam)
	}

	// 2. Cek apakah user adalah admin utama tim ini
	if team.Team.AdminUserID == userID {
		return models.ErrCannotRemoveAdmin
	}

	// 3. Cek apakah user ada di tim tersebut sebelum menghapus
	// GetTeamMembershipByUserID bisa digunakan, tapi repo.RemoveMember sudah melakukan pengecekan ini (mengembalikan ErrMemberNotFoundInTeam)

	errRemove := s.teamRepo.RemoveMember(teamID, userID)
	if errRemove != nil {
		if errors.Is(errRemove, models.ErrMemberNotFoundInTeam) {
			return models.ErrMemberNotFoundInTeam
		}
		// ErrCannotRemoveAdmin juga bisa dilempar dari repo jika ada logika dobel.
		if errors.Is(errRemove, models.ErrCannotRemoveAdmin) {
			return models.ErrCannotRemoveAdmin
		}
		return fmt.Errorf("gagal menghapus anggota dari tim: %w", errRemove)
	}
	return nil
}

func (s *teamService) GetTeamMembers(ctx context.Context, teamID int64) ([]models.TeamMemberDetail, error) {
	// Pastikan tim ada terlebih dahulu
	_, errTeam := s.teamRepo.GetTeamByID(teamID)
	if errTeam != nil {
		if errors.Is(errTeam, models.ErrTeamNotFound) {
			return nil, models.ErrTeamNotFound
		}
		return nil, fmt.Errorf("gagal memeriksa tim sebelum mengambil anggota: %w", errTeam)
	}
	return s.teamRepo.GetMembersByTeamID(teamID)
}

// GetUsersAvailableForTeamMembership mengambil user yang bisa ditambahkan sebagai anggota tim.
// Kriteria: Belum ada di tim manapun, dan memiliki peran yang sesuai (CRM, Telemarketing).
func (s *teamService) GetUsersAvailableForTeamMembership(ctx context.Context, searchTerm string, limit int) ([]models.UserBasicInfo, error) {
	// Daftar peran yang diizinkan menjadi anggota tim biasa
	rolesToInclude := []models.UserRole{models.CRMRole, models.TelemarketingRole} // HANYA CRM DAN TELEMARKETING

	// Kita tidak perlu excludeUserIDs di sini karena GetUsersNotYetInTeam sudah menyaring berdasarkan
	// apakah user sudah ada di team_members atau sebagai admin di teams.
	return s.teamRepo.GetUsersNotYetInTeam(nil, rolesToInclude, searchTerm, limit)
}

// GetUsersAvailableForTeamAdmin mengambil user yang bisa dijadikan admin tim.
// Kriteria: Belum menjadi admin di tim lain, dan memiliki peran yang sesuai (Admin, Superadmin).
func (s *teamService) GetUsersAvailableForTeamAdmin(ctx context.Context, currentAdminIDToExclude int64, searchTerm string, limit int) ([]models.UserBasicInfo, error) {
	var excludeIDs []int64
	if currentAdminIDToExclude > 0 {
		excludeIDs = append(excludeIDs, currentAdminIDToExclude)
	}
	return s.teamRepo.GetPotentialAdmins(excludeIDs, searchTerm, limit)
}
