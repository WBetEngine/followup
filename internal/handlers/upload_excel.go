package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"followup/internal/database"
	"followup/internal/models"
	"followup/internal/repository"
	"followup/internal/services"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

var (
	ipRegexUsername        = regexp.MustCompile(`IP:\s*([0-9\.]+)`)
	lastLoginRegexUsername = regexp.MustCompile(`Terakhir Terlihat:\s*(.+)`)
	// emailRegexUsername     = regexp.MustCompile(`([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`) // Tidak digunakan lagi di sini
)

// TempUploadSession menyimpan data upload sementara
type TempUploadSession struct {
	ID        string              `json:"id"`
	Data      []models.MemberData `json:"data"`
	BrandName string              `json:"brandName"`
	Timestamp time.Time           `json:"timestamp"`
}

// Simpan session upload sementara (dalam produksi seharusnya menggunakan Redis/database)
var tempUploadSessions = make(map[string]TempUploadSession)

// UploadExcelHandler menangani upload file Excel
func UploadExcelHandler(w http.ResponseWriter, r *http.Request) {
	// Cek metode
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Batasi ukuran upload ke 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		respondWithError(w, "Ukuran file terlalu besar (maksimum 10MB)")
		return
	}

	// Ambil file
	file, handler, err := r.FormFile("excelFile")
	if err != nil {
		respondWithError(w, "Gagal mendapatkan file")
		return
	}
	defer file.Close()

	// Ambil brandName dari form
	brandName := r.FormValue("brandName")
	if strings.TrimSpace(brandName) == "" {
		respondWithError(w, "Nama Brand wajib diisi")
		return
	}

	// Validasi tipe file
	if !isExcelFile(handler.Filename) {
		respondWithError(w, "Format file tidak valid. Hanya file .xlsx atau .xls yang diperbolehkan")
		return
	}

	// Simpan file sementara
	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, handler.Filename)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		respondWithError(w, "Gagal menyimpan file sementara")
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFilePath) // Hapus file sementara setelah selesai

	// Salin file ke file sementara
	fileBytes := make([]byte, handler.Size)
	if _, err := file.Read(fileBytes); err != nil {
		respondWithError(w, "Gagal membaca file")
		return
	}
	if _, err := tempFile.Write(fileBytes); err != nil {
		respondWithError(w, "Gagal menyimpan file sementara")
		return
	}
	tempFile.Close() // Tutup file sebelum dibuka oleh excelize

	// Buka file Excel
	xlsx, err := excelize.OpenFile(tempFilePath)
	if err != nil {
		respondWithError(w, "Gagal membuka file Excel: "+err.Error())
		return
	}
	defer xlsx.Close()

	// Ambil semua sheet
	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		respondWithError(w, "File Excel tidak memiliki sheet")
		return
	}

	// Baca data dari sheet pertama
	rows, err := xlsx.GetRows(sheets[0])
	if err != nil {
		respondWithError(w, "Gagal membaca data Excel: "+err.Error())
		return
	}

	// Cek apakah perlu melewati header
	skipHeader := r.FormValue("skipHeader") == "on"
	startRow := 0
	if skipHeader && len(rows) > 0 {
		startRow = 1
	}

	// Proses data
	var memberData []models.MemberData
	for i := startRow; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 9 { // minimal 9 kolom berdasarkan format yang ada
			if len(row) <= 1 || strings.TrimSpace(row[1]) == "" { // Jika kolom username kosong
				continue // lewati baris jika username tidak ada (asumsi username wajib)
			}
		}

		// Proses setiap baris, sekarang teruskan brandName
		member := processMemberRow(row, brandName)
		memberData = append(memberData, member)
	}

	if len(memberData) == 0 {
		respondWithError(w, "Tidak ada data valid ditemukan di file Excel")
		return
	}

	// Buat session ID
	sessionID := uuid.New().String()

	// Simpan ke session sementara
	tempUploadSessions[sessionID] = TempUploadSession{
		ID:        sessionID,
		Data:      memberData,
		BrandName: brandName,
		Timestamp: time.Now(),
	}

	// Bersihkan session lama (> 30 menit)
	cleanupOldSessions()

	log.Printf("[UPLOAD_EXCEL] Generated Session ID: %s for Brand: %s", sessionID, brandName)

	// Kirim respons sukses dengan preview data
	response := map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Berhasil memproses %d data", len(memberData)),
		"data":      memberData,
		"sessionId": sessionID,
		"brandName": brandName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ImportExcelHandler mengimpor data Excel yang sudah divalidasi ke database
