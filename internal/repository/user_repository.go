package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"followup/internal/models"
	"log"
	"strings"
)

// UserRepositoryInterface mendefinisikan interface untuk operasi database terkait User.
type UserRepositoryInterface interface {
	GetAllAssignableUsers() ([]models.UserBasicInfo, error)
	GetUserByUsername(username string) (*models.User, error)
	GetAllUsers(page, limit int, searchTerm, roleFilter string) ([]models.User, int, error)
	CreateUser(user *models.User) error
	SuperadminExists() (bool, error)
	DeleteUser(id int64) error
	GetUserByID(id int64) (*models.User, error)
	UpdateUser(user *models.User) error
}

// userRepository adalah implementasi dari UserRepositoryInterface.
type userRepository struct {
	db *sql.DB
}

// NewUserRepository membuat instance baru dari userRepository.
func NewUserRepository(db *sql.DB) UserRepositoryInterface {
	return &userRepository{db: db}
}

// GetAllAssignableUsers mengambil semua user (username dan nama) yang bisa ditugaskan sebagai CRM.
// Untuk saat ini, mengambil semua user. Bisa disesuaikan untuk filter berdasarkan role.
func (r *userRepository) GetAllAssignableUsers() ([]models.UserBasicInfo, error) {
	query := "SELECT username, name FROM users ORDER BY name ASC"

	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("Error querying assignable users: %v", err)
		return nil, fmt.Errorf("failed to query assignable users: %w", err)
	}
	defer rows.Close()

	var users []models.UserBasicInfo
	for rows.Next() {
		var user models.UserBasicInfo
		if err := rows.Scan(&user.Username, &user.Name); err != nil {
			log.Printf("Error scanning assignable user row: %v", err)
			return nil, fmt.Errorf("failed to scan assignable user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating assignable user rows: %v", err)
		return nil, fmt.Errorf("error iterating assignable user rows: %w", err)
	}

	return users, nil
}

// GetUserByUsername mengambil data user berdasarkan username.
func (r *userRepository) GetUserByUsername(username string) (*models.User, error) {
	query := "SELECT id, username, password, name, email, role, created_at, updated_at FROM users WHERE username = $1"
	user := &models.User{}
	err := r.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Password, // Ini akan menjadi PasswordHash
		&user.Name,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user dengan username '%s' tidak ditemukan", username) // Error yang lebih spesifik
		}
		log.Printf("Error querying user by username '%s': %v", username, err)
		return nil, fmt.Errorf("gagal mengambil user: %w", err)
	}
	return user, nil
}

// GetAllUsers mengambil daftar semua pengguna dengan paginasi, pencarian, dan filter.
func (r *userRepository) GetAllUsers(page, limit int, searchTerm, roleFilter string) ([]models.User, int, error) {
	var users []models.User
	var baseArgs []interface{}
	// var countArgs []interface{} // Deklarasi countArgs akan di-handle di bawah

	// Kondisi dasar untuk mengecualikan superadmin
	baseSuperAdminCondition := fmt.Sprintf(" role != '%s' ", models.SuperadminRole) // Menggunakan string langsung untuk superadmin

	// baseQueryStr tidak lagi digunakan karena finalBaseQuery sudah mencakup FROM dan JOIN clauses
	// baseQueryStr := "FROM users WHERE " + baseSuperAdminCondition
	countQueryStr := "SELECT COUNT(*) FROM users WHERE " + baseSuperAdminCondition

	conditionsForBase := ""
	conditionsForCount := "" // Pisahkan conditions untuk base dan count agar penomoran parameter benar

	baseParamIdx := 1
	// countParamIdx akan direset untuk countArgs

	if searchTerm != "" {
		// Untuk Base Query
		searchCondition := fmt.Sprintf(" AND (username ILIKE $%d OR name ILIKE $%d OR email ILIKE $%d) ", baseParamIdx, baseParamIdx+1, baseParamIdx+2)
		conditionsForBase += searchCondition
		baseArgs = append(baseArgs, "%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%")
		baseParamIdx += 3

		// Untuk Count Query - argumen akan dibuat terpisah
		// conditionsForCount akan menggunakan placeholder yang sama tapi dengan argumen berbeda
	}

	if roleFilter != "" {
		// Untuk Base Query
		roleCondition := fmt.Sprintf(" AND role = $%d ", baseParamIdx)
		conditionsForBase += roleCondition
		baseArgs = append(baseArgs, roleFilter)
		baseParamIdx++

		// Untuk Count Query - argumen akan dibuat terpisah
	}

	finalBaseQuery := `SELECT u.id, u.username, u.name, u.email, u.role, u.created_at, u.updated_at, t.name AS team_name 
	                     FROM users u
	                     LEFT JOIN team_members tm ON u.id = tm.user_id
	                     LEFT JOIN teams t ON tm.team_id = t.id
	                     WHERE ` + baseSuperAdminCondition + conditionsForBase

	// Bangun argumen dan kondisi untuk Count Query secara terpisah
	var countArgs []interface{}
	countParamIdxForCount := 1 // Reset parameter index untuk count query

	if searchTerm != "" {
		searchConditionCount := fmt.Sprintf(" AND (username ILIKE $%d OR name ILIKE $%d OR email ILIKE $%d) ", countParamIdxForCount, countParamIdxForCount+1, countParamIdxForCount+2)
		conditionsForCount += searchConditionCount
		countArgs = append(countArgs, "%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%")
		countParamIdxForCount += 3
	}
	if roleFilter != "" {
		roleConditionCount := fmt.Sprintf(" AND role = $%d ", countParamIdxForCount)
		conditionsForCount += roleConditionCount
		countArgs = append(countArgs, roleFilter)
		countParamIdxForCount++
	}
	finalCountQuery := countQueryStr + conditionsForCount

	// Hitung total record
	var totalRecords int
	err := r.db.QueryRow(finalCountQuery, countArgs...).Scan(&totalRecords)
	if err != nil {
		// Log query dan argumennya untuk debugging jika terjadi error
		log.Printf("Error counting users: %v. Query: [%s], Args: %v", err, finalCountQuery, countArgs)
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	if totalRecords == 0 {
		return users, 0, nil
	}

	finalBaseQuery += " ORDER BY created_at DESC"
	if limit > 0 {
		finalBaseQuery += fmt.Sprintf(" LIMIT $%d", len(baseArgs)+1)
		baseArgs = append(baseArgs, limit)
	}
	if page > 0 && limit > 0 {
		offset := (page - 1) * limit
		finalBaseQuery += fmt.Sprintf(" OFFSET $%d", len(baseArgs)+1)
		baseArgs = append(baseArgs, offset)
	}

	rows, err := r.db.Query(finalBaseQuery, baseArgs...)
	if err != nil {
		log.Printf("Error querying users: %v. Query: [%s], Args: %v", err, finalBaseQuery, baseArgs)
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Name, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt, &user.TeamName); err != nil {
			log.Printf("Error scanning user row: %v", err)
			return nil, 0, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating user rows: %v", err)
		return nil, 0, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, totalRecords, nil
}

// CreateUser menyimpan pengguna baru ke database.
// ID, CreatedAt, dan UpdatedAt pada struct user akan diisi setelah berhasil dibuat.
func (r *userRepository) CreateUser(user *models.User) error {
	query := `INSERT INTO users (username, password, name, email, role)
	          VALUES ($1, $2, $3, $4, $5)
	          RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(query,
		user.Username,
		user.Password, // Ini harus sudah berupa hash password
		user.Name,
		user.Email,
		user.Role.String(), // Menggunakan .String() untuk mendapatkan nilai string dari UserRole
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Cek apakah error karena duplikasi username atau email
		// Ini tergantung pada bagaimana driver database mengembalikan error. Untuk PostgreSQL, bisa seperti ini:
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			if strings.Contains(err.Error(), "users_username_key") {
				return models.ErrDuplicateUsername
			}
			if strings.Contains(err.Error(), "users_email_key") {
				return models.ErrDuplicateEmail
			}
		}
		log.Printf("Error creating user '%s': %v", user.Username, err)
		return fmt.Errorf("gagal membuat user: %w", err)
	}

	return nil
}

// SuperadminExists memeriksa apakah ada pengguna dengan peran superadmin.
func (r *userRepository) SuperadminExists() (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE role = $1)`
	err := r.db.QueryRow(query, models.SuperadminRole.String()).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // Bisa terjadi jika tabel kosong, tapi EXISTS harusnya mengembalikan false
			return false, nil
		}
		return false, fmt.Errorf("error checking if superadmin exists: %w", err)
	}
	return exists, nil
}

