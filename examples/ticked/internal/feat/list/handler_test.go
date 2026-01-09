package list

import (
	"context"
	"embed"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hatmaxkit/hatmax/auth"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/web"
)

//go:embed testdata/assets/templates/todos/list.html testdata/assets/templates/todos/item.html
var testAssetsFS embed.FS

func setupTestHandler(t *testing.T) *Handler {
	t.Helper()

	store := newMockStore()
	svc := NewService(store, nil, log.NewNoopLogger())
	tmpl := web.NewTemplateManager(testAssetsFS, log.NewNoopLogger())

	if err := tmpl.Start(context.Background()); err != nil {
		t.Fatalf("cannot start template manager: %v", err)
	}

	return NewHandler(svc, tmpl, log.NewNoopLogger())
}

func TestNewHandler(t *testing.T) {
	store := newMockStore()
	svc := NewService(store, nil, log.NewNoopLogger())
	h := NewHandler(svc, nil, log.NewNoopLogger())

	if h == nil {
		t.Error("NewHandler should return non-nil handler")
	}
}

func TestHandlerRegisterRoutes(t *testing.T) {
	store := newMockStore()
	svc := NewService(store, nil, log.NewNoopLogger())
	h := NewHandler(svc, nil, log.NewNoopLogger())

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	routes := []string{}
	chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		routes = append(routes, method+" "+route)
		return nil
	})

	expectedRoutes := []string{
		"GET /list",
		"POST /list/items",
		"POST /list/items/{itemID}/toggle",
		"DELETE /list/items/{itemID}",
	}

	for _, exp := range expectedRoutes {
		found := false
		for _, r := range routes {
			if r == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected route %s not found", exp)
		}
	}
}

func TestHandleListNoUser(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/list", nil)
	w := httptest.NewRecorder()

	h.handleList(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect status %d, got %d", http.StatusSeeOther, w.Code)
	}
}

func TestHandleListWithUser(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/list", nil)
	ctx := auth.WithUser(req.Context(), &auth.User{ID: "user-1", Email: "test@example.com"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Todo List") {
		t.Error("response should contain 'Todo List'")
	}
}

func TestHandleAddItemNoUser(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/list/items", nil)
	w := httptest.NewRecorder()

	h.handleAddItem(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleAddItemEmptyText(t *testing.T) {
	h := setupTestHandler(t)

	form := url.Values{}
	form.Set("text", "")
	req := httptest.NewRequest("POST", "/list/items", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := auth.WithUser(req.Context(), &auth.User{ID: "user-1", Email: "test@example.com"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleAddItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAddItemSuccess(t *testing.T) {
	h := setupTestHandler(t)

	form := url.Values{}
	form.Set("text", "Buy milk")
	req := httptest.NewRequest("POST", "/list/items", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := auth.WithUser(req.Context(), &auth.User{ID: "user-1", Email: "test@example.com"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleAddItem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Buy milk") {
		t.Error("response should contain item text")
	}
}

func TestHandleToggleItemNoUser(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/list/items/123/toggle", nil)
	w := httptest.NewRecorder()

	h.handleToggleItem(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleToggleItemNoItemID(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/list/items//toggle", nil)
	ctx := auth.WithUser(req.Context(), &auth.User{ID: "user-1", Email: "test@example.com"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleToggleItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleRemoveItemNoUser(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest("DELETE", "/list/items/123", nil)
	w := httptest.NewRecorder()

	h.handleRemoveItem(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleRemoveItemNoItemID(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest("DELETE", "/list/items/", nil)
	ctx := auth.WithUser(req.Context(), &auth.User{ID: "user-1", Email: "test@example.com"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleRemoveItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleToggleItemSuccess(t *testing.T) {
	store := newMockStore()
	svc := NewService(store, nil, log.NewNoopLogger())
	tmpl := web.NewTemplateManager(testAssetsFS, log.NewNoopLogger())
	tmpl.Start(context.Background())
	h := NewHandler(svc, tmpl, log.NewNoopLogger())

	ctx := context.Background()
	item, _ := svc.AddItem(ctx, "user-1", "Test item")

	req := httptest.NewRequest("POST", "/list/items/"+item.ItemID+"/toggle", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("itemID", item.ItemID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "user-1", Email: "test@example.com"}))
	w := httptest.NewRecorder()

	h.handleToggleItem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHandleToggleItemNotFound(t *testing.T) {
	store := newMockStore()
	svc := NewService(store, nil, log.NewNoopLogger())
	tmpl := web.NewTemplateManager(testAssetsFS, log.NewNoopLogger())
	tmpl.Start(context.Background())
	h := NewHandler(svc, tmpl, log.NewNoopLogger())

	ctx := context.Background()
	svc.AddItem(ctx, "user-1", "Test item")

	req := httptest.NewRequest("POST", "/list/items/nonexistent/toggle", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("itemID", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "user-1", Email: "test@example.com"}))
	w := httptest.NewRecorder()

	h.handleToggleItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleRemoveItemSuccess(t *testing.T) {
	store := newMockStore()
	svc := NewService(store, nil, log.NewNoopLogger())
	tmpl := web.NewTemplateManager(testAssetsFS, log.NewNoopLogger())
	tmpl.Start(context.Background())
	h := NewHandler(svc, tmpl, log.NewNoopLogger())

	ctx := context.Background()
	item, _ := svc.AddItem(ctx, "user-1", "Test item")

	req := httptest.NewRequest("DELETE", "/list/items/"+item.ItemID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("itemID", item.ItemID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "user-1", Email: "test@example.com"}))
	w := httptest.NewRecorder()

	h.handleRemoveItem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHandleRemoveItemNotFound(t *testing.T) {
	store := newMockStore()
	svc := NewService(store, nil, log.NewNoopLogger())
	tmpl := web.NewTemplateManager(testAssetsFS, log.NewNoopLogger())
	tmpl.Start(context.Background())
	h := NewHandler(svc, tmpl, log.NewNoopLogger())

	ctx := context.Background()
	svc.AddItem(ctx, "user-1", "Test item")

	req := httptest.NewRequest("DELETE", "/list/items/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("itemID", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "user-1", Email: "test@example.com"}))
	w := httptest.NewRecorder()

	h.handleRemoveItem(w, req)
}
