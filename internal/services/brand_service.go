package services

import (
	"followup/internal/models"
	"followup/internal/repository"
)

// BrandService defines the interface for brand related services.
// It acts as an intermediary between handlers and the brand repository.
// Business logic related to brands will be placed here.
// For example, complex validation or orchestrating calls to multiple repositories.
type BrandService interface {
	CreateBrand(name string) (*models.Brand, error)
	GetAllBrands(searchTerm string) ([]models.Brand, error)
	GetBrandByID(id int64) (*models.Brand, error)
	UpdateBrand(id int64, name string) (*models.Brand, error)
	DeleteBrand(id int64) error
	GetBrandWithMemberCount(id int64) (*models.Brand, int, error) // Combines GetBrandByID and GetMemberCountByBrandName
	GetAllBrandsWithMemberCount(searchTerm string) ([]map[string]interface{}, error)
}

type brandService struct {
	repo repository.BrandRepository
}

// NewBrandService creates a new BrandService.
func NewBrandService(repo repository.BrandRepository) BrandService {
	return &brandService{repo: repo}
}

func (s *brandService) CreateBrand(name string) (*models.Brand, error) {
	// Di sini bisa ditambahkan validasi nama brand jika perlu
	brand := &models.Brand{
		Name: name,
	}
	_, err := s.repo.CreateBrand(brand)
	if err != nil {
		return nil, err
	}
	return brand, nil
}

func (s *brandService) GetAllBrands(searchTerm string) ([]models.Brand, error) {
	return s.repo.GetAllBrands(searchTerm)
}

func (s *brandService) GetBrandByID(id int64) (*models.Brand, error) {
	return s.repo.GetBrandByID(id)
}

func (s *brandService) UpdateBrand(id int64, name string) (*models.Brand, error) {
	brand, err := s.repo.GetBrandByID(id)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, nil // Atau error brand tidak ditemukan
	}
	brand.Name = name
	err = s.repo.UpdateBrand(brand)
	if err != nil {
		return nil, err
	}
	return brand, nil
}

func (s *brandService) DeleteBrand(id int64) error {
	// Mungkin ada logika tambahan di sini, misal cek apakah brand masih digunakan
	return s.repo.DeleteBrand(id)
}

func (s *brandService) GetBrandWithMemberCount(id int64) (*models.Brand, int, error) {
	brand, err := s.repo.GetBrandByID(id)
	if err != nil {
		return nil, 0, err
	}
	if brand == nil {
		return nil, 0, nil // Brand not found
	}

	count, err := s.repo.GetMemberCountByBrandName(brand.Name)
	if err != nil {
		return brand, 0, err
	}
	return brand, count, nil
}

func (s *brandService) GetAllBrandsWithMemberCount(searchTerm string) ([]map[string]interface{}, error) {
	brands, err := s.repo.GetAllBrands(searchTerm)
	if err != nil {
		return nil, err
	}

	result := []map[string]interface{}{}
	for _, brand := range brands {
		count, err := s.repo.GetMemberCountByBrandName(brand.Name)
		// Jika ada error saat mengambil count, kita bisa set count ke 0 atau handle errornya
		// Untuk saat ini, kita abaikan error count per brand agar daftar brand tetap tampil
		if err != nil {
			count = 0 // Atau log errornya
		}
		result = append(result, map[string]interface{}{
			"ID":          brand.ID,
			"Name":        brand.Name,
			"CreatedAt":   brand.CreatedAt,
			"UpdatedAt":   brand.UpdatedAt,
			"MemberCount": count,
		})
	}
	return result, nil
}
