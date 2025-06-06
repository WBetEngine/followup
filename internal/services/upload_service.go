package services

import (
	"fmt"
	"strings"

	"followup/internal/models"
	"followup/internal/repository"
)

// UploadService mendefinisikan interface untuk logika bisnis terkait unggah data.
type UploadService interface {
	// ImportMembers memproses dan menyimpan sekumpulan data member.
	// Mengembalikan jumlah member yang baru diimpor, jumlah member yang dilewati karena duplikat nomor telepon,
	// jumlah member yang dilewati karena nomor telepon kosong, dan error jika ada.
	ImportMembers(data []models.MemberData, brandName string) (importedCount int, duplicateSkippedCount int, emptyPhoneSkippedCount int, err error)
}

// uploadService adalah implementasi dari UploadService.
type uploadService struct {
	memberRepo repository.MemberRepositoryInterface
}

// NewUploadService membuat instance baru dari uploadService.
func NewUploadService(memberRepo repository.MemberRepositoryInterface) UploadService {
	return &uploadService{memberRepo: memberRepo}
}

// ImportMembers mengimplementasikan logika untuk mengimpor data member.
// Memeriksa duplikasi berdasarkan phoneNumber dan brandName.
// Melewati baris jika phoneNumber kosong.
func (s *uploadService) ImportMembers(data []models.MemberData, brandName string) (importedCount int, duplicateSkippedCount int, emptyPhoneSkippedCount int, err error) {
	if len(data) == 0 {
		return 0, 0, 0, fmt.Errorf("tidak ada data member untuk diimpor")
	}

	var membersToInsert []models.MemberData

	for i := range data {
		data[i].BrandName = brandName // Pastikan brandName konsisten

		if strings.TrimSpace(data[i].PhoneNumber) == "" {
			emptyPhoneSkippedCount++
			continue // Lewati baris ini karena nomor telepon kosong
		}

		// Gunakan PhoneNumber untuk pengecekan duplikasi
		// MemberExists sekarang mengembalikan (bool, int, error). Kita abaikan int (ID member yang ada) di sini.
		exists, _, errCheck := s.memberRepo.MemberExists(data[i].PhoneNumber, data[i].BrandName)
		if errCheck != nil {
			// Jika ada error saat memeriksa, anggap sebagai error keseluruhan proses impor
			// Kembalikan jumlah yang sudah diproses (0 untuk imported, yang sudah dihitung untuk skipped)
			return 0, duplicateSkippedCount, emptyPhoneSkippedCount, fmt.Errorf("gagal memeriksa keberadaan member dengan telepon %s untuk brand %s: %w", data[i].PhoneNumber, data[i].BrandName, errCheck)
		}

		if exists {
			duplicateSkippedCount++
		} else {
			membersToInsert = append(membersToInsert, data[i])
		}
	}

	if len(membersToInsert) == 0 {
		// Tidak ada member baru untuk diinsert
		return 0, duplicateSkippedCount, emptyPhoneSkippedCount, nil
	}

	insertedThisBatch, errInsert := s.memberRepo.BulkInsertMembers(membersToInsert)
	importedCount = insertedThisBatch // Jumlah yang benar-benar berhasil diinsert
	if errInsert != nil {
		// Jika BulkInsertMembers mengembalikan error, importedCount dari sana mungkin sudah parsial.
		return importedCount, duplicateSkippedCount, emptyPhoneSkippedCount, fmt.Errorf("gagal menyimpan data member baru ke database: %w", errInsert)
	}

	return importedCount, duplicateSkippedCount, emptyPhoneSkippedCount, nil
}
