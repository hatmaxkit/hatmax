package auth

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hatmaxkit/hatmax/auth"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/web"
)

// Handler handles authentication HTTP requests.
type Handler struct {
	svc     *auth.Service
	queries *Queries
	tmpl    *web.TemplateManager
	log     log.Logger
}

// NewHandler creates a new auth handler.
func NewHandler(svc *auth.Service, queries *Queries, tmpl *web.TemplateManager, log log.Logger) *Handler {
	return &Handler{
		svc:     svc,
		queries: queries,
		tmpl:    tmpl,
		log:     log,
	}
}

// RegisterRoutes registers auth routes on the router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.handleIndex)
	r.Get("/signin", h.handleSigninForm)
	r.Post("/signin", h.handleSignin)
	r.Get("/signup", h.handleSignupForm)
	r.Post("/signup", h.handleSignup)
	r.Post("/signout", h.handleSignout)
}

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

	session, err := h.svc.Signin(r.Context(), email, password)
	if err != nil {
		h.log.Errorf("signin failed for %s: %v", email, err)
		h.renderSigninError(w, "Invalid email or password")
		return
	}

	// Calculate max age from session expiry
	maxAge := int(time.Until(session.ExpiresAt).Seconds())
	auth.SetSessionCookie(w, session.Token, maxAge)

	// HTMX redirect
	w.Header().Set("HX-Redirect", "/list")
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

	// Check if this is the first user (make them superadmin)
	count, err := h.queries.CountUsers(r.Context())
	if err != nil {
		h.log.Errorf("count users failed: %v", err)
		h.renderSignupError(w, "An error occurred")
		return
	}

	_, err = h.svc.Signup(r.Context(), email, password)
	if err != nil {
		h.log.Errorf("signup failed for %s: %v", email, err)
		h.renderSignupError(w, err.Error())
		return
	}

	// If first user, make them superadmin (unique, highest authority)
	if count == 0 {
		user, err := h.queries.GetUserByEmail(r.Context(), email)
		if err == nil {
			h.queries.UpdateUserRoles(r.Context(), user.ID, []string{"superadmin"}, time.Now())
			h.log.Infof("First user %s promoted to superadmin", email)
		}
	}

	// Auto signin after signup
	session, err := h.svc.Signin(r.Context(), email, password)
	if err != nil {
		// Signup succeeded but signin failed - redirect to signin
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	maxAge := int(time.Until(session.ExpiresAt).Seconds())
	auth.SetSessionCookie(w, session.Token, maxAge)

	// HTMX redirect
	w.Header().Set("HX-Redirect", "/list")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleSignout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		// cookie.Value is the session token
		h.svc.Signout(r.Context(), cookie.Value)
	}

	auth.ClearSessionCookie(w)

	// HTMX redirect
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
