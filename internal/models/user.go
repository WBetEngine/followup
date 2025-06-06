package models

import (
	"database/sql"
	"errors"
	"time"
)

// User represents a user in the system.
type User struct {
	ID        int64          `json:"id" db:"id"`
	Username  string         `json:"username" db:"username"`
	Password  string         `json:"-" db:"password"` // Password should not be exposed in JSON
	Name      string         `json:"name" db:"name"`
	Email     string         `json:"email" db:"email"`
	Role      UserRole       `json:"role" db:"role"` // Menggunakan tipe UserRole yang baru
	TeamName  sql.NullString `json:"team_name,omitempty" db:"team_name"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// UserBasicInfo holds essential information for assigning users as CRM.
// This struct is used to populate dropdowns for CRM assignment.
type UserBasicInfo struct {
	ID       int64  `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
	Name     string `json:"name" db:"name"`
}

// Definisikan error-error spesifik untuk operasi user
var (
	ErrUserNotFound            = errors.New("pengguna tidak ditemukan")
	ErrInvalidCredentials      = errors.New("kredensial tidak valid")
	ErrUsernameTaken           = errors.New("username sudah digunakan")
	ErrEmailTaken              = errors.New("email sudah digunakan")
	ErrDuplicateUsername       = errors.New("username sudah terdaftar")
	ErrDuplicateEmail          = errors.New("email sudah terdaftar")
	ErrPasswordTooShort        = errors.New("password terlalu pendek")
	ErrPasswordMismatch        = errors.New("konfirmasi password tidak cocok")
	ErrInvalidRole             = errors.New("peran tidak valid")
	ErrSuperadminAlreadyExists = errors.New("superadmin sudah ada, tidak dapat membuat lebih dari satu")
)

// UserRole merepresentasikan peran pengguna dalam sistem.
