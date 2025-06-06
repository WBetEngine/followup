package models

// MenuKey mendefinisikan tipe untuk kunci identifikasi menu.
type MenuKey string

// Konstanta untuk kunci menu yang valid.
const (
	MenuDashboard      MenuKey = "MENU_DASHBOARD"
	MenuMemberList     MenuKey = "MENU_MEMBER_LIST"
	MenuBrandList      MenuKey = "MENU_BRAND_LIST"
	MenuUploadMember   MenuKey = "MENU_UPLOAD_MEMBER"
	MenuUserList       MenuKey = "MENU_USER_LIST"
	MenuTeam           MenuKey = "MENU_TEAM"
	MenuFollowUp       MenuKey = "MENU_FOLLOW_UP"
	MenuLangganan      MenuKey = "MENU_LANGGANAN"
	MenuInvalidNumber  MenuKey = "MENU_INVALID_NUMBER"
	MenuDepositList    MenuKey = "MENU_DEPOSIT_LIST"
	MenuWithdrawalList MenuKey = "MENU_WITHDRAWAL_LIST"
	MenuWallet         MenuKey = "MENU_WALLET"
	MenuSettings       MenuKey = "MENU_SETTINGS"
	MenuBonusList      MenuKey = "MENU_BONUS_LIST"
)

// AllMenuKeys mengembalikan slice dari semua MenuKey yang valid.
// Ini bisa digunakan untuk iterasi atau validasi.
func AllMenuKeys() []MenuKey {
	return []MenuKey{
		MenuDashboard,
		MenuMemberList,
		MenuBrandList,
		MenuUploadMember,
		MenuUserList,
		MenuTeam,
		MenuFollowUp,
		MenuLangganan,
		MenuInvalidNumber,
		MenuDepositList,
		MenuWithdrawalList,
		MenuWallet,
		MenuSettings,
		MenuBonusList,
	}
}

// Menu represents a menu item with its key, name, path, and icon.
// Ini bisa digunakan untuk membangun sidebar secara dinamis.
type MenuItem struct {
	Key   MenuKey
	Name  string
	Path  string
	Icon  string // Class ikon (misalnya, Font Awesome)
	Group string // Untuk mengelompokkan menu jika ada (opsional)
	Order int    // Untuk pengurutan tampilan (opsional)
}

// GetMenuItems mengembalikan daftar semua item menu yang terdefinisi.
// Ini akan menjadi sumber utama untuk membangun sidebar dan daftar izin.
func GetMenuItems() []MenuItem {
	return []MenuItem{
		{Key: MenuDashboard, Name: "Dashboard", Path: "/dashboard", Icon: "fas fa-home", Order: 10},

		{Key: MenuMemberList, Name: "Member", Path: "/member", Icon: "fas fa-users", Group: "Database", Order: 20},
		{Key: MenuBrandList, Name: "Brand", Path: "/brand", Icon: "fas fa-tag", Group: "Database", Order: 21},
		{Key: MenuUploadMember, Name: "Upload Member", Path: "/upload/excel", Icon: "fas fa-file-excel", Group: "Database", Order: 22},
		{Key: MenuUserList, Name: "User", Path: "/user", Icon: "fas fa-user-cog", Group: "Database", Order: 23},
		{Key: MenuTeam, Name: "Team", Path: "/team", Icon: "fas fa-users-cog", Group: "Database", Order: 24},

		{Key: MenuFollowUp, Name: "Follow Up", Path: "/followup", Icon: "fas fa-list-check", Group: "Relationship Management", Order: 30},
		{Key: MenuLangganan, Name: "Langganan", Path: "/langganan", Icon: "fas fa-id-badge", Group: "Relationship Management", Order: 31},
		{Key: MenuInvalidNumber, Name: "Invalid Number", Path: "/invalid-numbers", Icon: "fas fa-phone-slash", Group: "Relationship Management", Order: 32},

		{Key: MenuDepositList, Name: "Deposit", Path: "/deposit", Icon: "fas fa-money-bill-wave", Group: "Transaksi", Order: 40},
		{Key: MenuWithdrawalList, Name: "Withdrawal", Path: "/withdrawal", Icon: "fas fa-wallet", Group: "Transaksi", Order: 41},
		{Key: MenuWallet, Name: "Dompet", Path: "/wallet", Icon: "fas fa-credit-card", Group: "Transaksi", Order: 42},

		{Key: MenuSettings, Name: "Setting", Path: "/setting", Icon: "fas fa-cog", Order: 50},
		{Key: MenuBonusList, Name: "Bonus", Path: "/bonus", Icon: "fas fa-gift", Order: 60},
	}
}
