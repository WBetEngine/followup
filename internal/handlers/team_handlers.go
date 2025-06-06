package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"followup/internal/auth"
	"followup/internal/models"
	"followup/internal/services"

	"github.com/go-chi/chi/v5"
	// "gopkg.in/validator.v2" // Komentari untuk sementara
)

// TeamHandler menangani request HTTP terkait manajemen tim.
type TeamHandler struct {
	service services.TeamServiceInterface
	// authService dihapus, kita akan menggunakan auth.GetUserFromRequest langsung
}

// NewTeamHandler membuat instance baru dari TeamHandler.
func NewTeamHandler(ts services.TeamServiceInterface) *TeamHandler { // Hapus authService dari parameter
	return &TeamHandler{
		service: ts,
	}
}

// respondJSON adalah helper untuk mengirim respons JSON.
func (h *TeamHandler) respondJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON response: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		// Hindari mengirimkan detail error internal ke klien dalam produksi
		// cukup pesan generik.
		if _, writeErr := w.Write([]byte(`{"message": "Gagal memproses respons server"}`)); writeErr != nil {
			log.Printf("Error writing generic error response: %v", writeErr)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if _, err := w.Write(response); err != nil {
		log.Printf("Error writing JSON response: %v", err)
	}
}

// respondError adalah helper untuk mengirim error JSON.
func (h *TeamHandler) respondError(w http.ResponseWriter, statusCode int, message string, validationErrors map[string]string) {
	responsePayload := map[string]interface{}{"message": message}
	if len(validationErrors) > 0 {
		responsePayload["errors"] = validationErrors
	}
	h.respondJSON(w, statusCode, responsePayload)
}

// CreateTeamHandler menangani pembuatan tim baru.
// POST /api/teams
func (h *TeamHandler) CreateTeamHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := auth.GetUserFromRequest(r)
	if err != nil || currentUser == nil {
		h.respondError(w, http.StatusUnauthorized, "Akses ditolak. Silakan login.", nil)
		return
	}
	if currentUser.Role != models.SuperadminRole {
		h.respondError(w, http.StatusForbidden, "Hanya Superadmin yang dapat membuat tim.", nil)
		return
	}

	var req models.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Request body tidak valid atau tidak berformat JSON.", nil)
		return
	}
	defer r.Body.Close()

	// Validasi manual dasar
	valErrors := make(map[string]string)
	if strings.TrimSpace(req.Name) == "" {
		valErrors["name"] = "Nama tim tidak boleh kosong."
	}
	if req.AdminUserID <= 0 {
		valErrors["admin_user_id"] = "Admin User ID tidak boleh kosong dan harus valid."
	}
	if len(valErrors) > 0 {
		h.respondError(w, http.StatusUnprocessableEntity, "Data tidak valid.", valErrors)
		return
	}
	// if err := validator.Validate(req); err != nil { // Komentari validasi eksternal untuk sementara
	// h.respondError(w, http.StatusUnprocessableEntity, "Validasi gagal.", common.FormatValidatorError(err))
	// return
	// }

	var description sql.NullString
	if req.Description != nil {
		description.String = *req.Description
		description.Valid = true
	}

	team, valErrorsService, serviceErr := h.service.CreateTeam(r.Context(), req.Name, description, req.AdminUserID)
	if len(valErrorsService) > 0 {
		h.respondError(w, http.StatusUnprocessableEntity, "Data tim tidak valid.", valErrorsService)
		return
	}
	if serviceErr != nil {
		if errors.Is(serviceErr, models.ErrTeamNameTaken) || errors.Is(serviceErr, models.ErrUserNotFound) {
			h.respondError(w, http.StatusConflict, serviceErr.Error(), nil)
		} else {
			log.Printf("Error creating team in service: %v", serviceErr)
			h.respondError(w, http.StatusInternalServerError, "Gagal membuat tim di server.", nil)
		}
		return
	}

	h.respondJSON(w, http.StatusCreated, team)
}

// GetTeamHandler menangani pengambilan detail tim.
// GET /api/teams/{teamID}
func (h *TeamHandler) GetTeamHandler(w http.ResponseWriter, r *http.Request) {
	teamIDStr := chi.URLParam(r, "teamID")
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil || teamID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Team ID tidak valid.", nil)
		return
	}

	team, err := h.service.GetTeamDetails(r.Context(), teamID)
	if err != nil {
		if errors.Is(err, models.ErrTeamNotFound) {
			h.respondError(w, http.StatusNotFound, "Tim tidak ditemukan.", nil)
		} else {
			log.Printf("Error getting team %d: %v", teamID, err)
			h.respondError(w, http.StatusInternalServerError, "Gagal mengambil data tim.", nil)
		}
		return
	}
	h.respondJSON(w, http.StatusOK, team)
}

