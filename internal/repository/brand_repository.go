package repository

import (
	"database/sql"
	"followup/internal/models"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

type BrandRepository interface {
	CreateBrand(brand *models.Brand) (int64, error)
	GetAllBrands(searchTerm string) ([]models.Brand, error)
	GetBrandByID(id int64) (*models.Brand, error)
	UpdateBrand(brand *models.Brand) error
	DeleteBrand(id int64) error
	GetMemberCountByBrandName(brandName string) (int, error)
}

type brandRepository struct {
	db *sql.DB
}

func NewBrandRepository(db *sql.DB) BrandRepository {
	return &brandRepository{db: db}
}

func (r *brandRepository) CreateBrand(brand *models.Brand) (int64, error) {
	query := `INSERT INTO brands (name, created_at, updated_at) VALUES ($1, $2, $3) RETURNING id`
	brand.CreatedAt = time.Now()
	brand.UpdatedAt = time.Now()
	err := r.db.QueryRow(query, brand.Name, brand.CreatedAt, brand.UpdatedAt).Scan(&brand.ID)
	if err != nil {
		return 0, err
	}
	return brand.ID, nil
}

func (r *brandRepository) GetAllBrands(searchTerm string) ([]models.Brand, error) {
	query := `SELECT id, name, created_at, updated_at FROM brands`
	args := []interface{}{}
	if searchTerm != "" {
		query += " WHERE name ILIKE $1"
		args = append(args, "%"+searchTerm+"%")
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	brands := []models.Brand{}
	for rows.Next() {
		var brand models.Brand
		if err := rows.Scan(&brand.ID, &brand.Name, &brand.CreatedAt, &brand.UpdatedAt); err != nil {
			return nil, err
		}
		brands = append(brands, brand)
	}
	return brands, nil
}

func (r *brandRepository) GetBrandByID(id int64) (*models.Brand, error) {
	query := `SELECT id, name, created_at, updated_at FROM brands WHERE id = $1`
	brand := &models.Brand{}
	err := r.db.QueryRow(query, id).Scan(&brand.ID, &brand.Name, &brand.CreatedAt, &brand.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Atau error khusus "not found"
		}
		return nil, err
	}
	return brand, nil
}

func (r *brandRepository) UpdateBrand(brand *models.Brand) error {
	query := `UPDATE brands SET name = $1, updated_at = $2 WHERE id = $3`
	brand.UpdatedAt = time.Now()
	_, err := r.db.Exec(query, brand.Name, brand.UpdatedAt, brand.ID)
	return err
}

func (r *brandRepository) DeleteBrand(id int64) error {
	query := `DELETE FROM brands WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *brandRepository) GetMemberCountByBrandName(brandName string) (int, error) {
	query := `SELECT COUNT(*) FROM members WHERE brand_name = $1`
	var count int
	err := r.db.QueryRow(query, brandName).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
