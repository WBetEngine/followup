package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"followup/internal/models"
	"log"
	"strings"
	"time"

	"github.com/lib/pq" // PostgreSQL driver
)

// MemberRepositoryInterface mendefinisikan interface untuk operasi database terkait Member.
// Mencakup semua fungsi yang dibutuhkan oleh MemberService dan operasi lainnya.
type MemberRepositoryInterface interface {
	CreateMember(member *models.MemberData) (int, error)
	GetMemberByID(memberID int) (*models.MemberData, error)
	GetAllMembers(page, limit int, searchTerm, filterBrand, filterStatus, sortBy, sortOrder string) ([]models.MemberData, int, error)
	UpdateMemberPhoneNumber(memberID int, newPhoneNumber string) error
	UpdateMemberCRM(memberID int, crmInfo sql.NullString, crmUserID sql.NullInt64) error
	MemberExists(phoneNumber, brandName string) (bool, int, error) // bool, existingMemberID, error
	BulkInsertMembers(members []models.MemberData) (int, error)
}

// memberRepository adalah implementasi dari MemberRepositoryInterface.
type memberRepository struct {
	db *sql.DB
}

// NewMemberRepository membuat instance baru dari memberRepository.
func NewMemberRepository(db *sql.DB) MemberRepositoryInterface {
	return &memberRepository{db: db}
}

// CreateMember menyimpan member baru ke database.
func (r *memberRepository) CreateMember(member *models.MemberData) (int, error) {
	query := `
		INSERT INTO members (
			username, email, phone_number, bank_name, account_name, account_no, 
			brand_name, status, membership_status, ip_address, last_login, join_date, 
			saldo, membership_email, turnover, win_loss, points, referral, uplink, 
			crm_info, crm_user_id, created_at, updated_at, uploaded_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 
			$14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		) RETURNING id
	`
	now := time.Now()

	var joinDateToInsert interface{}
	if member.JoinDate != nil && *member.JoinDate != "" {
		joinDateToInsert = *member.JoinDate
	} else {
		joinDateToInsert = nil
	}

	var lastLoginToInsert interface{}
	if member.LastLogin != nil && *member.LastLogin != "" {
		lastLoginToInsert = *member.LastLogin
	} else {
		lastLoginToInsert = nil
	}

	var memberID int
	err := r.db.QueryRow(
		query,
		member.Username,
		NewNullString(member.Email),
		member.PhoneNumber,
		NewNullString(member.BankName),
		NewNullString(member.AccountName),
		NewNullString(member.AccountNo),
		member.BrandName,
		NewNullString(member.Status),
		NewNullString(member.MembershipStatus),
		NewNullString(member.IPAddress),
		lastLoginToInsert,
		joinDateToInsert,
		NewNullString(member.Saldo),
		NewNullString(member.MembershipEmail),
		NewNullString(member.Turnover),
		NewNullString(member.WinLoss),
		NewNullString(member.Points),
		NewNullString(member.Referral),
		NewNullString(member.Uplink),
		NewNullStringFromPtr(member.CRMInfo),
		member.CRMUserID,
		now,
		now,
		now,
	).Scan(&memberID)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				log.Printf("Error CreateMember: Pelanggaran unik - Pesan: %s, Constraint: %s, Detail: %s", pqErr.Message, pqErr.Constraint, pqErr.Detail)
				if strings.Contains(pqErr.Constraint, "username_unique_idx") || strings.Contains(pqErr.Message, "username") { // Lebih generik jika nama constraint berubah
					return 0, errors.New("username sudah terdaftar")
				}
				if strings.Contains(pqErr.Constraint, "members_phone_number_brand_name_key") { // Sesuaikan dengan nama constraint di DB Anda
					return 0, errors.New("nomor telepon sudah terdaftar untuk brand ini")
				}
				return 0, fmt.Errorf("data member sudah ada: %s", pqErr.Detail)
			}
		}
		log.Printf("Error CreateMember: Gagal mengeksekusi query: %v", err)
		return 0, fmt.Errorf("gagal membuat member: %w", err)
	}

	log.Printf("Member baru berhasil dibuat dengan ID: %d", memberID)
	return memberID, nil
}

