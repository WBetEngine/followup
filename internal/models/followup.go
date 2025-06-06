package models

import (
	"database/sql"
	"time"
)

// FollowupListItem merepresentasikan satu item data dalam daftar followup.
// Field disesuaikan dengan informasi yang dibutuhkan di tabel halaman followup.
type FollowupListItem struct {
	ID                int64          `json:"id" db:"id"`                                   // ID unik member
	Username          string         `json:"username" db:"username"`                       // Username member
	Email             sql.NullString `json:"email" db:"membership_email"`                  // Email member (bisa null)
	PhoneNumber       sql.NullString `json:"phone_number" db:"phone_number"`               // Nomor telepon member (bisa null)
	BankName          sql.NullString `json:"bank_name" db:"bank_name"`                     // Nama bank member (bisa null)
	AccountNo         sql.NullString `json:"account_no" db:"account_no"`                   // Nomor rekening member (bisa null)
	BrandName         sql.NullString `json:"brand_name" db:"brand_name"`                   // Nama brand
	Status            string         `json:"status" db:"status"`                           // Status deposit (misalnya, "New Deposit", "Redeposit")
	CRMUsername       sql.NullString `json:"crm_username" db:"crm_username"`               // Username CRM yang menangani
	DepositPending    bool           `json:"deposit_pending" db:"deposit_pending"`         // Apakah ada deposit yang pending untuk member ini
	LastInteractionAt sql.NullTime   `json:"last_interaction_at" db:"last_interaction_at"` // Waktu interaksi terakhir (opsional, untuk sorting)
	MemberCreatedAt   time.Time      `json:"member_created_at" db:"member_created_at"`     // Kapan member dibuat (untuk sorting atau info)
}

// FollowupFilters menyimpan kriteria filter untuk query daftar followup.
type FollowupFilters struct {
	SearchTerm string   // Untuk pencarian berdasarkan username, email, nohp
	Status     []string // Filter berdasarkan status (bisa beberapa status)
	BrandID    int64    // Filter berdasarkan ID brand
	// Tambahkan filter lain jika perlu, misal CRM ID, Team ID, dll.
}

// Konstanta untuk status followup (jika diperlukan secara eksplisit)
const (
	StatusFollowupNewDeposit = "New Deposit"
	StatusFollowupRedeposit  = "Redeposit"
	StatusFollowupPending    = "Pending" // Mungkin ini bukan status member tapi status transaksi
)
