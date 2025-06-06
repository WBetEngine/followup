package services

import (
	"followup/internal/auth"
	"followup/internal/models"
	"followup/internal/repository"
	"log"
	// "fmt" // Mungkin dibutuhkan nanti
)

// FollowupServiceInterface mendefinisikan interface untuk layanan terkait data followup.
type FollowupServiceInterface interface {
	GetAllFollowups(filters models.FollowupFilters, page, limit int, currentUser *auth.UserClaims) ([]models.FollowupListItem, int, error)
	// Tambahkan metode layanan lain jika diperlukan
}

// followupService adalah implementasi dari FollowupServiceInterface.
type followupService struct {
	followupRepo repository.FollowupRepositoryInterface
	// Mungkin perlu service lain, misalnya User Service untuk detail CRM jika tidak langsung dari repo
}

// NewFollowupService membuat instance baru dari followupService.
func NewFollowupService(followupRepo repository.FollowupRepositoryInterface) FollowupServiceInterface {
	return &followupService{
		followupRepo: followupRepo,
	}
}

// GetAllFollowups mengambil daftar data followup dengan menerapkan logika bisnis jika ada.
func (s *followupService) GetAllFollowups(filters models.FollowupFilters, page, limit int, currentUser *auth.UserClaims) ([]models.FollowupListItem, int, error) {
	log.Printf("Service: GetAllFollowups called by User: %s, Role: %s, UserID: %d, Filters: %+v, Page: %d, Limit: %d",
		currentUser.Username, currentUser.Role, currentUser.UserID, filters, page, limit)

	followups, totalRecords, err := s.followupRepo.GetAll(filters, page, limit, currentUser)
	if err != nil {
		log.Printf("Service: Error getting followups from repository: %v", err)
		return nil, 0, err
	}

	log.Printf("Service: Successfully retrieved %d followups, TotalRecords: %d", len(followups), totalRecords)
	return followups, totalRecords, nil
}