// BulkInsertMembers mengimplementasikan penyimpanan banyak data member ke database.
func (r *memberRepository) BulkInsertMembers(members []models.MemberData) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("gagal memulai transaksi: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO members (username, ip_address, last_login, email, membership_status, phone_number, membership_email, bank_name, account_name, account_no, saldo, turnover, win_loss, points, join_date, referral, uplink, status, brand_name, crm_info, crm_user_id, created_at, updated_at, uploaded_at) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, NOW(), NOW(), NOW())`)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("gagal menyiapkan statement: %w", err)
	}
	defer stmt.Close()

	importedCount := 0
	for _, member := range members {
		_, err := stmt.Exec(
			member.Username,
			NewNullString(member.IPAddress),
			NewNullStringFromPtr(member.LastLogin),
			NewNullString(member.Email),
			NewNullString(member.MembershipStatus),
			member.PhoneNumber,
			NewNullString(member.MembershipEmail),
			NewNullString(member.BankName),
			NewNullString(member.AccountName),
			NewNullString(member.AccountNo),
			NewNullString(member.Saldo),
			NewNullString(member.Turnover),
			NewNullString(member.WinLoss),
			NewNullString(member.Points),
			NewNullStringFromPtr(member.JoinDate),
			NewNullString(member.Referral),
			NewNullString(member.Uplink),
			NewNullString(member.Status),
			member.BrandName,
			NewNullStringFromPtr(member.CRMInfo),
			member.CRMUserID,
		)
		if err != nil {
			tx.Rollback()
			log.Printf("Gagal mengeksekusi statement untuk Username: %s, Phone: %s, Brand: %s. Error: %v", member.Username, member.PhoneNumber, member.BrandName, err)
			return importedCount, fmt.Errorf("gagal mengeksekusi statement untuk username %s: %w", member.Username, err)
		}
		importedCount++
	}

	if err := tx.Commit(); err != nil {
		return importedCount, fmt.Errorf("gagal melakukan commit transaksi: %w", err)
	}

	return importedCount, nil
}

// MemberExists memeriksa apakah member dengan nomor telepon dan brand_name tertentu sudah ada.
func (r *memberRepository) MemberExists(phoneNumber, brandName string) (bool, int, error) {
	if strings.TrimSpace(phoneNumber) == "" {
		return false, 0, nil
	}
	query := "SELECT id FROM members WHERE phone_number = $1 AND brand_name = $2"
	var memberID int
	err := r.db.QueryRow(query, phoneNumber, brandName).Scan(&memberID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, 0, nil
		}
		log.Printf("Error checking if member exists (phone: %s, brand: %s): %v", phoneNumber, brandName, err)
		return false, 0, fmt.Errorf("failed to check member existence: %w", err)
	}
	return true, memberID, nil
}

