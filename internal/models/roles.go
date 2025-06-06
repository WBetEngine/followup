package models

import (
	"fmt"
)

// UserRole mendefinisikan tipe untuk peran pengguna.
type UserRole string

// Konstanta untuk peran pengguna yang valid.
const (
	SuperadminRole    UserRole = "superadmin"
	AdminRole         UserRole = "admin"
	TelemarketingRole UserRole = "telemarketing"
	CRMRole           UserRole = "crm"
	AdministratorRole UserRole = "administrator"
	// DefaultRole bisa ditambahkan jika diperlukan, atau peran tamu.
	// GuestRole UserRole = "guest"
)

// String mengembalikan representasi string dari UserRole.
// Ini berguna untuk logging atau penyimpanan jika tipe database adalah string.
func (r UserRole) String() string {
	return string(r)
}

// IsValid memeriksa apakah UserRole merupakan salah satu dari konstanta yang didefinisikan.
func (r UserRole) IsValid() bool {
	switch r {
	case SuperadminRole, AdminRole, TelemarketingRole, CRMRole, AdministratorRole:
		return true
	default:
		return false
	}
}

// GetValidRoles mengembalikan slice dari semua UserRole yang valid.
// Berguna untuk validasi atau membuat dropdown.
func GetValidRoles() []UserRole {
	return []UserRole{
		SuperadminRole,
		AdminRole,
		TelemarketingRole,
		CRMRole,
		AdministratorRole,
	}
}

// ParseUserRole mengkonversi string menjadi UserRole dan memvalidasinya.
// Mengembalikan error jika string peran tidak valid.
func ParseUserRole(roleStr string) (UserRole, error) {
	role := UserRole(roleStr) // Konversi string ke UserRole
	if !role.IsValid() {
		return "", fmt.Errorf("peran tidak valid: %s", roleStr) // Kembalikan error jika tidak valid
	}
	return role, nil
}