func ImportExcelHandler(w http.ResponseWriter, r *http.Request) {
	// Cek metode
	if r.Method != http.MethodPost {
		respondWithError(w, "Metode tidak diizinkan")
		return
	}

	// DEBUG: Log Content-Type header
	contentType := r.Header.Get("Content-Type")
	log.Printf("[IMPORT_EXCEL] Received Content-Type: %s", contentType)

	// Coba parse sebagai multipart form secara eksplisit
	// defaultMaxMemory adalah 32 << 20 (32 MB)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		// Jika error BUKAN karena body sudah dibaca atau tipe salah (misal application/x-www-form-urlencoded)
		// maka ini adalah error parsing yang signifikan.
		if err != http.ErrNotMultipart && err != http.ErrMissingBoundary {
			log.Printf("[IMPORT_EXCEL] Error saat ParseMultipartForm: %v", err)
			// Jika ParseMultipartForm gagal total, r.FormValue juga akan gagal.
			// Mungkin ada baiknya mencoba ParseForm biasa sebagai fallback jika ini bukan multipart.
			if err := r.ParseForm(); err != nil {
				log.Printf("[IMPORT_EXCEL] Error saat fallback ParseForm: %v", err)
				respondWithError(w, fmt.Sprintf("Gagal memparsing form data (multipart dan fallback): %s", err.Error()))
				return
			}
		} else {
			// Jika ini bukan multipart, ParseForm() akan dipanggil oleh FormValue nanti
			// atau kita bisa panggil ParseForm() di sini.
			log.Printf("[IMPORT_EXCEL] Bukan request multipart atau boundary hilang, mencoba ParseForm biasa. Original error: %v", err)
			if err := r.ParseForm(); err != nil {
				log.Printf("[IMPORT_EXCEL] Error saat ParseForm setelah cek multipart: %v", err)
				respondWithError(w, fmt.Sprintf("Gagal memparsing form data: %s", err.Error()))
				return
			}
		}
	} else {
		log.Println("[IMPORT_EXCEL] Berhasil ParseMultipartForm.")
	}

	// DEBUG: Log seluruh r.Form setelah semua upaya parsing
	log.Printf("[IMPORT_EXCEL] r.Form after all parsing attempts: %v", r.Form)
	if r.MultipartForm != nil {
		log.Printf("[IMPORT_EXCEL] r.MultipartForm.Value after parsing: %v", r.MultipartForm.Value)
	}

	// Ambil session ID
	sessionID := r.FormValue("sessionId")
	brandNameFromForm := r.FormValue("brandName")
	log.Printf("[IMPORT_EXCEL] Received Session ID (from FormValue): '%s', Brand (from FormValue): '%s'", sessionID, brandNameFromForm)

	if sessionID == "" {
		log.Printf("[IMPORT_EXCEL] Error: Received empty session ID ('%s') or brandName ('%s') from form.", sessionID, brandNameFromForm)
		respondWithError(w, "Session ID atau Nama Brand tidak valid (kosong setelah parsing)")
		return
	}

	// Ambil brandName dari form (dikirim oleh JavaScript saat konfirmasi)
	// Perhatikan: Jika brandName tidak diubah oleh user saat konfirmasi,
	// kita bisa juga mengambilnya dari session. Namun, mengambil dari form lebih straightforward
	// karena JS sudah menambahkannya ke payload konfirmasi.
	if strings.TrimSpace(brandNameFromForm) == "" {
		// Fallback ke brandName dari session jika tidak ada di form (seharusnya ada)
		sessionForFallback, existsFallback := tempUploadSessions[sessionID]
		if !existsFallback {
			respondWithError(w, "Session upload tidak ditemukan atau sudah kadaluwarsa")
			return
		}
		brandNameFromForm = sessionForFallback.BrandName
		if strings.TrimSpace(brandNameFromForm) == "" {
			respondWithError(w, "Nama Brand tidak ditemukan untuk impor")
			return
		}
	}

	// Cek apakah session ada
	session, exists := tempUploadSessions[sessionID]
	if !exists {
		log.Printf("[IMPORT_EXCEL] Error: Session ID %s not found in tempUploadSessions.", sessionID)
		respondWithError(w, "Session upload tidak ditemukan atau sudah kadaluwarsa")
		return
	}

	// --- Inisialisasi Service dan Repository (SEMENTARA, idealnya di-inject) ---
	db := database.GetDB()
	memberRepo := repository.NewMemberRepository(db)
	uploadService := services.NewUploadService(memberRepo)
	// --- Akhir Inisialisasi Sementara ---

	// Teruskan brandName ke service
	importedCount, duplicateSkippedCount, emptyPhoneSkippedCount, err := uploadService.ImportMembers(session.Data, brandNameFromForm)

	// Hapus session setelah upaya impor
	delete(tempUploadSessions, sessionID)

	if err != nil {
		// Kembalikan error sebagai JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Gagal mengimpor data: %s", err.Error()),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Kirim respons sukses JSON
	var messages []string
	messages = append(messages, fmt.Sprintf("Berhasil mengimpor %d data member baru ke database untuk brand '%s'.", importedCount, brandNameFromForm))
	if duplicateSkippedCount > 0 {
		messages = append(messages, fmt.Sprintf("%d data dilewati karena nomor telepon duplikat.", duplicateSkippedCount))
	}
	if emptyPhoneSkippedCount > 0 {
		messages = append(messages, fmt.Sprintf("%d data dilewati karena nomor telepon kosong.", emptyPhoneSkippedCount))
	}
	finalMessage := strings.Join(messages, " ")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"success":                true,
		"message":                finalMessage,
		"importedCount":          importedCount,
		"duplicateSkippedCount":  duplicateSkippedCount,
		"emptyPhoneSkippedCount": emptyPhoneSkippedCount,
	}
	json.NewEncoder(w).Encode(response)
}

