package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/username/repo/pkg/lib/core"
	authpkg "github.com/username/repo/pkg/lib/auth"
	"github.com/username/repo/services/auth/internal/config"
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

	user, ok := h.decodeUserPayload(w, r, log)
	if !ok {
		return
	}

	user.EnsureID()
	user.BeforeCreate()

	validationErrors := ValidateCreateUser(ctx, user)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		core.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

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

	id, ok := h.parseIDParam(w, r, log)
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

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	user, ok := h.decodeUserPayload(w, r, log)
	if !ok {
		return
	}

	user.SetID(id)
	user.BeforeUpdate()

	validationErrors := ValidateUpdateUser(ctx, id, user)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors, "id", id.String())
		core.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

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

	id, ok := h.parseIDParam(w, r, log)
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

func (h *UserHandler) parseIDParam(w http.ResponseWriter, r *http.Request, log core.Logger) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "id")
	if strings.TrimSpace(idStr) == "" {
		log.Debug("missing id param")
		core.RespondError(w, http.StatusBadRequest, "Missing or invalid id")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Debug("invalid id param", "id", idStr, "error", err)
		core.RespondError(w, http.StatusBadRequest, "Invalid id format")
		return uuid.Nil, false
	}

	return id, true
}

func (h *UserHandler) decodeUserPayload(w http.ResponseWriter, r *http.Request, log core.Logger) (User, bool) {
	var user User

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, UserMaxBodyBytes)
	defer r.Body.Close()

	// Read and decode the JSON payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("cannot read request body", "error", err)
		core.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return user, false
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		log.Debug("empty request body")
		core.RespondError(w, http.StatusBadRequest, "Request body is empty")
		return user, false
	}

	if err := json.Unmarshal(body, &user); err != nil {
		log.Debug("cannot decode JSON", "error", err)
		core.RespondError(w, http.StatusBadRequest, "Could not parse JSON")
		return user, false
	}

	return user, true
}