// ListTeamsHandler menangani pengambilan daftar tim.
// GET /api/teams
func (h *TeamHandler) ListTeamsHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("search")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if limit > 100 { // Batas atas untuk limit
		limit = 100
	}

	teams, totalRecords, err := h.service.ListTeams(r.Context(), searchTerm, page, limit)
	if err != nil {
		log.Printf("Error listing teams: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Gagal mengambil daftar tim.", nil)
		return
	}

	totalPages := 0
	if totalRecords > 0 && limit > 0 {
		totalPages = (totalRecords + limit - 1) / limit
	}

	response := map[string]interface{}{
		"teams":         teams,
		"total_records": totalRecords, // snake_case untuk konsistensi JSON
		"current_page":  page,
		"per_page":      limit,
		"total_pages":   totalPages,
	}
	h.respondJSON(w, http.StatusOK, response)
}

// UpdateTeamHandler menangani pembaruan tim.
// PUT /api/teams/{teamID}
func (h *TeamHandler) UpdateTeamHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := auth.GetUserFromRequest(r)
	if err != nil || currentUser == nil {
		h.respondError(w, http.StatusUnauthorized, "Akses ditolak. Silakan login.", nil)
		return
	}

	teamIDStr := chi.URLParam(r, "teamID")
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil || teamID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Team ID tidak valid.", nil)
		return
	}

	teamToCheck, err := h.service.GetTeamDetails(r.Context(), teamID)
	if err != nil {
		if errors.Is(err, models.ErrTeamNotFound) {
			h.respondError(w, http.StatusNotFound, "Tim yang akan diupdate tidak ditemukan.", nil)
		} else {
			log.Printf("UpdateTeamHandler: Error fetching team %d for auth check: %v", teamID, err)
			h.respondError(w, http.StatusInternalServerError, "Gagal memverifikasi izin update.", nil)
		}
		return
	}

	if currentUser.Role != models.SuperadminRole && teamToCheck.Team.AdminUserID != int64(currentUser.UserID) {
		h.respondError(w, http.StatusForbidden, "Anda tidak memiliki izin untuk memperbarui tim ini.", nil)
		return
	}

	var req models.UpdateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Request body tidak valid atau tidak berformat JSON.", nil)
		return
	}
	defer r.Body.Close()

	// Validasi manual dasar jika diperlukan, atau biarkan service yang menangani
	// if err := validator.Validate(req); err != nil { // Komentari
	// h.respondError(w, http.StatusUnprocessableEntity, "Validasi gagal.", common.FormatValidatorError(err))
	// return
	// }

	var namePtr *string
	if req.Name != nil {
		namePtr = req.Name
	}

	var descPtr *sql.NullString
	if req.Description != nil {
		desc := sql.NullString{String: *req.Description, Valid: true}
		descPtr = &desc
	}

	var adminIDPtr *int64
	if req.AdminUserID != nil {
		adminIDPtr = req.AdminUserID
	}

	updatedTeam, valErrors, serviceErr := h.service.UpdateTeam(r.Context(), teamID, namePtr, descPtr, adminIDPtr)
	if len(valErrors) > 0 {
		h.respondError(w, http.StatusUnprocessableEntity, "Data pembaruan tim tidak valid.", valErrors)
		return
	}
	if serviceErr != nil {
		if errors.Is(serviceErr, models.ErrTeamNotFound) {
			h.respondError(w, http.StatusNotFound, serviceErr.Error(), nil)
		} else if errors.Is(serviceErr, models.ErrTeamNameTaken) || errors.Is(serviceErr, models.ErrUserNotFound) {
			h.respondError(w, http.StatusConflict, serviceErr.Error(), nil)
		} else {
			log.Printf("Error updating team %d in service: %v", teamID, serviceErr)
			h.respondError(w, http.StatusInternalServerError, "Gagal memperbarui tim di server.", nil)
		}
		return
	}
	h.respondJSON(w, http.StatusOK, updatedTeam)
}

