package services

import (
	"database/sql"
	"errors"
	"followup/internal/models"
	"followup/internal/repository"
	"log"
	"math"
)

// MemberService mendefinisikan interface untuk layanan terkait member.
type MemberService interface {
	// GetAllMembers mengambil daftar member dengan paginasi, pencarian, filter, dan sorting.
	// Mengembalikan slice MemberData, total record, total halaman, dan error.
	GetAllMembers(page, limit int, searchTerm, filterBrand, filterStatus, sortBy, sortOrder string) ([]models.MemberData, int, int, error)
	// CreateMember membuat member baru.
	CreateMember(member *models.MemberData) (int, error) // ID member baru dan error
	// GetMemberByID mengambil member berdasarkan ID.
	GetMemberByID(memberID int) (*models.MemberData, error)
	// UpdateMemberPhoneNumber memperbarui nomor telepon member.
	UpdateMemberPhoneNumber(memberID int, newPhoneNumber string) error
	// UpdateMemberCRM memperbarui CRM yang ditugaskan untuk member.
	UpdateMemberCRM(memberID int, crmUsernameInput *string) error
}

type memberService struct {
	repo     repository.MemberRepositoryInterface
	userRepo repository.UserRepositoryInterface
}

// NewMemberService membuat instance baru dari MemberService.
func NewMemberService(repo repository.MemberRepositoryInterface, userRepo repository.UserRepositoryInterface) MemberService {
	return &memberService{repo: repo, userRepo: userRepo}
}

func (s *memberService) GetAllMembers(page, limit int, searchTerm, filterBrand, filterStatus, sortBy, sortOrder string) ([]models.MemberData, int, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50 // Default limit
	}

	members, totalRecords, err := s.repo.GetAllMembers(page, limit, searchTerm, filterBrand, filterStatus, sortBy, sortOrder)
	if err != nil {
		return nil, 0, 0, err
	}

	totalPages := 0
	if totalRecords > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(limit)))
	}

	return members, totalRecords, totalPages, nil
}

// CreateMember membuat member baru melalui repository.
func (s *memberService) CreateMember(member *models.MemberData) (int, error) {
	if member.CRMInfo != nil && *member.CRMInfo != "" && !member.CRMUserID.Valid {
		user, err := s.userRepo.GetUserByUsername(*member.CRMInfo)
		if err == nil && user != nil {
			if user.Role == models.CRMRole || user.Role == models.TelemarketingRole {
				member.CRMUserID = sql.NullInt64{Int64: user.ID, Valid: true}
				log.Printf("MemberService.CreateMember: CRMUserID diisi otomatis untuk CRMInfo: %s, UserID: %d", *member.CRMInfo, user.ID)
			} else {
				log.Printf("MemberService.CreateMember: User %s ditemukan tetapi bukan CRM/Telemarketing (Peran: %s)", *member.CRMInfo, user.Role)
			}
		} else {
			log.Printf("MemberService.CreateMember: Gagal menemukan user CRM dengan username: %s, Error: %v", *member.CRMInfo, err)
		}
	}
	return s.repo.CreateMember(member)
}

// GetMemberByID mengambil member berdasarkan ID melalui repository.
func (s *memberService) GetMemberByID(memberID int) (*models.MemberData, error) {
	return s.repo.GetMemberByID(memberID)
}

// UpdateMemberPhoneNumber memperbarui nomor telepon member melalui repository.
func (s *memberService) UpdateMemberPhoneNumber(memberID int, newPhoneNumber string) error {
	return s.repo.UpdateMemberPhoneNumber(memberID, newPhoneNumber)
}

// UpdateMemberCRM memperbarui CRM untuk member melalui repository.
func (s *memberService) UpdateMemberCRM(memberID int, crmUsernameInput *string) error {
	var crmInfoForRepo sql.NullString
	var crmUserIDForRepo sql.NullInt64

	if crmUsernameInput != nil && *crmUsernameInput != "" {
		crmUsername := *crmUsernameInput
		user, err := s.userRepo.GetUserByUsername(crmUsername)
		if err != nil {
			log.Printf("Error UpdateMemberCRM: Gagal mencari user CRM '%s': %v", crmUsername, err)
			return errors.New("user CRM tidak ditemukan")
		}

		if user.Role != models.CRMRole && user.Role != models.TelemarketingRole {
			log.Printf("Error UpdateMemberCRM: User '%s' bukan CRM atau Telemarketing (Peran: %s)", crmUsername, user.Role)
			return errors.New("pengguna yang dipilih bukan CRM atau Telemarketing")
		}

		crmInfoForRepo = sql.NullString{String: user.Username, Valid: true}
		crmUserIDForRepo = sql.NullInt64{Int64: user.ID, Valid: true}
	} else {
		crmInfoForRepo = sql.NullString{Valid: false}
		crmUserIDForRepo = sql.NullInt64{Valid: false}
	}

	return s.repo.UpdateMemberCRM(memberID, crmInfoForRepo, crmUserIDForRepo)
}
