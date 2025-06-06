package models

import (
	"database/sql"
	"time"
)

// MemberData adalah struct tunggal untuk data member, baik dari Excel maupun database.
// Tag `json` untuk serialisasi API/JavaScript.
// Tag `db` untuk referensi kolom database (berguna jika menggunakan sqlx atau untuk kejelasan).
type MemberData struct {
	ID               int64         `json:"id" db:"id"`
	Username         string        `json:"username" db:"username"`
	Email            string        `json:"email,omitempty" db:"email"`
	PhoneNumber      string        `json:"phone_number" db:"phone_number"`
	BankName         string        `json:"bank_name,omitempty" db:"bank_name"` // Konsisten dengan DB
	AccountName      string        `json:"account_name,omitempty" db:"account_name"`
	AccountNo        string        `json:"account_no,omitempty" db:"account_no"`
	BrandName        string        `json:"brand_name" db:"brand_name"`
	Status           string        `json:"status,omitempty" db:"status"` // e.g., "New Deposit", "Redeposit"
	MembershipStatus string        `json:"membership_status,omitempty" db:"membership_status"`
	IPAddress        string        `json:"ip_address,omitempty" db:"ip_address"`
	LastLogin        *string       `json:"last_login,omitempty" db:"last_login"`             // Pointer untuk nullable string
	JoinDate         *string       `json:"join_date,omitempty" db:"join_date"`               // Pointer untuk nullable string, atau *time.Time
	UploadedAt       time.Time     `json:"uploaded_at,omitempty" db:"uploaded_at"`           // Dibuat non-pointer jika selalu ada saat ambil dari DB
	Saldo            string        `json:"saldo,omitempty" db:"saldo"`                       // Tetap string untuk fleksibilitas input
	MembershipEmail  string        `json:"membership_email,omitempty" db:"membership_email"` // Email dari kolom membership Excel
	Turnover         string        `json:"turnover,omitempty" db:"turnover"`
	WinLoss          string        `json:"win_loss,omitempty" db:"win_loss"`
	Points           string        `json:"points,omitempty" db:"points"`
	Referral         string        `json:"referral,omitempty" db:"referral"`
	Uplink           string        `json:"uplink,omitempty" db:"uplink"`
	CRMInfo          *string       `json:"crm_info,omitempty" db:"crm_info"` // Kolom CRM baru
	CRMUserID        sql.NullInt64 `json:"crm_user_id,omitempty" db:"crm_user_id"`
	CreatedAt        time.Time     `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at,omitempty" db:"updated_at"`

	// Field spesifik parsing Excel (jika masih dibutuhkan terpisah dan belum termap ke atas)
	// No    string `json:"no,omitempty"` // Nomor urut dari Excel, mungkin tidak disimpan di DB
	// IP    string `json:"ip,omitempty"` // Sudah ada IPAddress

	// Field berikut dari definisi lama yang mungkin sudah tercakup atau perlu diklarifikasi:
	// Bank             string          `json:"bank"` // Sudah diganti BankName
	// Membership       string          `json:"membership_db,omitempty"` // Sudah ada MembershipStatus
	// RegistrationDate sql.NullTime    `json:"registration_date_db,omitempty"` // Sudah ada JoinDate (sebagai *string)
	// AccountNumber    string          `json:"account_number_db,omitempty"` // Sudah ada AccountNo
	// Amount           sql.NullFloat64 `json:"amount_db,omitempty"` // Saldo (string) bisa digunakan, konversi jika perlu
}