// parseMembershipColumn memecah kolom membership menjadi status, nomor HP, dan email.
func parseMembershipColumn(column string) map[string]string {
	result := map[string]string{
		"status":          "", // Akan menjadi MembershipStatus
		"phoneNumber":     "",
		"membershipEmail": "",
	}

	lines := strings.Split(strings.ReplaceAll(column, "\r\n", "\n"), "\n")

	if len(lines) > 0 {
		result["status"] = strings.TrimSpace(lines[0])
	}
	if len(lines) > 1 {
		phone := strings.TrimSpace(lines[1])
		// Hapus karakter non-digit dari nomor telepon sebelum validasi
		nonDigitRegex := regexp.MustCompile(`[^\d]`)
		cleanedPhone := nonDigitRegex.ReplaceAllString(phone, "")

		if cleanedPhone != "" {
			// Regex untuk nomor HP Indonesia (umumnya 8-15 digit setelah dibersihkan)
			phoneRegex := regexp.MustCompile(`^\d{8,15}$`)
			if phoneRegex.MatchString(cleanedPhone) {
				if strings.HasPrefix(cleanedPhone, "8") {
					result["phoneNumber"] = "0" + cleanedPhone
				} else {
					result["phoneNumber"] = cleanedPhone
				}
			} else {
				result["phoneNumber"] = cleanedPhone
			}
		}
	}
	if len(lines) > 2 {
		emailLine := strings.TrimSpace(lines[2])
		if emailLine != "" && emailLine != "---" {
			// Validasi email sederhana
			emailParseRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
			if emailParseRegex.MatchString(emailLine) {
				result["membershipEmail"] = emailLine
			}
		}
	}
	return result
}

