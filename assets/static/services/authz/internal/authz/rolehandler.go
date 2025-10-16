package authz

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/username/repo/pkg/lib/core"
	"github.com/username/repo/services/authz/internal/config"
)

// RoleHandler handles role-related HTTP requests
type RoleHandler struct {
	roleRepo RoleRepo
	xparams  config.XParams
}

// NewRoleHandler creates a new RoleHandler
func NewRoleHandler(roleRepo RoleRepo, xparams config.XParams) *RoleHandler {
	return &RoleHandler{
		roleRepo: roleRepo,
		xparams:  xparams,
	}
}

// RegisterRoutes registers role routes
func (h *RoleHandler) RegisterRoutes(r chi.Router) {
	r.Route("/authz/roles", func(r chi.Router) {
		r.Get("/", h.ListRoles)
		r.Post("/", h.CreateRole)
		r.Get("/{id}", h.GetRole)
		r.Put("/{id}", h.UpdateRole)
		r.Delete("/{id}", h.DeleteRole)
	})
}

// RoleRequest represents the request payload for creating/updating roles
type RoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// ListRoles handles GET /authz/roles
func (h *RoleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	// Parse query parameters
	status := r.URL.Query().Get("status")
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	var roles []*Role
	var err error

	if status != "" {
		roles, err = h.roleRepo.ListByStatus(ctx, status)
	} else {
		roles, err = h.roleRepo.List(ctx)
	}

	if err != nil {
		log.Error("failed to list roles", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Failed to retrieve roles")
		return
	}

	// Apply pagination if specified
	if page != "" && limit != "" {
		pageNum, _ := strconv.Atoi(page)
		limitNum, _ := strconv.Atoi(limit)
		roles = h.paginateRoles(roles, pageNum, limitNum)
	}

	// Generate HATEOAS links
	links := []core.Link{
		{Rel: "self", Href: "/authz/roles"},
		{Rel: "create", Href: "/authz/roles"},
	}

	response := core.SuccessResponse{
		Data:  roles,
		Links: links,
	}

	core.RespondSuccess(w, response.Data)
}

// CreateRole handles POST /authz/roles
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	var req RoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Debug("invalid request payload", "error", err)
		core.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate request
	if req.Name == "" {
		core.RespondError(w, http.StatusBadRequest, "Role name is required")
		return
	}

	// Check if role already exists
	existing, err := h.roleRepo.GetByName(ctx, req.Name)
	if err != nil {
		log.Error("error checking existing role", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Failed to create role")
		return
	}
	if existing != nil {
		core.RespondError(w, http.StatusConflict, "Role already exists")
		return
	}

	// Create new role
	role := NewRole()
	role.Name = req.Name
	role.Permissions = req.Permissions

	if err := h.roleRepo.Create(ctx, role); err != nil {
		log.Error("failed to create role", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Failed to create role")
		return
	}

	w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusCreated); json.NewEncoder(w).Encode(core.SuccessResponse{Data: role})
}

// GetRole handles GET /authz/roles/{id}
func (h *RoleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		core.RespondError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	role, err := h.roleRepo.Get(ctx, id)
	if err != nil {
		log.Error("failed to get role", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Failed to retrieve role")
		return
	}

	if role == nil {
		core.RespondError(w, http.StatusNotFound, "Role not found")
		return
	}

	w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusOK); json.NewEncoder(w).Encode(core.SuccessResponse{Data: role})
}

// UpdateRole handles PUT /authz/roles/{id}
func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		core.RespondError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	var req RoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get existing role
	role, err := h.roleRepo.Get(ctx, id)
	if err != nil {
		log.Error("failed to get role", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Failed to update role")
		return
	}

	if role == nil {
		core.RespondError(w, http.StatusNotFound, "Role not found")
		return
	}

	// Update role fields
	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Permissions != nil {
		role.Permissions = req.Permissions
	}

	if err := h.roleRepo.Save(ctx, role); err != nil {
		log.Error("failed to save role", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Failed to update role")
		return
	}

	w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusOK); json.NewEncoder(w).Encode(core.SuccessResponse{Data: role})
}

// DeleteRole handles DELETE /authz/roles/{id}
func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		core.RespondError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	if err := h.roleRepo.Delete(ctx, id); err != nil {
		log.Error("failed to delete role", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Failed to delete role")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *RoleHandler) logForRequest(r *http.Request) core.Logger {
	return h.xparams.Log.With(
		"request_id", middleware.GetReqID(r.Context()),
		"method", r.Method,
		"path", r.URL.Path,
	)
}

func (h *RoleHandler) paginateRoles(roles []*Role, page, limit int) []*Role {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	start := (page - 1) * limit
	end := start + limit

	if start >= len(roles) {
		return []*Role{}
	}
	if end > len(roles) {
		end = len(roles)
	}

	return roles[start:end]
}