// GetAllMembers mengambil data member dengan paginasi, pencarian, filter brand, dan sorting.
func (r *memberRepository) GetAllMembers(page, limit int, searchTerm, filterBrand, filterStatus, sortBy, sortOrder string) ([]models.MemberData, int, error) {
	selectFields := `
		m.id, m.username, COALESCE(m.ip_address, '') AS ip_address, COALESCE(m.last_login, '') AS last_login, COALESCE(m.email, '') AS email, 
		COALESCE(m.membership_status, '') AS membership_status, m.phone_number, COALESCE(m.membership_email, '') AS membership_email, 
		COALESCE(m.bank_name, '') AS bank_name, COALESCE(m.account_name, '') AS account_name, COALESCE(m.account_no, '') AS account_no, 
		COALESCE(m.saldo, '0') AS saldo, COALESCE(m.turnover, '0') AS turnover, COALESCE(m.win_loss, '0') AS win_loss, 
		COALESCE(m.points, '0') AS points, COALESCE(m.join_date, '') AS join_date, COALESCE(m.referral, '') AS referral, 
		COALESCE(m.uplink, '') AS uplink, COALESCE(m.status, '') AS status, m.brand_name, 
		COALESCE(actual_crm.username, m.crm_info, '') AS crm_info, -- Mengambil username CRM, fallback ke crm_info jika tidak ada join
		m.crm_user_id, -- Ditambahkan
		m.created_at, m.updated_at, m.uploaded_at
	`
	countSelectFields := `COUNT(DISTINCT m.id)`
	baseQuery := `FROM members m LEFT JOIN users actual_crm ON m.crm_user_id = actual_crm.id` // Ditambahkan LEFT JOIN

	var whereClauses []string
	var args []interface{}
	argId := 1

	if searchTerm != "" {
		searchPattern := "%" + strings.ToLower(searchTerm) + "%"
		searchConditions := []string{
			fmt.Sprintf("LOWER(m.username) ILIKE $%d", argId),
			fmt.Sprintf("LOWER(COALESCE(m.email,'')) ILIKE $%d", argId),
			fmt.Sprintf("m.phone_number ILIKE $%d", argId),
			fmt.Sprintf("LOWER(COALESCE(m.account_name,'')) ILIKE $%d", argId),
			fmt.Sprintf("COALESCE(m.account_no,'') ILIKE $%d", argId),
			fmt.Sprintf("LOWER(COALESCE(m.ip_address,'')) ILIKE $%d", argId),
			fmt.Sprintf("LOWER(m.brand_name) ILIKE $%d", argId),
		}
		whereClauses = append(whereClauses, "("+strings.Join(searchConditions, " OR ")+")")
		args = append(args, searchPattern)
		argId++
	}

	if filterBrand != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("m.brand_name = $%d", argId))
		args = append(args, filterBrand)
		argId++
	}
	if filterStatus != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("COALESCE(m.status, '') = $%d", argId))
		args = append(args, filterStatus)
		argId++
	}

	whereQuery := ""
	if len(whereClauses) > 0 {
		whereQuery = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	totalQueryArgs := make([]interface{}, len(args))
	copy(totalQueryArgs, args)
	totalQuery := fmt.Sprintf("SELECT %s %s%s", countSelectFields, baseQuery, whereQuery)
	var totalRecords int
	err := r.db.QueryRow(totalQuery, totalQueryArgs...).Scan(&totalRecords)
	if err != nil {
		log.Printf("Error counting members: %v, Query: %s, Args: %v", err, totalQuery, totalQueryArgs)
		return nil, 0, fmt.Errorf("failed to count members: %w", err)
	}

	allowedSortBy := map[string]string{
		"username": "m.username", "email": "m.email", "brand_name": "m.brand_name",
		"status": "m.status", "join_date": "m.join_date", "id": "m.id", "created_at": "m.created_at", "saldo": "m.saldo",
	}
	dbSortBy, ok := allowedSortBy[strings.ToLower(sortBy)]
	if !ok {
		dbSortBy = "m.id"
	}
	dbSortOrder := "ASC"
	if strings.ToUpper(sortOrder) == "DESC" {
		dbSortOrder = "DESC"
	}
	orderByQuery := fmt.Sprintf(" ORDER BY %s %s, m.id %s", dbSortBy, dbSortOrder, dbSortOrder)

	limitOffsetQuery := fmt.Sprintf(" LIMIT $%d OFFSET $%d", argId, argId+1)
	finalArgs := append(args, limit, (page-1)*limit)

	dataQuery := fmt.Sprintf("SELECT %s %s%s%s%s", selectFields, baseQuery, whereQuery, orderByQuery, limitOffsetQuery)

	rows, err := r.db.Query(dataQuery, finalArgs...)
	if err != nil {
		log.Printf("Error querying members: %v, Query: %s, Args: %v", err, dataQuery, finalArgs)
		return nil, 0, fmt.Errorf("failed to query members: %w", err)
	}
	defer rows.Close()

	var members []models.MemberData
	for rows.Next() {
		var member models.MemberData
		var ipAddress, email, bankName, accountName, accountNo, status, membershipStatus, saldo, membershipEmail, turnover, winLoss, points, referral, uplink, crmInfoFromQuery sql.NullString
		var lastLogin, joinDate sql.NullString
		// crm_user_id akan discan ke member.CRMUserID (sql.NullInt64)

		err := rows.Scan(
			&member.ID, &member.Username, &ipAddress, &lastLogin, &email,
			&membershipStatus, &member.PhoneNumber, &membershipEmail, &bankName,
			&accountName, &accountNo, &saldo, &turnover, &winLoss,
			&points, &joinDate, &referral, &uplink, &status, &member.BrandName,
			&crmInfoFromQuery, // Untuk COALESCE(actual_crm.username, m.crm_info, '')
			&member.CRMUserID, // Langsung ke field model
			&member.CreatedAt, &member.UpdatedAt, &member.UploadedAt,
		)
		if err != nil {
			log.Printf("Error scanning member row: %v", err)
			return nil, 0, fmt.Errorf("failed to scan member data: %w", err)
		}

		member.IPAddress = ipAddress.String
		if lastLogin.Valid {
			member.LastLogin = &lastLogin.String
		} else {
			member.LastLogin = nil
		}
		member.Email = email.String
		member.MembershipStatus = membershipStatus.String
		member.MembershipEmail = membershipEmail.String
		member.BankName = bankName.String
		member.AccountName = accountName.String
		member.AccountNo = accountNo.String
		member.Saldo = saldo.String
		member.Turnover = turnover.String
		member.WinLoss = winLoss.String
		member.Points = points.String
		if joinDate.Valid {
			member.JoinDate = &joinDate.String
		} else {
			member.JoinDate = nil
		}
		member.Referral = referral.String
		member.Uplink = uplink.String
		member.Status = status.String
		if crmInfoFromQuery.Valid {
			member.CRMInfo = &crmInfoFromQuery.String
		} else {
			member.CRMInfo = nil
		}

		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating member rows: %v", err)
		return nil, 0, fmt.Errorf("error iterating member rows: %w", err)
	}

	return members, totalRecords, nil
}

