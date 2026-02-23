// Package web provides the unified HTTP handler for ticked.
package web

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hatmaxkit/hatmax/auth"
	"github.com/hatmaxkit/hatmax/examples/ticked/internal/feat/audit"
	"github.com/hatmaxkit/hatmax/examples/ticked/internal/feat/list"
	"github.com/hatmaxkit/hatmax/htmx"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/middleware"
	"github.com/hatmaxkit/hatmax/web"
)

// authService defines the auth operations needed by the handler.
type authService interface {
	Signup(ctx context.Context, email, password string) (*auth.User, error)
	Signin(ctx context.Context, email, password string) (*auth.Session, error)
	Signout(ctx context.Context, sessionToken string) error
	ValidateSession(ctx context.Context, token string) (*auth.User, error)
}

// authQueries defines the auth query operations needed by the handler.
type authQueries interface {
	CountUsers(ctx context.Context) (int64, error)
	ListUsers(ctx context.Context) ([]*auth.User, error)
	GetUserByID(ctx context.Context, id string) (*auth.User, error)
	UpdateUserRoles(ctx context.Context, id string, roles []string, updatedAt time.Time) error
	UpdateUserActive(ctx context.Context, id string, active bool, updatedAt time.Time) error
}

// listService defines the list operations needed by the handler.
type listService interface {
	GetOrCreateList(ctx context.Context, userID string) (*list.TodoList, error)
	AddItem(ctx context.Context, userID, text string) (*list.TodoItem, error)
	ToggleItem(ctx context.Context, userID, itemID string) (*list.TodoItem, error)
	RemoveItem(ctx context.Context, userID, itemID string) error
}

// Handler handles all HTTP requests for ticked.
type Handler struct {
	assetsFS   embed.FS
	authSvc    authService
	authQ      authQueries
	listSvc    listService
	auditStore audit.Store
	tmpl       *web.TemplateManager
	log        log.Logger
}

// NewHandler creates a new web handler.
func NewHandler(
	assetsFS embed.FS,
	authSvc authService,
	authQ authQueries,
	listSvc listService,
	auditStore audit.Store,
	tmpl *web.TemplateManager,
	log log.Logger,
) *Handler {
	return &Handler{
		assetsFS:   assetsFS,
		authSvc:    authSvc,
		authQ:      authQ,
		listSvc:    listSvc,
		auditStore: auditStore,
		tmpl:       tmpl,
		log:        log,
	}
}

// RegisterRoutes registers all HTTP routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	staticFS, err := fs.Sub(h.assetsFS, "assets/static")
	if err != nil {
		h.log.Errorf("cannot create static sub-fs: %v", err)
	} else {
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	r.Get("/", h.handleIndex)
	r.Get("/signin", h.handleSigninForm)
	r.Post("/signin", h.handleSignin)
	r.Get("/signup", h.handleSignupForm)
	r.Post("/signup", h.handleSignup)

	// User routes (authenticated)
	r.Group(func(r chi.Router) {
		r.Use(h.requireAuth)
		r.Post("/signout", h.handleSignout)
		r.Get("/list-items", h.handleListItems)
		r.Post("/add-item", h.handleAddItem)
		r.Post("/toggle-item", h.handleToggleItem)
		r.Post("/delete-item", h.handleDeleteItem)
	})

	// Admin routes (authenticated + admin/superadmin role)
	r.Group(func(r chi.Router) {
		r.Use(h.requireRoles("admin", "superadmin"))
		r.Get("/admin", h.handleDashboard)
		r.Get("/admin/list-users", h.handleListUsers)
		r.Get("/admin/get-user", h.handleGetUser)
		r.Post("/admin/update-roles", h.handleUpdateRoles)
		r.Post("/admin/toggle-user", h.handleToggleUser)
		r.Get("/admin/list-events", h.handleListEvents)
	})
}

// requireAuth is a middleware that validates authentication.
func (h *Handler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.SessionCookieName)
		if err != nil {
			http.Redirect(w, r, "/signin", http.StatusSeeOther)

			return
		}

		user, err := h.authSvc.ValidateSession(r.Context(), cookie.Value)
		if err != nil {
			auth.ClearSessionCookie(w)
			http.Redirect(w, r, "/signin", http.StatusSeeOther)

			return
		}

		ctx := auth.WithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireRoles returns a middleware that validates authentication and roles.
func (h *Handler) requireRoles(roles ...string) func(http.Handler) http.Handler {
	return middleware.RequireRoles(h.authSvc, roles...)
}

// --- Auth handlers ---

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}

func (h *Handler) handleSigninForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Sign In - Ticked",
	}
	h.tmpl.Render(w, "auth", "signin", data)
}

func (h *Handler) handleSignin(w http.ResponseWriter, r *http.Request) {
	form, err := web.ParseForm(r)
	if err != nil {
		h.renderSigninError(w, "Invalid form data")

		return
	}

	email := form.String("email")
	password := form.String("password")

	if email == "" || password == "" {
		h.renderSigninError(w, "Email and password are required")

		return
	}

	session, err := h.authSvc.Signin(r.Context(), email, password)
	if err != nil {
		h.log.Errorf("signin failed for %s: %v", email, err)
		h.renderSigninError(w, "Invalid email or password")

		return
	}

	maxAge := int(time.Until(session.ExpiresAt).Seconds())
	auth.SetSessionCookie(w, session.Token, maxAge)

	w.Header().Set("HX-Redirect", "/list-items")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleSignupForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Sign Up - Ticked",
	}
	h.tmpl.Render(w, "auth", "signup", data)
}

