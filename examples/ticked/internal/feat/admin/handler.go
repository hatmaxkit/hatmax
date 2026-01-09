package admin

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	hatmaxAuth "github.com/hatmaxkit/hatmax/auth"
	"github.com/hatmaxkit/hatmax/examples/ticked/internal/feat/audit"
	tickedAuth "github.com/hatmaxkit/hatmax/examples/ticked/internal/feat/auth"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/web"
)

// Handler handles admin HTTP requests.
type Handler struct {
	queries    *tickedAuth.Queries
	auditStore audit.Store
	tmpl       *web.TemplateManager
	log        log.Logger
}

// NewHandler creates a new admin handler.
func NewHandler(queries *tickedAuth.Queries, auditStore audit.Store, tmpl *web.TemplateManager, log log.Logger) *Handler {
	return &Handler{
		queries:    queries,
		auditStore: auditStore,
		tmpl:       tmpl,
		log:        log,
	}
}

// RegisterRoutes registers admin routes on the router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/admin", h.handleDashboard)
	r.Get("/admin/users", h.handleListUsers)
	r.Get("/admin/users/{userID}", h.handleUserDetail)
	r.Post("/admin/users/{userID}/roles", h.handleUpdateRoles)
	r.Post("/admin/users/{userID}/toggle", h.handleToggleActive)
	r.Get("/admin/events", h.handleListEvents)
}

func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := hatmaxAuth.GetUser(r.Context())

	count, err := h.queries.CountUsers(r.Context())
	if err != nil {
		h.log.Errorf("count users failed: %v", err)
		count = 0
	}

	data := map[string]interface{}{
		"Title":       "Admin Dashboard - Ticked",
		"UserEmail":   currentUser.Email,
		"TotalUsers":  count,
	}
	h.tmpl.Render(w, "admin", "dashboard", data)
}

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := hatmaxAuth.GetUser(r.Context())

	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		h.log.Errorf("list users failed: %v", err)
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "Users - Ticked Admin",
		"UserEmail": currentUser.Email,
		"Users":     users,
	}
	h.tmpl.Render(w, "admin", "users", data)
}

func (h *Handler) handleUserDetail(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := hatmaxAuth.GetUser(r.Context())
	userID := chi.URLParam(r, "userID")

	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		h.log.Errorf("get user %s failed: %v", userID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Title":       "User Details - Ticked Admin",
		"UserEmail":   currentUser.Email,
		"User":        user,
		"RolesString": strings.Join(user.Roles, ", "),
	}
	h.tmpl.Render(w, "admin", "user", data)
}

func (h *Handler) handleUpdateRoles(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")

	form, err := web.ParseForm(r)
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	rolesStr := form.String("roles")
	var roles []string
	if rolesStr != "" {
		for _, role := range strings.Split(rolesStr, ",") {
			role = strings.TrimSpace(role)
			if role != "" {
				roles = append(roles, role)
			}
		}
	}

	if err := h.queries.UpdateUserRoles(r.Context(), userID, roles, time.Now()); err != nil {
		h.log.Errorf("update roles for %s failed: %v", userID, err)
		http.Error(w, "Failed to update roles", http.StatusInternalServerError)
		return
	}

	h.log.Infof("updated roles for user %s: %v", userID, roles)

	// HTMX redirect back to user detail
	w.Header().Set("HX-Redirect", "/admin/users/"+userID)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleToggleActive(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := hatmaxAuth.GetUser(r.Context())
	userID := chi.URLParam(r, "userID")

	// Prevent self-deactivation
	if userID == currentUser.ID {
		http.Error(w, "Cannot deactivate yourself", http.StatusBadRequest)
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	newActive := !user.Active
	if err := h.queries.UpdateUserActive(r.Context(), userID, newActive, time.Now()); err != nil {
		h.log.Errorf("toggle active for %s failed: %v", userID, err)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	h.log.Infof("toggled active for user %s: %v", userID, newActive)

	// HTMX redirect back to user detail
	w.Header().Set("HX-Redirect", "/admin/users/"+userID)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleListEvents(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := hatmaxAuth.GetUser(r.Context())

	events, err := h.auditStore.List(r.Context(), 100)
	if err != nil {
		h.log.Errorf("list events failed: %v", err)
		http.Error(w, "Failed to load events", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "Audit Events - Ticked Admin",
		"UserEmail": currentUser.Email,
		"Events":    events,
	}
	h.tmpl.Render(w, "admin", "events", data)
}
