package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"followup/internal/models"
	"log"
	"strings"
	"time"
)

// TeamRepositoryInterface defines the interface for database operations related to Teams and TeamMembers.
type TeamRepositoryInterface interface {
	CreateTeam(team *models.Team) (int64, error)
	GetTeamByID(id int64) (*models.TeamWithDetails, error) // Akan mengambil juga basic info admin & member count
	GetTeamByName(name string) (*models.Team, error)       // Untuk validasi nama unik
	GetTeamsPage(searchTerm string, page, limit int) ([]models.Team, int, error)
	UpdateTeam(team *models.Team) error
	DeleteTeam(id int64) error
	CheckUserIsAdmin(userID int64) (bool, int64, error) // Cek apakah user adalah admin di suatu tim, return teamID

	AddMember(teamID, userID int64) (int64, error)
	GetMembersByTeamID(teamID int64) ([]models.TeamMemberDetail, error)
	GetTeamMembershipByUserID(userID int64) (*models.TeamMember, error)
	RemoveMember(teamID, userID int64) error
	// RemoveAllMembersFromTeam(teamID int64) error // Mungkin berguna jika tidak menggunakan ON DELETE CASCADE
	GetTeamIDForUser(userID int64) (int64, bool, error) // Mengembalikan teamID dan boolean (true jika ditemukan)

	GetUsersNotYetInTeam(excludeUserIDs []int64, rolesToInclude []models.UserRole, searchTerm string, limit int) ([]models.UserBasicInfo, error)
	GetPotentialAdmins(excludeUserIDs []int64, searchTerm string, limit int) ([]models.UserBasicInfo, error)
}

type teamRepository struct {
	db *sql.DB
}

// NewTeamRepository creates a new instance of TeamRepository.
func NewTeamRepository(db *sql.DB) TeamRepositoryInterface {
	return &teamRepository{db: db}
}

func (r *teamRepository) CreateTeam(team *models.Team) (int64, error) {
	query := `INSERT INTO teams (name, description, admin_user_id, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5) RETURNING id`
	team.CreatedAt = time.Now()
	team.UpdatedAt = time.Now()

	err := r.db.QueryRow(query, team.Name, team.Description, team.AdminUserID, team.CreatedAt, team.UpdatedAt).Scan(&team.ID)
	if err != nil {
		if strings.Contains(err.Error(), "uq_teams_name") {
			return 0, fmt.Errorf("nama tim '%s' sudah digunakan: %w", team.Name, models.ErrTeamNameTaken)
		}
		if strings.Contains(err.Error(), "fk_teams_admin_user") {
			return 0, fmt.Errorf("admin user ID %d tidak valid: %w", team.AdminUserID, models.ErrUserNotFound)
		}
		log.Printf("Error creating team '%s': %v", team.Name, err)
		return 0, fmt.Errorf("gagal membuat tim: %w", err)
	}
	return team.ID, nil
}

func (r *teamRepository) GetTeamByID(id int64) (*models.TeamWithDetails, error) {
	teamQuery := `
		SELECT t.id, t.name, t.description, t.admin_user_id, u_admin.username AS admin_username, 
		       t.created_at, t.updated_at, COALESCE(COUNT(DISTINCT tm.user_id), 0) AS member_count
		FROM teams t
		JOIN users u_admin ON t.admin_user_id = u_admin.id
		LEFT JOIN team_members tm ON t.id = tm.team_id
		WHERE t.id = $1
		GROUP BY t.id, u_admin.username
	`
	twd := &models.TeamWithDetails{}
	twd.Admin = &models.UserBasicInfo{}

	err := r.db.QueryRow(teamQuery, id).Scan(
		&twd.Team.ID, &twd.Team.Name, &twd.Team.Description, &twd.Team.AdminUserID, &twd.Admin.Username,
		&twd.Team.CreatedAt, &twd.Team.UpdatedAt, &twd.Team.MemberCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrTeamNotFound
		}
		log.Printf("Error querying team by ID %d: %v", id, err)
		return nil, fmt.Errorf("gagal mengambil tim berdasarkan ID: %w", err)
	}
	twd.Admin.ID = twd.Team.AdminUserID // Set Admin ID from team data

	// Ambil anggota tim secara terpisah untuk menghindari kompleksitas query tunggal
	members, err := r.GetMembersByTeamID(id)
	if err != nil {
		// Jika error bukan karena tidak ada anggota (misalnya, error DB), kembalikan error
		// Jika tidak ada anggota itu normal, members akan jadi slice kosong
		log.Printf("Warning/Error fetching members for team ID %d during GetTeamByID: %v", id, err)
		// Kita tidak mengembalikan error di sini jika hanya member yang tidak ada, twd.Members akan kosong
	}
	twd.Members = members

	return twd, nil
}

