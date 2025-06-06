package repository

import (
	"database/sql"
	"fmt"
	"followup/internal/auth" // Diperlukan untuk auth.UserClaims
	"followup/internal/models"
	"log"
	"strings"
	// "strconv" // Mungkin dibutuhkan nanti untuk konversi
)

// FollowupRepositoryInterface mendefinisikan interface untuk operasi database terkait data followup.
type FollowupRepositoryInterface interface {
	GetAll(filters models.FollowupFilters, page, limit int, currentUser *auth.UserClaims) ([]models.FollowupListItem, int, error)
	// Tambahkan metode lain yang dibutuhkan, misalnya untuk mendapatkan detail, update status, dll.
}

// followupRepository adalah implementasi dari FollowupRepositoryInterface.
type followupRepository struct {
	db *sql.DB
}

// NewFollowupRepository membuat instance baru dari followupRepository.
func NewFollowupRepository(db *sql.DB) FollowupRepositoryInterface {
	return &followupRepository{db: db}
}

// GetAll mengambil daftar data followup berdasarkan filter, paginasi, dan peran pengguna.
func (r *followupRepository) GetAll(filters models.FollowupFilters, page, limit int, currentUser *auth.UserClaims) ([]models.FollowupListItem, int, error) {
	var followups []models.FollowupListItem

	var baseQueryArgs []interface{}
	var countQueryArgs []interface{}

	baseQuery := `
		SELECT
			m.id AS id,
			m.username AS username,
			m.membership_email AS membership_email,
			m.phone_number AS phone_number,
			m.bank_name AS bank_name,
			m.account_no AS account_no,
			b.name AS brand_name, 
			m.status AS status, 
			crm.username AS crm_username,
			EXISTS (SELECT 1 FROM deposits d WHERE d.member_id = m.id AND d.status = 'pending') AS deposit_pending,
			m.created_at AS member_created_at
		FROM members m
		LEFT JOIN users crm ON m.crm_user_id = crm.id
		LEFT JOIN brands b ON m.brand_id = b.id
	`
	countBaseQuery := `
		SELECT COUNT(DISTINCT m.id) 
		FROM members m 
		LEFT JOIN users crm ON m.crm_user_id = crm.id 
		LEFT JOIN brands b ON m.brand_id = b.id
	`

	var conditions []string
	// Tambahkan kondisi dasar bahwa crm_user_id harus ada (member sudah ditangani)
	conditions = append(conditions, "m.crm_user_id IS NOT NULL")
	paramIdx := 1

	// Filter berdasarkan peran pengguna
	switch currentUser.Role {
	case models.SuperadminRole:
		// Tidak ada filter tambahan
	case models.AdminRole:
		// Admin: lihat data member dari tim CRM yang bekerja sama.
		// Asumsi: currentUser.UserID adalah ID admin yang ada di tabel teams.admin_user_id
		// Asumsi: member memiliki crm_user_id yang merupakan anggota dari tim admin tersebut.
		condition := fmt.Sprintf("m.crm_user_id IN (SELECT tm.user_id FROM team_members tm JOIN teams t ON tm.team_id = t.id WHERE t.admin_user_id = $%d)", paramIdx)
		conditions = append(conditions, condition)
		baseQueryArgs = append(baseQueryArgs, currentUser.UserID)
		countQueryArgs = append(countQueryArgs, currentUser.UserID)
		paramIdx++
	case models.CRMRole, models.TelemarketingRole:
		condition := fmt.Sprintf("m.crm_user_id = $%d", paramIdx)
		conditions = append(conditions, condition)
		baseQueryArgs = append(baseQueryArgs, currentUser.UserID) // Menggunakan UserID dari UserClaims
		countQueryArgs = append(countQueryArgs, currentUser.UserID)
		paramIdx++
	default:
		log.Printf("User %s with role %s has no specific data view rule for followup, returning empty.", currentUser.Username, currentUser.Role)
		return followups, 0, nil
	}

	// Filter berdasarkan SearchTerm
	if filters.SearchTerm != "" {
		condition := fmt.Sprintf("(m.username ILIKE $%d OR m.membership_email ILIKE $%d OR m.phone_number ILIKE $%d)", paramIdx, paramIdx+1, paramIdx+2)
		conditions = append(conditions, condition)
		searchTermLike := "%" + filters.SearchTerm + "%"
		baseQueryArgs = append(baseQueryArgs, searchTermLike, searchTermLike, searchTermLike)
		countQueryArgs = append(countQueryArgs, searchTermLike, searchTermLike, searchTermLike)
		paramIdx += 3
	}

	// Filter berdasarkan Status
	if len(filters.Status) > 0 {
		var statusPlaceholders []string
		for _, status := range filters.Status {
			statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", paramIdx))
			baseQueryArgs = append(baseQueryArgs, status)
			countQueryArgs = append(countQueryArgs, status)
			paramIdx++
		}
		conditions = append(conditions, fmt.Sprintf("m.status IN (%s)", strings.Join(statusPlaceholders, ",")))
	}

	// Filter berdasarkan BrandID
	if filters.BrandID > 0 {
		condition := fmt.Sprintf("m.brand_id = $%d", paramIdx)
		conditions = append(conditions, condition)
		baseQueryArgs = append(baseQueryArgs, filters.BrandID)
		countQueryArgs = append(countQueryArgs, filters.BrandID)
		paramIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	finalCountQuery := countBaseQuery + whereClause
	var totalRecords int
	err := r.db.QueryRow(finalCountQuery, countQueryArgs...).Scan(&totalRecords)
	if err != nil {
		log.Printf("Error counting followup data: %v. Query: [%s], Args: %v", err, finalCountQuery, countQueryArgs)
		return nil, 0, fmt.Errorf("gagal menghitung data followup: %w", err)
	}

	if totalRecords == 0 {
		return followups, 0, nil
	}

	finalQuery := baseQuery + whereClause + " ORDER BY m.created_at DESC"
	if limit > 0 {
		finalQuery += fmt.Sprintf(" LIMIT $%d", paramIdx)
		baseQueryArgs = append(baseQueryArgs, limit)
		paramIdx++
		if page > 0 {
			offset := (page - 1) * limit
			finalQuery += fmt.Sprintf(" OFFSET $%d", paramIdx)
			baseQueryArgs = append(baseQueryArgs, offset)
		}
	}

	rows, err := r.db.Query(finalQuery, baseQueryArgs...)
	if err != nil {
		log.Printf("Error querying followup data: %v. Query: [%s], Args: %v", err, finalQuery, baseQueryArgs)
		return nil, 0, fmt.Errorf("gagal mengambil data followup: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.FollowupListItem
		// Pastikan semua field di SELECT ada di Scan dan urutannya benar
		err := rows.Scan(
			&item.ID,
			&item.Username,
			&item.Email,
			&item.PhoneNumber,
			&item.BankName,
			&item.AccountNo,
			&item.BrandName,
			&item.Status,
			&item.CRMUsername,
			&item.DepositPending,
			&item.MemberCreatedAt,
			// &item.LastInteractionAt, // Jika ditambahkan di SELECT, tambahkan di sini juga
		)
		if err != nil {
			log.Printf("Error scanning followup row: %v", err)
			return nil, 0, fmt.Errorf("gagal memindai baris followup: %w", err)
		}
		followups = append(followups, item)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating followup rows: %v", err)
		return nil, 0, fmt.Errorf("error setelah iterasi baris followup: %w", err)
	}

	return followups, totalRecords, nil
}