// UpdateMemberPhoneNumber memperbarui nomor telepon member berdasarkan ID.
func (r *memberRepository) UpdateMemberPhoneNumber(memberID int, newPhoneNumber string) error {
	query := `UPDATE members SET phone_number = $1, updated_at = $2 WHERE id = $3`
	now := time.Now()

	result, err := r.db.Exec(query, newPhoneNumber, now, memberID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				log.Printf("Error UpdateMemberPhoneNumber: Pelanggaran unik - Pesan: %s, Constraint: %s", pqErr.Message, pqErr.Constraint)
				// Anda mungkin perlu memeriksa nama constraint di sini untuk pesan error yang lebih spesifik
				// Misalnya, jika ada constraint unik pada (phone_number, brand_name)
				return errors.New("nomor telepon sudah terdaftar")
			}
		}
		log.Printf("Error UpdateMemberPhoneNumber: Gagal mengeksekusi query untuk ID %d: %v", memberID, err)
		return fmt.Errorf("gagal memperbarui nomor telepon member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error UpdateMemberPhoneNumber: Gagal mendapatkan baris yang terpengaruh untuk ID %d: %v", memberID, err)
		return fmt.Errorf("gagal mendapatkan info pembaruan nomor telepon: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("Error UpdateMemberPhoneNumber: Tidak ada member ditemukan dengan ID %d untuk diperbarui", memberID)
		return errors.New("member tidak ditemukan")
	}

	log.Printf("Nomor telepon untuk member ID %d berhasil diperbarui.", memberID)
	return nil
}

