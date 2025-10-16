package authn

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/adrianpk/hatmax-ref/pkg/lib/core"
	"github.com/adrianpk/hatmax-ref/services/authn/internal/config"
)

const UserMaxBodyBytes = 1 << 20

// NewUserHandler creates a new UserHandler for the User aggregate.
func NewUserHandler(repo UserRepo, xparams config.XParams) *UserHandler {
	return &UserHandler{
		repo:    repo,
		xparams: xparams,
	}
}

type UserHandler struct {
	repo    UserRepo
	xparams config.XParams
}

func (h *UserHandler) RegisterRoutes(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/", h.CreateUser)
		r.Get("/", h.GetAllUsers)
		r.Get("/{id}", h.GetUser)
		r.Put("/{id}", h.UpdateUser)
		r.Delete("/{id}", h.DeleteUser)
	})
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	req, ok := h.decodeUserCreatePayload(w, r)
	if !ok {
		return
	}

	validationErrors := ValidateCreateUserRequest(ctx, req)
	if len(validationErrors) > 0 {
		core.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	user := req.ToUser()
	user.EnsureID()
	user.BeforeCreate()

	if err := h.repo.Create(ctx, &user); err != nil {
		log.Error("cannot create user", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Could not create user")
		return
	}

	// Standard links
	links := core.RESTfulLinksFor(&user)

	w.WriteHeader(http.StatusCreated)
	core.RespondSuccess(w, user, links...)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r)
	if !ok {
		return
	}

	user, err := h.repo.Get(ctx, id)
	if err != nil {
		log.Error("error loading user", "error", err, "id", id.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not retrieve user")
		return
	}

	if user == nil {
		core.RespondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Standard links
	links := core.RESTfulLinksFor(user)

	core.RespondSuccess(w, user, links...)
}

func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	users, err := h.repo.List(ctx)
	if err != nil {
		log.Error("error retrieving users", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Could not list all users")
		return
	}

	// Collection response
	core.RespondCollection(w, users, "user")
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r)
	if !ok {
		return
	}

	req, ok := h.decodeUserUpdatePayload(w, r)
	if !ok {
		return
	}

	validationErrors := ValidateUpdateUserRequest(ctx, id, req)
	if len(validationErrors) > 0 {
		core.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	user := req.ToUser()
	user.SetID(id)
	user.BeforeUpdate()

	if err := h.repo.Save(ctx, &user); err != nil {
		log.Error("cannot save user", "error", err, "id", id.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not update user")
		return
	}

	// Standard links
	links := core.RESTfulLinksFor(&user)

	core.RespondSuccess(w, user, links...)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		log.Error("error deleting user", "error", err, "id", id.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods following same patterns as ListHandler

func (h *UserHandler) logForRequest(r *http.Request) core.Logger {
	return h.xparams.Log.With(
		"request_id", middleware.GetReqID(r.Context()),
		"method", r.Method,
		"path", r.URL.Path,
	)
}

func (h *UserHandler) parseIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "id")
	if strings.TrimSpace(idStr) == "" {
		core.RespondError(w, http.StatusBadRequest, "Missing or invalid id")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		core.RespondError(w, http.StatusBadRequest, "Invalid id format")
		return uuid.Nil, false
	}

	return id, true
}

func (h *UserHandler) decodeUserCreatePayload(w http.ResponseWriter, r *http.Request) (UserCreateRequest, bool) {
	var req UserCreateRequest

	r.Body = http.MaxBytesReader(w, r.Body, UserMaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		core.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return req, false
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		core.RespondError(w, http.StatusBadRequest, "Request body is empty")
		return req, false
	}

	if err := json.Unmarshal(body, &req); err != nil {
		core.RespondError(w, http.StatusBadRequest, "Could not parse JSON")
		return req, false
	}

	return req, true
}

func (h *UserHandler) decodeUserUpdatePayload(w http.ResponseWriter, r *http.Request) (UserUpdateRequest, bool) {
	var req UserUpdateRequest

	r.Body = http.MaxBytesReader(w, r.Body, UserMaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		core.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return req, false
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		core.RespondError(w, http.StatusBadRequest, "Request body is empty")
		return req, false
	}

	if err := json.Unmarshal(body, &req); err != nil {
		core.RespondError(w, http.StatusBadRequest, "Could not parse JSON")
		return req, false
	}

	return req, true
}