// processMemberRow memproses satu baris data dari Excel menjadi struct MemberData
// brandName ditambahkan sebagai parameter
func processMemberRow(row []string, brandName string) models.MemberData {
	// Default value jika kolom tidak ada atau kosong
	getCol := func(index int) string {
		if index < len(row) {
			return strings.TrimSpace(row[index])
		}
		return ""
	}

	// Parsing kolom yang kompleks
	usernameInfo := parseUsernameColumn(getCol(1)) // Kolom Username (B)
	bankInfo := parseBankColumn(getCol(3))         // Kolom Bank (D)
	saldoInfo := parseSaldoColumn(getCol(4))       // Kolom Saldo (E)

	// membershipInfo sudah dideklarasikan di atas dan menggunakan getCol(2)
	// actualTurnoverData tidak lagi digunakan karena turnover diambil dari saldoInfo

	membershipInfo := parseMembershipColumn(getCol(2))

	member := models.MemberData{
		Username: usernameInfo["username"],
		// Menggunakan key yang benar dari parseMembershipColumn dan field yang benar di MemberData
		MembershipEmail:  membershipInfo["membershipEmail"],
		PhoneNumber:      membershipInfo["phoneNumber"],
		BankName:         bankInfo["bank"],
		AccountName:      bankInfo["name"],    // Menggunakan key "name" dari parseBankColumn
		AccountNo:        bankInfo["account"], // Memperbaiki typo dari "account\" menjadi "account"
		Saldo:            saldoInfo["balance"],
		Turnover:         saldoInfo["turnover"],                    // Diambil dari parseSaldoColumn
		WinLoss:          getCol(5),                                // Kolom F adalah Tanggal Bergabung, WinLoss mungkin perlu disesuaikan jika kolomnya berbeda. Sesuai komentar lama: "Kolom Menang Kalah (F) - JoinDate dipindah, ini jadi WL"
		MembershipStatus: membershipInfo["status"],                 // Diambil dari parseMembershipColumn
		Referral:         getCol(7),                                // Kolom Referral (H)
		Uplink:           strings.ReplaceAll(getCol(8), "\n", " "), // Kolom Uplink (I), bersihkan newline
		BrandName:        brandName,
	}

	if ip, ok := usernameInfo["ip"]; ok {
		member.IPAddress = ip // Diubah dari IP
	}
	if lastLoginVal, ok := usernameInfo["last_login"]; ok {
		if lastLoginVal != "" {
			member.LastLogin = &lastLoginVal // Menjadi pointer
		}
	}

	// Penanganan untuk JoinDate (sekarang adalah row[5] karena WL di kolom F asli adalah Date)
	// Indeks kolom disesuaikan dengan contoh format baru:
	// A: No, B: Username (+IP, Last Login), C: TO, D: Bank (+AccName, AccNo), E: Saldo (+WD), F: TGL JOIN, G: Membership (+Email, Phone), H: Referral, I: Uplink
	// PERHATIAN: Komentar di atas mungkin tidak lagi sepenuhnya akurat setelah perubahan logika parsing.
	// Pastikan getCol(5) memang benar untuk JoinDate.
	joinDateStr := getCol(5)
	if joinDateStr != "" {
		member.JoinDate = &joinDateStr // Menjadi pointer
	}

	// Logika status member berdasarkan saldo (dari processSaldoColumn)
	if status, ok := saldoInfo["status"]; ok {
		member.Status = status // Status: New Deposit / Redeposit
	} else {
		// Default status jika tidak bisa ditentukan dari saldoInfo (misalnya saldo 0)
		if member.Saldo == "0" || member.Saldo == "" {
			member.Status = "New Deposit"
		} else {
			member.Status = "Redeposit"
		}
	}

	return member
}

// parseUsernameColumn mem-parse kolom username yang mungkin berisi IP dan tanggal login terakhir.
func parseUsernameColumn(column string) map[string]string {
	result := map[string]string{
		"username":   "",
		"ip":         "",
		"last_login": "",
		// "email":      "", // Dihapus
	}

	lines := strings.Split(strings.ReplaceAll(column, "\r\n", "\n"), "\n")

	if len(lines) > 0 {
		result["username"] = strings.TrimSpace(lines[0])
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "IP:") {
			matches := ipRegexUsername.FindStringSubmatch(line)
			if len(matches) > 1 {
				result["ip"] = matches[1]
			}
		}

		if strings.HasPrefix(line, "Terakhir Terlihat:") {
			matches := lastLoginRegexUsername.FindStringSubmatch(line)
			if len(matches) > 1 {
				result["last_login"] = matches[1]
			}
		}
		// Logika pencarian email dihapus dari sini
	}
	return result
}

// parseBankColumn mem-parse kolom bank yang mungkin berisi nama akun dan nomor akun.
func parseBankColumn(column string) map[string]string {
	result := map[string]string{
		"bank":    "",
		"name":    "",
		"account": "",
	}

	// Pisahkan berdasarkan baris baru
	lines := strings.Split(strings.ReplaceAll(column, "\r\n", "\n"), "\n")

	// Biasanya baris pertama adalah nama bank
	if len(lines) > 0 {
		result["bank"] = strings.TrimSpace(lines[0])
	}

	// Baris kedua biasanya nama pemilik rekening
	if len(lines) > 1 {
		result["name"] = strings.TrimSpace(lines[1])
	}

	// Baris ketiga biasanya nomor rekening
	if len(lines) > 2 {
		// Cari pola nomor rekening (angka)
		numRegex := regexp.MustCompile(`(\d+)`)
		matches := numRegex.FindStringSubmatch(lines[2])
		if len(matches) > 1 {
			result["account"] = matches[1]
		}
	}

	return result
}