// GetMemberByID mengambil satu member berdasarkan ID.
func (r *memberRepository) GetMemberByID(memberID int) (*models.MemberData, error) {
	query := `
		SELECT 
			m.id, m.username, COALESCE(m.ip_address, '') AS ip_address, COALESCE(m.last_login, '') AS last_login, COALESCE(m.email, '') AS email, 
			COALESCE(m.membership_status, '') AS membership_status, m.phone_number, COALESCE(m.membership_email, '') AS membership_email, 
			COALESCE(m.bank_name, '') AS bank_name, COALESCE(m.account_name, '') AS account_name, COALESCE(m.account_no, '') AS account_no, 
			COALESCE(m.saldo, '0') AS saldo, COALESCE(m.turnover, '0') AS turnover, COALESCE(m.win_loss, '0') AS win_loss, 
			COALESCE(m.points, '0') AS points, COALESCE(m.join_date, '') AS join_date, COALESCE(m.referral, '') AS referral, 
			COALESCE(m.uplink, '') AS uplink, COALESCE(m.status, '') AS status, m.brand_name, 
			COALESCE(actual_crm.username, m.crm_info, '') AS crm_info, -- Mengambil username CRM, fallback ke crm_info
			m.crm_user_id, -- Ditambahkan
			m.created_at, m.updated_at, m.uploaded_at
		FROM members m LEFT JOIN users actual_crm ON m.crm_user_id = actual_crm.id -- Ditambahkan LEFT JOIN
		WHERE m.id = $1
	`
	var member models.MemberData
	var ipAddress, email, bankName, accountName, accountNo, status, membershipStatus, saldo, membershipEmail, turnover, winLoss, points, referral, uplink, crmInfoFromQuery sql.NullString
	var lastLogin, joinDate sql.NullString

	err := r.db.QueryRow(query, memberID).Scan(
		&member.ID, &member.Username, &ipAddress, &lastLogin, &email,
		&membershipStatus, &member.PhoneNumber, &membershipEmail, &bankName,
		&accountName, &accountNo, &saldo, &turnover, &winLoss,
		&points, &joinDate, &referral, &uplink, &status, &member.BrandName,
		&crmInfoFromQuery, // Untuk COALESCE(actual_crm.username, m.crm_info, '')
		&member.CRMUserID, // Langsung ke field model
		&member.CreatedAt, &member.UpdatedAt, &member.UploadedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("member tidak ditemukan")
		}
		log.Printf("Error GetMemberByID (ID: %d): %v", memberID, err)
		return nil, fmt.Errorf("gagal mengambil data member: %w", err)
	}

	member.IPAddress = ipAddress.String
	if lastLogin.Valid {
		member.LastLogin = &lastLogin.String
	} else {
		member.LastLogin = nil
	}
	member.Email = email.String
	member.MembershipStatus = membershipStatus.String
	member.MembershipEmail = membershipEmail.String
	member.BankName = bankName.String
	member.AccountName = accountName.String
	member.AccountNo = accountNo.String
	member.Saldo = saldo.String
	member.Turnover = turnover.String
	member.WinLoss = winLoss.String
	member.Points = points.String
	if joinDate.Valid {
		member.JoinDate = &joinDate.String
	} else {
		member.JoinDate = nil
	}
	member.Referral = referral.String
	member.Uplink = uplink.String
	member.Status = status.String
	if crmInfoFromQuery.Valid {
		member.CRMInfo = &crmInfoFromQuery.String
	} else {
		member.CRMInfo = nil
	}

	return &member, nil
}

// UpdateMemberCRM memperbarui informasi CRM untuk seorang member.
// Sekarang menerima crmUserID dan crmInfo secara terpisah.
func (r *memberRepository) UpdateMemberCRM(memberID int, crmInfo sql.NullString, crmUserID sql.NullInt64) error {
	query := `UPDATE members SET crm_info = $1, crm_user_id = $2, updated_at = $3 WHERE id = $4`
	now := time.Now()

	result, err := r.db.Exec(query, crmInfo, crmUserID, now, memberID)
	if err != nil {
		log.Printf("Error UpdateMemberCRM: Gagal mengeksekusi query untuk ID %d: %v", memberID, err)
		return fmt.Errorf("gagal memperbarui CRM member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error UpdateMemberCRM: Gagal mendapatkan baris yang terpengaruh untuk ID %d: %v", memberID, err)
		return fmt.Errorf("gagal mendapatkan info pembaruan CRM: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("Error UpdateMemberCRM: Tidak ada member ditemukan dengan ID %d untuk diperbarui CRM-nya", memberID)
		return errors.New("member tidak ditemukan untuk pembaruan CRM")
	}

	log.Printf("CRM untuk member ID %d berhasil diperbarui. crm_info: %v, crm_user_id: %v", memberID, crmInfo, crmUserID)
	return nil
}

// Helper untuk konversi string ke sql.NullString
func NewNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

// Helper untuk konversi *string (pointer) ke interface{} yang bisa sql.NullString atau nil
func NewNullStringFromPtr(s *string) interface{} {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}