// DeleteUser menghapus pengguna berdasarkan ID.
func (r *userRepository) DeleteUser(id int64) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Printf("Error deleting user with ID %d: %v", id, err)
		return fmt.Errorf("gagal menghapus pengguna: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected after deleting user with ID %d: %v", id, err)
		return fmt.Errorf("gagal memeriksa hasil penghapusan pengguna: %w", err)
	}

	if rowsAffected == 0 {
		return models.ErrUserNotFound // Menggunakan error yang sudah ada jika tidak ada baris yang terpengaruh
	}

	return nil
}

// GetUserByID mengambil data pengguna berdasarkan ID.
func (r *userRepository) GetUserByID(id int64) (*models.User, error) {
	query := `SELECT id, username, name, email, role, created_at, updated_at FROM users WHERE id = $1`
	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Name,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrUserNotFound // Menggunakan error yang sudah ada
		}
		log.Printf("Error querying user by ID %d: %v", id, err)
		return nil, fmt.Errorf("gagal mengambil user berdasarkan ID: %w", err)
	}
	// Penting: Jangan scan user.Password di sini karena kita tidak ingin hash password terekspos
	// kecuali benar-benar dibutuhkan untuk operasi tertentu (yang seharusnya tidak untuk menampilkan form edit).
	return user, nil
}

// UpdateUser memperbarui data pengguna di database.
// Hanya field Name, Email, Role, dan Password (jika disediakan) yang akan diperbarui.
func (r *userRepository) UpdateUser(user *models.User) error {
	query := `UPDATE users SET name = $1, email = $2, role = $3, updated_at = NOW()`
	args := []interface{}{user.Name, user.Email, user.Role.String()}
	paramCount := 3

	if user.Password != "" { // Password adalah hash baru, atau string kosong jika tidak diubah
		paramCount++
		query += fmt.Sprintf(`, password = $%d`, paramCount)
		args = append(args, user.Password)
	}

	paramCount++
	query += fmt.Sprintf(` WHERE id = $%d`, paramCount)
	args = append(args, user.ID)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_key") {
			return models.ErrDuplicateEmail
		}
		log.Printf("Error updating user with ID %d: %v. Query: %s, Args: %v", user.ID, err, query, args)
		return fmt.Errorf("gagal memperbarui pengguna: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected after updating user with ID %d: %v", user.ID, err)
		return fmt.Errorf("gagal memeriksa hasil pembaruan pengguna: %w", err)
	}

	if rowsAffected == 0 {
		return models.ErrUserNotFound // Tidak ada user yang diperbarui, kemungkinan ID tidak ditemukan
	}

	return nil
}