// parseSaldoColumn mem-parse kolom saldo dan menentukan status deposit.
func parseSaldoColumn(column string) map[string]string {
	result := map[string]string{
		"balance": "",
		"status":  "",
	}

	// Pisahkan berdasarkan baris baru
	lines := strings.Split(strings.ReplaceAll(column, "\r\n", "\n"), "\n")

	// Biasanya baris pertama adalah saldo
	if len(lines) > 0 {
		saldoRegex := regexp.MustCompile(`Rp\s*([0-9,.]+)`)
		matches := saldoRegex.FindStringSubmatch(lines[0])
		if len(matches) > 1 {
			result["balance"] = "Rp " + matches[1]
		}
	}

	// Cari turnover
	for _, line := range lines {
		if strings.Contains(line, "Turnover:") {
			turnoverRegex := regexp.MustCompile(`Turnover:\s*Rp\s*([0-9,.]+)`)
			matches := turnoverRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				result["turnover"] = "Rp " + matches[1]
			}
		}
	}

	// Cari kemenangan/kekalahan
	for _, line := range lines {
		if strings.Contains(line, "Kemenangan") || strings.Contains(line, "Kekalahan") {
			winLossRegex := regexp.MustCompile(`Kemenangan\s*(\d+)\s*Kekalahan\s*(\d+)`)
			matches := winLossRegex.FindStringSubmatch(line)
			if len(matches) > 2 {
				result["win_loss"] = fmt.Sprintf("Win: %s, Loss: %s", matches[1], matches[2])
			}
		}
	}

	// Cari points
	for _, line := range lines {
		if strings.Contains(line, "Points:") {
			pointsRegex := regexp.MustCompile(`Points:\s*(\d+)`)
			matches := pointsRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				result["points"] = matches[1]
			}
		}
	}

	// Logika status berdasarkan saldo
	cleanedSaldo := strings.TrimSpace(result["balance"])
	cleanedSaldo = strings.ReplaceAll(cleanedSaldo, "Rp", "")
	cleanedSaldo = strings.ReplaceAll(cleanedSaldo, ".", "") // Hapus pemisah ribuan jika ada
	cleanedSaldo = strings.ReplaceAll(cleanedSaldo, ",", "") // Hapus pemisah desimal jika ada (asumsi tidak ada desimal signifikan untuk logika ini)
	cleanedSaldo = strings.TrimSpace(cleanedSaldo)

	// Cek apakah saldo adalah "0"
	isZeroBalance := cleanedSaldo == "0" || cleanedSaldo == ""

	if isZeroBalance {
		result["status"] = "New Deposit"
	} else {
		// Cek apakah ada angka selain 0
		hasValue := false
		for _, char := range cleanedSaldo {
			if char >= '1' && char <= '9' {
				hasValue = true
				break
			}
		}
		if hasValue {
			result["status"] = "Redeposit"
		}
		// Jika cleanedSaldo tidak kosong, bukan "0", tapi tidak memiliki angka > 0 (misal "Rp "),
		// status mungkin tidak berubah atau default ke sesuatu.
		// Untuk saat ini, jika tidak zero dan ada nilai > 0, maka "Redeposit".
		// Jika status asli dari Excel penting dan tidak ingin ditimpa, logika ini perlu disesuaikan.
	}

	return result
}

// Fungsi bantuan untuk validasi dan respons

// isExcelFile memeriksa apakah file adalah file Excel (.xlsx atau .xls)
func isExcelFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".xlsx" || ext == ".xls"
}

// respondWithError mengirim respons error dalam format JSON
func respondWithError(w http.ResponseWriter, message string) {
	response := map[string]interface{}{
		"success": false,
		"message": message,
	}

	w.Header().Set("Content-Type", "application/json")
	// Selalu kirim 200 OK, status error ditangani oleh payload JSON "success": false
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// cleanupOldSessions membersihkan session upload yang sudah lama
func cleanupOldSessions() {
	threshold := time.Now().Add(-30 * time.Minute)

	for id, session := range tempUploadSessions {
		if session.Timestamp.Before(threshold) {
			delete(tempUploadSessions, id)
		}
	}
}