// DeleteTeamHandler menangani penghapusan tim.
// DELETE /api/teams/{teamID}
func (h *TeamHandler) DeleteTeamHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := auth.GetUserFromRequest(r)
	if err != nil || currentUser == nil {
		h.respondError(w, http.StatusUnauthorized, "Akses ditolak. Silakan login.", nil)
		return
	}
	if currentUser.Role != models.SuperadminRole {
		h.respondError(w, http.StatusForbidden, "Hanya Superadmin yang dapat menghapus tim.", nil)
		return
	}

	teamIDStr := chi.URLParam(r, "teamID")
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil || teamID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Team ID tidak valid.", nil)
		return
	}

	serviceErr := h.service.DeleteTeam(r.Context(), teamID)
	if serviceErr != nil {
		if errors.Is(serviceErr, models.ErrTeamNotFound) {
			h.respondError(w, http.StatusNotFound, "Tim tidak ditemukan.", nil)
		} else if errors.Is(serviceErr, models.ErrTeamHasMembers) {
			h.respondError(w, http.StatusConflict, serviceErr.Error(), nil)
		} else {
			log.Printf("Error deleting team %d: %v", teamID, serviceErr)
			h.respondError(w, http.StatusInternalServerError, "Gagal menghapus tim.", nil)
		}
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Tim ID %d berhasil dihapus.", teamID)})
}

// AddTeamMemberHandler menangani penambahan anggota ke tim.
// POST /api/teams/{teamID}/members
func (h *TeamHandler) AddTeamMemberHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := auth.GetUserFromRequest(r)
	if err != nil || currentUser == nil {
		h.respondError(w, http.StatusUnauthorized, "Akses ditolak. Silakan login.", nil)
		return
	}

	teamIDStr := chi.URLParam(r, "teamID")
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil || teamID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Team ID tidak valid.", nil)
		return
	}

	isSuperAdmin := currentUser.Role == models.SuperadminRole
	var isAdminOfThisTeam bool = false
	teamDetailsForAuth, errAdminCheck := h.service.GetTeamDetails(r.Context(), teamID)

	if errAdminCheck != nil {
		if errors.Is(errAdminCheck, models.ErrTeamNotFound) {
			h.respondError(w, http.StatusNotFound, "Tim target untuk menambah anggota tidak ditemukan.", nil)
			return
		}
		log.Printf("AddTeamMemberHandler: Error checking team admin status for team %d, user %d: %v", teamID, currentUser.UserID, errAdminCheck)
		h.respondError(w, http.StatusInternalServerError, "Gagal memverifikasi izin penambahan anggota.", nil)
		return
	}
	if teamDetailsForAuth != nil {
		isAdminOfThisTeam = teamDetailsForAuth.Team.AdminUserID == int64(currentUser.UserID)
	}

	if !isSuperAdmin && !isAdminOfThisTeam {
		h.respondError(w, http.StatusForbidden, "Hanya Superadmin atau Admin tim ini yang dapat menambahkan anggota.", nil)
		return
	}

	var req models.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Request body tidak valid atau tidak berformat JSON.", nil)
		return
	}
	defer r.Body.Close()

	if req.UserID <= 0 {
		h.respondError(w, http.StatusUnprocessableEntity, "Data tidak valid.", map[string]string{"user_id": "User ID tidak boleh kosong."})
		return
	}

	memberDetail, valErrors, serviceErr := h.service.AddMemberToTeam(r.Context(), teamID, req.UserID)
	if len(valErrors) > 0 {
		h.respondError(w, http.StatusUnprocessableEntity, "Gagal menambahkan anggota.", valErrors)
		return
	}
	if serviceErr != nil {
		if errors.Is(serviceErr, models.ErrTeamNotFound) || errors.Is(serviceErr, models.ErrUserNotFound) {
			h.respondError(w, http.StatusNotFound, serviceErr.Error(), nil)
		} else if errors.Is(serviceErr, models.ErrUserAlreadyInTeam) {
			h.respondError(w, http.StatusConflict, serviceErr.Error(), nil)
		} else {
			log.Printf("Error adding member %d to team %d in service: %v", req.UserID, teamID, serviceErr)
			h.respondError(w, http.StatusInternalServerError, "Gagal menambahkan anggota ke server.", nil)
		}
		return
	}
	h.respondJSON(w, http.StatusCreated, memberDetail)
}

