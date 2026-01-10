package list

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hatmaxkit/hatmax/auth"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/web"
	"github.com/hatmaxkit/hatmax/web/htmx"
)

// Handler handles todo list HTTP requests.
type Handler struct {
	svc  *Service
	tmpl *web.TemplateManager
	log  log.Logger
}

// NewHandler creates a new list handler.
func NewHandler(svc *Service, tmpl *web.TemplateManager, log log.Logger) *Handler {
	return &Handler{
		svc:  svc,
		tmpl: tmpl,
		log:  log,
	}
}

// RegisterRoutes registers list routes on the router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/list-items", h.handleListItems)
	r.Post("/add-item", h.handleAddItem)
	r.Post("/toggle-item", h.handleToggleItem)
	r.Post("/delete-item", h.handleDeleteItem)
}

func (h *Handler) handleListItems(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUser(r.Context())
	if !ok {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	list, err := h.svc.GetOrCreateList(r.Context(), user.ID)
	if err != nil {
		h.log.Errorf("get list failed: %v", err)
		http.Error(w, "Failed to load list", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":     "My List - Ticked",
		"UserEmail": user.Email,
		"Items":     list.Items,
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

	item, err := h.svc.AddItem(r.Context(), user.ID, text)
	if err != nil {
		h.log.Errorf("add item failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return the new item partial for HTMX
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

	item, err := h.svc.ToggleItem(r.Context(), user.ID, itemID)
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

	err = h.svc.RemoveItem(r.Context(), user.ID, itemID)
	htmx.RespondDelete(w, err, h.log)
}
