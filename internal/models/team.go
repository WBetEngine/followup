package models

import (
	"database/sql"
	"errors"
	"time"
)

// Team represents a team in the system.
type Team struct {
	ID          int64          `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Description sql.NullString `json:"description,omitempty" db:"description"` // Bisa NULL
	AdminUserID int64          `json:"admin_user_id" db:"admin_user_id"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`

	// Fields for display, populated by specific queries (e.g., join)
	AdminUsername string `json:"admin_username,omitempty" db:"admin_username"` // Username admin untuk tampilan
	MemberCount   int    `json:"member_count,omitempty" db:"member_count"`     // Jumlah anggota tim
}

// TeamMember represents a user's membership in a team.
// Ini adalah representasi dari baris di tabel team_members.
type TeamMember struct {
	ID        int64     `json:"id" db:"id"` // ID dari record di tabel team_members
	TeamID    int64     `json:"team_id" db:"team_id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TeamMemberDetail is a richer representation of a team member for display purposes.
type TeamMemberDetail struct {
	UserID       int64     `json:"user_id" db:"user_id"`
	Username     string    `json:"username" db:"username"`
	UserFullName string    `json:"user_full_name" db:"user_full_name"`
	UserRole     UserRole  `json:"user_role" db:"user_role"`           // Peran user dari tabel 'users'
	TeamID       int64     `json:"team_id" db:"team_id"`               // TeamID tempat dia menjadi anggota
	TeamName     string    `json:"team_name,omitempty" db:"team_name"` // Nama tim (opsional, jika perlu)
	JoinedAt     time.Time `json:"joined_at" db:"joined_at"`           // Kapan user join tim ini (CreatedAt dari team_members)
}

// TeamWithDetails is a struct to hold a team and its members' details for API responses or page data.
type TeamWithDetails struct {
	Team
	Admin   *UserBasicInfo     `json:"admin,omitempty"` // Info dasar admin
	Members []TeamMemberDetail `json:"members,omitempty"`
}

// --- Request/Response Payloads (Contoh untuk API) ---

// CreateTeamRequest defines the payload for creating a new team.
type CreateTeamRequest struct {
	Name        string  `json:"name" validate:"required,min=3,max=100"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"` // Pointer agar bisa kirim null atau string kosong
	AdminUserID int64   `json:"admin_user_id" validate:"required,gt=0"`
}

// UpdateTeamRequest defines the payload for updating an existing team.
type UpdateTeamRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=3,max=100"`  // Pointer agar field opsional
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"` // Pointer agar field opsional
	AdminUserID *int64  `json:"admin_user_id,omitempty" validate:"omitempty,gt=0"`  // Pointer agar field opsional
}

// AddMemberRequest defines the payload for adding a member to a team.
type AddMemberRequest struct {
	UserID int64 `json:"user_id" validate:"required,gt=0"`
}

// Error constants related to Team operations
var (
	ErrTeamNameTaken           = errors.New("nama tim sudah digunakan")
	ErrTeamNotFound            = errors.New("tim tidak ditemukan")
	ErrUserAlreadyInTeam       = errors.New("pengguna sudah menjadi anggota tim") // Lebih generik daripada 'tim lain'
	ErrCannotRemoveAdmin       = errors.New("admin utama tim tidak bisa dihapus sebagai anggota biasa, ganti admin terlebih dahulu")
	ErrMemberNotFoundInTeam    = errors.New("anggota tidak ditemukan di tim ini")
	ErrAdminStillAssigned      = errors.New("pengguna masih menjadi admin di sebuah tim, tidak bisa dihapus atau diubah perannya")
	ErrUserNotAdmin            = errors.New("pengguna bukan admin tim")
	ErrTeamHasMembers          = errors.New("tim masih memiliki anggota, tidak dapat dihapus") // Untuk operasi delete team
	ErrCannotDemoteSelfAsAdmin = errors.New("admin tidak dapat mengubah perannya sendiri atau menghapus dirinya dari tim melalui operasi ini")
	ErrInvalidTeamData         = errors.New("data tim tidak valid")
	ErrInvalidMemberData       = errors.New("data anggota tim tidak valid")
)