// RemoveTeamMemberHandler menangani penghapusan anggota dari tim.
// DELETE /api/teams/{teamID}/members/{userID}
func (h *TeamHandler) RemoveTeamMemberHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := auth.GetUserFromRequest(r)
	if err != nil || currentUser == nil {
		h.respondError(w, http.StatusUnauthorized, "Akses ditolak. Silakan login.", nil)
		return
	}

	teamIDStr := chi.URLParam(r, "teamID")
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil || teamID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Team ID tidak valid.", nil)
		return
	}
	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		h.respondError(w, http.StatusBadRequest, "User ID tidak valid.", nil)
		return
	}

	isSuperAdmin := currentUser.Role == models.SuperadminRole
	var isAdminOfThisTeam bool = false
	teamDetails, errAdminCheck := h.service.GetTeamDetails(r.Context(), teamID)

	if errAdminCheck != nil {
		if errors.Is(errAdminCheck, models.ErrTeamNotFound) {
			h.respondError(w, http.StatusNotFound, "Tim target untuk menghapus anggota tidak ditemukan.", nil)
			return
		}
		log.Printf("RemoveTeamMemberHandler: Error checking team admin status for team %d, user %d: %v", teamID, currentUser.UserID, errAdminCheck)
		h.respondError(w, http.StatusInternalServerError, "Gagal memverifikasi izin penghapusan anggota.", nil)
		return
	}
	if teamDetails != nil {
		isAdminOfThisTeam = teamDetails.Team.AdminUserID == int64(currentUser.UserID)
	}

	if !isSuperAdmin && !isAdminOfThisTeam {
		h.respondError(w, http.StatusForbidden, "Hanya Superadmin atau Admin tim ini yang dapat menghapus anggota.", nil)
		return
	}

	serviceErr := h.service.RemoveMemberFromTeam(r.Context(), teamID, userID)
	if serviceErr != nil {
		if errors.Is(serviceErr, models.ErrTeamNotFound) || errors.Is(serviceErr, models.ErrUserNotFound) || errors.Is(serviceErr, models.ErrMemberNotFoundInTeam) {
			h.respondError(w, http.StatusNotFound, serviceErr.Error(), nil)
		} else if errors.Is(serviceErr, models.ErrCannotRemoveAdmin) {
			h.respondError(w, http.StatusForbidden, serviceErr.Error(), nil)
		} else {
			log.Printf("Error removing member %d from team %d: %v", userID, teamID, serviceErr)
			h.respondError(w, http.StatusInternalServerError, "Gagal menghapus anggota.", nil)
		}
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Anggota ID %d berhasil dihapus dari tim ID %d.", userID, teamID)})
}

// GetTeamMembersHandler menangani pengambilan daftar anggota tim.
// GET /api/teams/{teamID}/members
func (h *TeamHandler) GetTeamMembersHandler(w http.ResponseWriter, r *http.Request) {
	teamIDStr := chi.URLParam(r, "teamID")
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil || teamID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Team ID tidak valid.", nil)
		return
	}

	members, err := h.service.GetTeamMembers(r.Context(), teamID)
	if err != nil {
		if errors.Is(err, models.ErrTeamNotFound) {
			h.respondError(w, http.StatusNotFound, "Tim tidak ditemukan saat mengambil anggota.", nil)
		} else {
			log.Printf("Error getting members for team %d: %v", teamID, err)
			h.respondError(w, http.StatusInternalServerError, "Gagal mengambil daftar anggota tim.", nil)
		}
		return
	}
	h.respondJSON(w, http.StatusOK, members)
}

// GetAssignableUsersForTeamHandler mengembalikan daftar user yang bisa ditambahkan ke tim.
// GET /api/users/assignable-to-team
func (h *TeamHandler) GetAssignableUsersForTeamHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("search")
	limitStr := r.URL.Query().Get("limit")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	users, err := h.service.GetUsersAvailableForTeamMembership(r.Context(), searchTerm, limit)
	if err != nil {
		log.Printf("Error getting assignable users for team membership: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Gagal mengambil daftar pengguna yang dapat ditambahkan.", nil)
		return
	}
	h.respondJSON(w, http.StatusOK, users)
}

// GetAssignableAdminsForTeamHandler mengembalikan daftar user yang bisa dijadikan admin tim.
// GET /api/users/assignable-as-admin
func (h *TeamHandler) GetAssignableAdminsForTeamHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("search")
	currentAdminIDStr := r.URL.Query().Get("current_admin_id")
	limitStr := r.URL.Query().Get("limit")

	currentAdminID, _ := strconv.ParseInt(currentAdminIDStr, 10, 64)

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	users, err := h.service.GetUsersAvailableForTeamAdmin(r.Context(), currentAdminID, searchTerm, limit)
	if err != nil {
		log.Printf("Error getting assignable admins for team: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Gagal mengambil daftar calon admin.", nil)
		return
	}
	h.respondJSON(w, http.StatusOK, users)
}
