package handlers

import (
	// "followup/internal/render" // Tidak perlu import render di sini jika tidak ada struct Render
	"followup/internal/services"
	"log"
	"net/http"
	"strconv"

	// kita butuh ini untuk memanggil fungsi render.TemplateWithBase

	"github.com/go-chi/chi/v5"
)

// BrandHandler struct holds the brand service for action methods.
// Metode render untuk halaman utama brand ada di page_handlers.go.
type BrandHandler struct {
	Service  services.BrandService
	PageSize int // Untuk paginasi, jika diperlukan nanti
}

// NewBrandHandler creates a new BrandHandler with necessary dependencies.
func NewBrandHandler(service services.BrandService) *BrandHandler {
	return &BrandHandler{
		Service:  service,
		PageSize: 50, // Default page size, bisa diubah sesuai kebutuhan
	}
}

// AddBrand handles the creation of a new brand.
func (h *BrandHandler) AddBrand(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/brand?error=Gagal+memproses+form", http.StatusSeeOther)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Redirect(w, r, "/brand?error=Nama+brand+tidak+boleh+kosong", http.StatusSeeOther)
		return
	}

	_, err := h.Service.CreateBrand(name)
	if err != nil {
		log.Printf("Error creating brand: %v", err)
		http.Redirect(w, r, "/brand?error=Gagal+menambahkan+brand:+server+error", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/brand?success=Brand+berhasil+ditambahkan", http.StatusSeeOther)
}

// UpdateBrand handles updating an existing brand's name.
func (h *BrandHandler) UpdateBrand(w http.ResponseWriter, r *http.Request) {
	brandIDStr := chi.URLParam(r, "id")
	brandID, err := strconv.ParseInt(brandIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/brand?error=ID+brand+tidak+valid", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/brand?error=Gagal+memproses+form", http.StatusSeeOther)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Redirect(w, r, "/brand?error=Nama+brand+tidak+boleh+kosong", http.StatusSeeOther)
		return
	}

	_, err = h.Service.UpdateBrand(brandID, name)
	if err != nil {
		log.Printf("Error updating brand %d: %v", brandID, err)
		http.Redirect(w, r, "/brand?error=Gagal+memperbarui+brand:+server+error", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/brand?success=Brand+berhasil+diperbarui", http.StatusSeeOther)
}

// DeleteBrand handles the deletion of a brand.
func (h *BrandHandler) DeleteBrand(w http.ResponseWriter, r *http.Request) {
	brandIDStr := chi.URLParam(r, "id")
	brandID, err := strconv.ParseInt(brandIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/brand?error=ID+brand+tidak+valid", http.StatusSeeOther)
		return
	}

	err = h.Service.DeleteBrand(brandID)
	if err != nil {
		log.Printf("Error deleting brand %d: %v", brandID, err)
		http.Redirect(w, r, "/brand?error=Gagal+menghapus+brand:+server+error", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/brand?success=Brand+berhasil+dihapus", http.StatusSeeOther)
}

// ViewBrandMembers (Placeholder - akan mengarahkan ke halaman member dengan filter brand)
// Saat ini hanya log dan redirect ke halaman member tanpa filter
// Nantinya akan dimodifikasi untuk menampilkan member berdasarkan brand.
func (h *BrandHandler) ViewBrandMembers(w http.ResponseWriter, r *http.Request) {
	brandIDStr := chi.URLParam(r, "id")
	brandID, err := strconv.ParseInt(brandIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/brand?error=ID+brand+tidak+valid", http.StatusSeeOther)
		return
	}

	brand, err := h.Service.GetBrandByID(brandID)
	if err != nil || brand == nil {
		log.Printf("Error getting brand %d for member view: %v", brandID, err)
		http.Redirect(w, r, "/brand?error=Brand+tidak+ditemukan", http.StatusSeeOther)
		return
	}

	// Redirect ke halaman member dengan filter brand (misal: /member?brand_name=NAMA_BRAND)
	log.Printf("Viewing members for brand ID: %d, Name: %s", brandID, brand.Name)
	http.Redirect(w, r, "/member?brand_name="+brand.Name, http.StatusSeeOther)
}