func (r *teamRepository) GetTeamByName(name string) (*models.Team, error) {
	query := `SELECT id, name, description, admin_user_id, created_at, updated_at FROM teams WHERE name = $1`
	team := &models.Team{}
	err := r.db.QueryRow(query, name).Scan(
		&team.ID, &team.Name, &team.Description, &team.AdminUserID, &team.CreatedAt, &team.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrTeamNotFound // Atau bisa return nil, nil jika nama tidak ditemukan adalah kondisi normal
		}
		log.Printf("Error querying team by name '%s': %v", name, err)
		return nil, fmt.Errorf("gagal mengambil tim berdasarkan nama: %w", err)
	}
	return team, nil
}

func (r *teamRepository) GetTeamsPage(searchTerm string, page, limit int) ([]models.Team, int, error) {
	var teams []models.Team
	var args []interface{}
	var countArgs []interface{}

	baseQuery := `
		SELECT t.id, t.name, t.description, t.admin_user_id, u.username as admin_username, 
		       t.created_at, t.updated_at, COALESCE(mc.member_count, 0) as member_count
		FROM teams t
		JOIN users u ON t.admin_user_id = u.id
		LEFT JOIN (
		    SELECT team_id, COUNT(user_id) as member_count 
		    FROM team_members 
		    GROUP BY team_id
		) mc ON t.id = mc.team_id
	`
	countQuery := `SELECT COUNT(DISTINCT t.id) FROM teams t`

	conditions := ""
	paramIdx := 1

	if searchTerm != "" {
		conditions += fmt.Sprintf(" WHERE LOWER(t.name) ILIKE $%d OR LOWER(t.description) ILIKE $%d OR LOWER(u.username) ILIKE $%d", paramIdx, paramIdx, paramIdx)
		searchPattern := "%" + strings.ToLower(searchTerm) + "%"
		args = append(args, searchPattern)
		countArgs = append(countArgs, searchPattern)
		paramIdx++
	}

	finalCountQuery := countQuery + conditions
	var totalRecords int
	err := r.db.QueryRow(finalCountQuery, countArgs...).Scan(&totalRecords)
	if err != nil {
		log.Printf("Error counting teams: %v. Query: [%s], Args: %v", err, finalCountQuery, countArgs)
		return nil, 0, fmt.Errorf("gagal menghitung total tim: %w", err)
	}

	if totalRecords == 0 {
		return teams, 0, nil
	}

	finalQuery := baseQuery + conditions + " ORDER BY t.name ASC"
	if limit > 0 {
		finalQuery += fmt.Sprintf(" LIMIT $%d", paramIdx)
		args = append(args, limit)
		paramIdx++
		if page > 0 {
			offset := (page - 1) * limit
			finalQuery += fmt.Sprintf(" OFFSET $%d", paramIdx)
			args = append(args, offset)
		}
	}

	rows, err := r.db.Query(finalQuery, args...)
	if err != nil {
		log.Printf("Error querying teams page: %v. Query: [%s], Args: %v", err, finalQuery, args)
		return nil, 0, fmt.Errorf("gagal mengambil daftar tim: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var team models.Team
		if err := rows.Scan(
			&team.ID, &team.Name, &team.Description, &team.AdminUserID, &team.AdminUsername,
			&team.CreatedAt, &team.UpdatedAt, &team.MemberCount,
		); err != nil {
			log.Printf("Error scanning team row: %v", err)
			return nil, 0, fmt.Errorf("gagal memindai data tim: %w", err)
		}
		teams = append(teams, team)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating team rows: %v", err)
		return nil, 0, fmt.Errorf("error iterasi baris tim: %w", err)
	}

	return teams, totalRecords, nil
}

func (r *teamRepository) UpdateTeam(team *models.Team) error {
	query := `UPDATE teams SET name = $1, description = $2, admin_user_id = $3, updated_at = $4 WHERE id = $5`
	team.UpdatedAt = time.Now()

	result, err := r.db.Exec(query, team.Name, team.Description, team.AdminUserID, team.UpdatedAt, team.ID)
	if err != nil {
		if strings.Contains(err.Error(), "uq_teams_name") {
			return fmt.Errorf("nama tim '%s' sudah digunakan: %w", team.Name, models.ErrTeamNameTaken)
		}
		if strings.Contains(err.Error(), "fk_teams_admin_user") {
			return fmt.Errorf("admin user ID %d tidak valid: %w", team.AdminUserID, models.ErrUserNotFound)
		}
		log.Printf("Error updating team ID %d: %v", team.ID, err)
		return fmt.Errorf("gagal memperbarui tim: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected after updating team ID %d: %v", team.ID, err)
		return fmt.Errorf("gagal memeriksa hasil pembaruan tim: %w", err)
	}
	if rowsAffected == 0 {
		return models.ErrTeamNotFound
	}
	return nil
}

func (r *teamRepository) DeleteTeam(id int64) error {
	query := `DELETE FROM teams WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Printf("Error deleting team ID %d: %v", id, err)
		return fmt.Errorf("gagal menghapus tim: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected after deleting team ID %d: %v", id, err)
		return fmt.Errorf("gagal memeriksa hasil penghapusan tim: %w", err)
	}
	if rowsAffected == 0 {
		return models.ErrTeamNotFound
	}
	return nil
}

func (r *teamRepository) CheckUserIsAdmin(userID int64) (bool, int64, error) {
	var teamID int64
	query := `SELECT id FROM teams WHERE admin_user_id = $1`
	err := r.db.QueryRow(query, userID).Scan(&teamID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, 0, nil // Bukan admin dari tim manapun
		}
		log.Printf("Error checking if user %d is admin: %v", userID, err)
		return false, 0, fmt.Errorf("gagal memeriksa status admin pengguna: %w", err)
	}
	return true, teamID, nil
}

func (r *teamRepository) AddMember(teamID, userID int64) (int64, error) {
	query := `INSERT INTO team_members (team_id, user_id, created_at) VALUES ($1, $2, $3) RETURNING id`
	var memberID int64
	createdAt := time.Now()
	err := r.db.QueryRow(query, teamID, userID, createdAt).Scan(&memberID)
	if err != nil {
		if strings.Contains(err.Error(), "uq_team_members_user_id") {
			return 0, fmt.Errorf("pengguna ID %d sudah menjadi anggota tim lain: %w", userID, models.ErrUserAlreadyInTeam)
		}
		if strings.Contains(err.Error(), "fk_team_members_team") {
			return 0, fmt.Errorf("tim ID %d tidak ditemukan: %w", teamID, models.ErrTeamNotFound)
		}
		if strings.Contains(err.Error(), "fk_team_members_user") {
			return 0, fmt.Errorf("pengguna ID %d tidak ditemukan: %w", userID, models.ErrUserNotFound)
		}
		log.Printf("Error adding member UID %d to team TID %d: %v", userID, teamID, err)
		return 0, fmt.Errorf("gagal menambahkan anggota ke tim: %w", err)
	}
	return memberID, nil
}

func (r *teamRepository) GetMembersByTeamID(teamID int64) ([]models.TeamMemberDetail, error) {
	query := `
		SELECT tm.user_id, u.username, u.name AS user_full_name, u.role AS user_role, tm.team_id, tm.created_at AS joined_at
		FROM team_members tm
		JOIN users u ON tm.user_id = u.id
		WHERE tm.team_id = $1
		ORDER BY u.name ASC
	`
	rows, err := r.db.Query(query, teamID)
	if err != nil {
		log.Printf("Error querying members for team ID %d: %v", teamID, err)
		return nil, fmt.Errorf("gagal mengambil anggota tim: %w", err)
	}
	defer rows.Close()

	var members []models.TeamMemberDetail
	for rows.Next() {
		var member models.TeamMemberDetail
		if err := rows.Scan(&member.UserID, &member.Username, &member.UserFullName, &member.UserRole, &member.TeamID, &member.JoinedAt); err != nil {
			log.Printf("Error scanning team member row for team ID %d: %v", teamID, err)
			return nil, fmt.Errorf("gagal memindai data anggota tim: %w", err)
		}
		members = append(members, member)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating team member rows for team ID %d: %v", teamID, err)
		return nil, fmt.Errorf("error iterasi baris anggota tim: %w", err)
	}
	return members, nil
}

func (r *teamRepository) GetTeamMembershipByUserID(userID int64) (*models.TeamMember, error) {
	query := `SELECT id, team_id, user_id, created_at FROM team_members WHERE user_id = $1`
	member := &models.TeamMember{}
	err := r.db.QueryRow(query, userID).Scan(&member.ID, &member.TeamID, &member.UserID, &member.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User bukan anggota tim manapun, bukan error
		}
		log.Printf("Error querying team membership for user ID %d: %v", userID, err)
		return nil, fmt.Errorf("gagal mengambil keanggotaan tim pengguna: %w", err)
	}
	return member, nil
}

func (r *teamRepository) GetTeamIDForUser(userID int64) (int64, bool, error) {
	var teamID int64
	query := `SELECT team_id FROM team_members WHERE user_id = $1`
	err := r.db.QueryRow(query, userID).Scan(&teamID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil // User tidak ditemukan di tim mana pun
		}
		log.Printf("Error getting team_id for user_id %d: %v", userID, err)
		return 0, false, fmt.Errorf("gagal mendapatkan team_id untuk user: %w", err)
	}
	return teamID, true, nil
}

func (r *teamRepository) RemoveMember(teamID, userID int64) error {
	// Periksa apakah user yang akan dihapus adalah admin dari tim ini
	var adminUserIDOfTeam int64
	checkAdminQuery := `SELECT admin_user_id FROM teams WHERE id = $1`
	err := r.db.QueryRow(checkAdminQuery, teamID).Scan(&adminUserIDOfTeam)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ErrTeamNotFound
		}
		log.Printf("Error checking admin for team ID %d before removing member UID %d: %v", teamID, userID, err)
		return fmt.Errorf("gagal memeriksa admin tim: %w", err)
	}

	if adminUserIDOfTeam == userID {
		return models.ErrCannotRemoveAdmin // Admin utama tidak boleh dihapus dari anggota biasa
	}

	query := `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`
	result, err := r.db.Exec(query, teamID, userID)
	if err != nil {
		log.Printf("Error removing member UID %d from team TID %d: %v", userID, teamID, err)
		return fmt.Errorf("gagal menghapus anggota dari tim: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected after removing member UID %d from team TID %d: %v", userID, teamID, err)
		return fmt.Errorf("gagal memeriksa hasil penghapusan anggota: %w", err)
	}
	if rowsAffected == 0 {
		return models.ErrMemberNotFoundInTeam // User bukan anggota tim ini
	}
	return nil
}

// GetUsersNotYetInTeam mengambil daftar user (ID, Username, Name) yang belum ada di tim manapun
// dan memiliki peran tertentu (misal CRM, Telemarketing).
func (r *teamRepository) GetUsersNotYetInTeam(excludeUserIDs []int64, rolesToInclude []models.UserRole, searchTerm string, limit int) ([]models.UserBasicInfo, error) {
	var users []models.UserBasicInfo
	var args []interface{}
	paramIdx := 1

	queryStr := `
        SELECT u.id, u.username, u.name 
        FROM users u
        LEFT JOIN team_members tm ON u.id = tm.user_id
        LEFT JOIN teams t_admin ON u.id = t_admin.admin_user_id
        WHERE tm.id IS NULL AND t_admin.id IS NULL` // Hanya user yang tidak ada di team_members DAN bukan admin di tabel teams

	if len(rolesToInclude) > 0 {
		rolePlaceholders := make([]string, len(rolesToInclude))
		for i, role := range rolesToInclude {
			rolePlaceholders[i] = fmt.Sprintf("$%d", paramIdx)
			args = append(args, role.String())
			paramIdx++
		}
		queryStr += fmt.Sprintf(" AND u.role IN (%s)", strings.Join(rolePlaceholders, ", "))
	}

	if len(excludeUserIDs) > 0 {
		excludePlaceholders := make([]string, len(excludeUserIDs))
		for i, userID := range excludeUserIDs {
			excludePlaceholders[i] = fmt.Sprintf("$%d", paramIdx)
			args = append(args, userID)
			paramIdx++
		}
		queryStr += fmt.Sprintf(" AND u.id NOT IN (%s)", strings.Join(excludePlaceholders, ", "))
	}

	if searchTerm != "" {
		queryStr += fmt.Sprintf(" AND (LOWER(u.username) ILIKE $%d OR LOWER(u.name) ILIKE $%d)", paramIdx, paramIdx)
		args = append(args, "%"+strings.ToLower(searchTerm)+"%")
		paramIdx++
	}

	queryStr += " ORDER BY u.name ASC"
	if limit > 0 {
		queryStr += fmt.Sprintf(" LIMIT $%d", paramIdx)
		args = append(args, limit)
	}

	rows, err := r.db.Query(queryStr, args...)
	if err != nil {
		log.Printf("Error querying users not in team: %v. Query: %s, Args: %v", err, queryStr, args)
		return nil, fmt.Errorf("gagal mengambil daftar pengguna yang belum masuk tim: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user models.UserBasicInfo
		if err := rows.Scan(&user.ID, &user.Username, &user.Name); err != nil {
			log.Printf("Error scanning user_basic_info row: %v", err)
			return nil, fmt.Errorf("gagal memindai data pengguna: %w", err)
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating user_basic_info rows: %v", err)
		return nil, fmt.Errorf("error iterasi baris pengguna: %w", err)
	}
	return users, nil
}

// GetPotentialAdmins mengambil daftar user yang bisa menjadi admin tim (misalnya, peran 'admin' atau 'superadmin')
// dan belum menjadi admin tim lain.
func (r *teamRepository) GetPotentialAdmins(excludeUserIDs []int64, searchTerm string, limit int) ([]models.UserBasicInfo, error) {
	var users []models.UserBasicInfo
	var args []interface{}
	paramIdx := 1

	// User dengan peran 'admin' atau 'superadmin' yang tidak tercatat sebagai admin_user_id di tabel teams
	queryStr := `
        SELECT u.id, u.username, u.name
        FROM users u
        LEFT JOIN teams t ON u.id = t.admin_user_id
        WHERE t.id IS NULL AND (u.role = $1 OR u.role = $2)`
	args = append(args, models.AdminRole.String(), models.SuperadminRole.String())
	paramIdx = 3

	if len(excludeUserIDs) > 0 {
		excludePlaceholders := make([]string, len(excludeUserIDs))
		for i, userID := range excludeUserIDs {
			excludePlaceholders[i] = fmt.Sprintf("$%d", paramIdx)
			args = append(args, userID)
			paramIdx++
		}
		queryStr += fmt.Sprintf(" AND u.id NOT IN (%s)", strings.Join(excludePlaceholders, ", "))
	}

	if searchTerm != "" {
		queryStr += fmt.Sprintf(" AND (LOWER(u.username) ILIKE $%d OR LOWER(u.name) ILIKE $%d)", paramIdx, paramIdx)
		args = append(args, "%"+strings.ToLower(searchTerm)+"%")
		paramIdx++
	}

	queryStr += " ORDER BY u.name ASC"
	if limit > 0 {
		queryStr += fmt.Sprintf(" LIMIT $%d", paramIdx)
		args = append(args, limit)
	}

	rows, err := r.db.Query(queryStr, args...)
	if err != nil {
		log.Printf("Error querying potential admins: %v. Query: %s, Args: %v", err, queryStr, args)
		return nil, fmt.Errorf("gagal mengambil daftar calon admin: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user models.UserBasicInfo
		if err := rows.Scan(&user.ID, &user.Username, &user.Name); err != nil {
			log.Printf("Error scanning potential admin row: %v", err)
			return nil, fmt.Errorf("gagal memindai data calon admin: %w", err)
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating potential admin rows: %v", err)
		return nil, fmt.Errorf("error iterasi baris calon admin: %w", err)
	}
	return users, nil
}