func (h *Handler) handleSignup(w http.ResponseWriter, r *http.Request) {
	form, err := web.ParseForm(r)
	if err != nil {
		h.renderSignupError(w, "Invalid form data")

		return
	}

	email := form.String("email")
	password := form.String("password")
	confirmPassword := form.String("confirm_password")

	if email == "" || password == "" {
		h.renderSignupError(w, "Email and password are required")

		return
	}

	if password != confirmPassword {
		h.renderSignupError(w, "Passwords do not match")

		return
	}

	_, err = h.authSvc.Signup(r.Context(), email, password)
	if err != nil {
		h.log.Errorf("signup failed for %s: %v", email, err)
		h.renderSignupError(w, err.Error())

		return
	}

	session, err := h.authSvc.Signin(r.Context(), email, password)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)

		return
	}

	maxAge := int(time.Until(session.ExpiresAt).Seconds())
	auth.SetSessionCookie(w, session.Token, maxAge)

	w.Header().Set("HX-Redirect", "/list-items")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleSignout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		h.authSvc.Signout(r.Context(), cookie.Value)
	}

	auth.ClearSessionCookie(w)

	w.Header().Set("HX-Redirect", "/signin")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) renderSigninError(w http.ResponseWriter, errorMsg string) {
	data := map[string]interface{}{
		"Title": "Sign In - Ticked",
		"Error": errorMsg,
	}
	h.tmpl.Render(w, "auth", "signin", data)
}

func (h *Handler) renderSignupError(w http.ResponseWriter, errorMsg string) {
	data := map[string]interface{}{
		"Title": "Sign Up - Ticked",
		"Error": errorMsg,
	}
	h.tmpl.Render(w, "auth", "signup", data)
}

// --- List handlers ---

func (h *Handler) handleListItems(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUser(r.Context())
	if !ok {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)

		return
	}

	items, err := h.listSvc.GetOrCreateList(r.Context(), user.ID)
	if err != nil {
		h.log.Errorf("get list failed: %v", err)
		http.Error(w, "Failed to load list", http.StatusInternalServerError)

		return
	}

	data := map[string]interface{}{
		"Title":     "My List - Ticked",
		"UserEmail": user.Email,
		"Items":     items.Items,
	}
	h.tmpl.Render(w, "todos", "list", data)
}

func (h *Handler) handleAddItem(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUser(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)

		return
	}

	form, err := web.ParseForm(r)
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)

		return
	}

	text := form.String("text")
	if text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)

		return
	}

	item, err := h.listSvc.AddItem(r.Context(), user.ID, text)
	if err != nil {
		h.log.Errorf("add item failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	h.tmpl.RenderPartial(w, "todos", "item", item)
}

func (h *Handler) handleToggleItem(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUser(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)

		return
	}

	form, err := web.ParseForm(r)
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)

		return
	}

	itemID := form.String("id")
	if itemID == "" {
		http.Error(w, "Item ID required", http.StatusBadRequest)

		return
	}

	item, err := h.listSvc.ToggleItem(r.Context(), user.ID, itemID)
	if err != nil {
		h.log.Errorf("toggle item failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	h.tmpl.RenderPartial(w, "todos", "item", item)
}

func (h *Handler) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUser(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)

		return
	}

	form, err := web.ParseForm(r)
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)

		return
	}

	itemID := form.String("id")
	if itemID == "" {
		http.Error(w, "Item ID required", http.StatusBadRequest)

		return
	}

	err = h.listSvc.RemoveItem(r.Context(), user.ID, itemID)
	if err != nil {
		h.log.Errorf("error deleting item: %v", err)
		http.Error(w, "Cannot delete", http.StatusInternalServerError)

		return
	}

	htmx.RespondDelete(w)
}

// --- Admin handlers ---

func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := auth.GetUser(r.Context())

	count, err := h.authQ.CountUsers(r.Context())
	if err != nil {
		h.log.Errorf("count users failed: %v", err)

		count = 0
	}

	data := map[string]interface{}{
		"Title":      "Admin Dashboard - Ticked",
		"UserEmail":  currentUser.Email,
		"TotalUsers": count,
	}
	h.tmpl.Render(w, "admin", "dashboard", data)
}

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := auth.GetUser(r.Context())

	users, err := h.authQ.ListUsers(r.Context())
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

func (h *Handler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := auth.GetUser(r.Context())

	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)

		return
	}

	user, err := h.authQ.GetUserByID(r.Context(), userID)
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
	form, err := web.ParseForm(r)
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)

		return
	}

	userID := form.String("id")
	if userID == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)

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

	err = h.authQ.UpdateUserRoles(r.Context(), userID, roles, time.Now())
	if err != nil {
		h.log.Errorf("update roles for %s failed: %v", userID, err)
		http.Error(w, "Failed to update roles", http.StatusInternalServerError)

		return
	}

	h.log.Infof("updated roles for user %s: %v", userID, roles)

	w.Header().Set("HX-Redirect", "/admin/get-user?id="+userID)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleToggleUser(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := auth.GetUser(r.Context())

	form, err := web.ParseForm(r)
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)

		return
	}

	userID := form.String("id")
	if userID == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)

		return
	}

	if userID == currentUser.ID {
		http.Error(w, "Cannot deactivate yourself", http.StatusBadRequest)

		return
	}

	user, err := h.authQ.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)

		return
	}

	newActive := !user.Active

	err = h.authQ.UpdateUserActive(r.Context(), userID, newActive, time.Now())
	if err != nil {
		h.log.Errorf("toggle active for %s failed: %v", userID, err)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)

		return
	}

	h.log.Infof("toggled active for user %s: %v", userID, newActive)

	w.Header().Set("HX-Redirect", "/admin/get-user?id="+userID)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleListEvents(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := auth.GetUser(r.Context())

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
