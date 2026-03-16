package web

import (
	"context"
	"embed"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/examples/ticked/internal/feat/audit"
	"hatmax.adrianpk.com/examples/ticked/internal/feat/list"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/ui"
	"hatmax.adrianpk.com/web"
)

//go:embed testdata/assets
var testAssetsFS embed.FS

// --- Fakes ---

type fakeAuditStore struct {
	records []audit.Record
	listErr error
}

func (f *fakeAuditStore) Save(ctx context.Context, record *audit.Record) error {
	return nil
}

func (f *fakeAuditStore) List(ctx context.Context, limit int) ([]audit.Record, error) {
	return f.records, f.listErr
}

// --- Tests ---

func TestHandlerRegisterRoutes(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))

	h := &Handler{
		tmpl: tmplMgr,
		log:  logger,
	}

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/"},
		{"GET", "/signin"},
		{"POST", "/signin"},
		{"GET", "/signup"},
		{"POST", "/signup"},
		{"POST", "/signout"},
		{"GET", "/list-items"},
		{"POST", "/add-item"},
		{"POST", "/toggle-item"},
		{"POST", "/delete-item"},
		{"GET", "/admin"},
		{"GET", "/admin/list-users"},
		{"GET", "/admin/get-user"},
		{"POST", "/admin/update-roles"},
		{"POST", "/admin/toggle-user"},
		{"GET", "/admin/list-events"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, nil)

			match := chi.NewRouteContext()
			if !r.Match(match, req.Method, req.URL.Path) {
				t.Errorf("route %s %s not registered", rt.method, rt.path)
			}
		})
	}
}

func TestHandleIndex(t *testing.T) {
	logger := log.NewTestLogger("error")
	h := &Handler{log: logger}

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	h.handleIndex(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}

	if loc := rec.Header().Get("Location"); loc != "/signin" {
		t.Errorf("expected redirect to /signin, got %s", loc)
	}
}

func TestHandleSigninForm(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	tmplMgr.Start(context.Background())

	h := &Handler{
		tmpl: tmplMgr,
		log:  logger,
	}

	req := httptest.NewRequest("GET", "/signin", nil)
	rec := httptest.NewRecorder()

	h.handleSigninForm(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleSignupForm(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	tmplMgr.Start(context.Background())

	h := &Handler{
		tmpl: tmplMgr,
		log:  logger,
	}

	req := httptest.NewRequest("GET", "/signup", nil)
	rec := httptest.NewRecorder()

	h.handleSignupForm(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleSignin_EmptyFields(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	tmplMgr.Start(context.Background())

	h := &Handler{
		tmpl: tmplMgr,
		log:  logger,
	}

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"empty email", "", "password"},
		{"empty password", "test@example.com", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("email", tt.email)
			form.Set("password", tt.password)

			req := httptest.NewRequest("POST", "/signin", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h.handleSignin(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
		})
	}
}

func TestHandleSignup_EmptyFields(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	tmplMgr.Start(context.Background())

	h := &Handler{
		tmpl: tmplMgr,
		log:  logger,
	}

	tests := []struct {
		name            string
		email           string
		password        string
		confirmPassword string
	}{
		{"empty email", "", "password", "password"},
		{"empty password", "test@example.com", "", ""},
		{"passwords dont match", "test@example.com", "password", "different"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("email", tt.email)
			form.Set("password", tt.password)
			form.Set("confirm_password", tt.confirmPassword)

			req := httptest.NewRequest("POST", "/signup", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h.handleSignup(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
		})
	}
}

func TestHandleSignout(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		authSvc: &fakeAuthSvc{},
		log:     logger,
	}

	req := httptest.NewRequest("POST", "/signout", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token123"})

	rec := httptest.NewRecorder()

	h.handleSignout(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if hx := rec.Header().Get("HX-Redirect"); hx != "/signin" {
		t.Errorf("expected HX-Redirect /signin, got %s", hx)
	}
}

func TestHandleListItems_NoUser(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		log: logger,
	}

	req := httptest.NewRequest("GET", "/list-items", nil)
	rec := httptest.NewRecorder()

	h.handleListItems(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
}

func TestHandleListItems_WithUser(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	tmplMgr.Start(context.Background())

	h := &Handler{
		listSvc: &fakeListSvc{
			list: &list.TodoList{
				ListID: "list1",
				UserID: "user1",
				Items:  []list.TodoItem{},
			},
		},
		tmpl: tmplMgr,
		log:  logger,
	}

	user := &auth.User{ID: "user1", Email: "test@example.com"}
	ctx := auth.WithUser(context.Background(), user)

	req := httptest.NewRequest("GET", "/list-items", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	h.handleListItems(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleAddItem_NoUser(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		log: logger,
	}

	form := url.Values{}
	form.Set("text", "test")

	req := httptest.NewRequest("POST", "/add-item", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	h.handleAddItem(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandleAddItem_EmptyText(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		log: logger,
	}

	user := &auth.User{ID: "user1", Email: "test@example.com"}
	ctx := auth.WithUser(context.Background(), user)

	form := url.Values{}
	form.Set("text", "")

	req := httptest.NewRequest("POST", "/add-item", strings.NewReader(form.Encode())).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	h.handleAddItem(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleAddItem_Success(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	tmplMgr.Start(context.Background())

	h := &Handler{
		listSvc: &fakeListSvc{
			item: &list.TodoItem{ItemID: "item1", Text: "test item"},
		},
		tmpl: tmplMgr,
		log:  logger,
	}

	user := &auth.User{ID: "user1", Email: "test@example.com"}
	ctx := auth.WithUser(context.Background(), user)

	form := url.Values{}
	form.Set("text", "test item")

	req := httptest.NewRequest("POST", "/add-item", strings.NewReader(form.Encode())).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	h.handleAddItem(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleToggleItem_NoUser(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		log: logger,
	}

	form := url.Values{}
	form.Set("id", "item1")

	req := httptest.NewRequest("POST", "/toggle-item", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	h.handleToggleItem(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandleDeleteItem_NoUser(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		log: logger,
	}

	form := url.Values{}
	form.Set("id", "item1")

	req := httptest.NewRequest("POST", "/delete-item", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	h.handleDeleteItem(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandleDashboard(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	tmplMgr.Start(context.Background())

	h := &Handler{
		authQ: &fakeAuthQ{countUsers: 10},
		tmpl:  tmplMgr,
		log:   logger,
	}

	user := &auth.User{ID: "admin1", Email: "admin@example.com", Roles: []string{"admin"}}
	ctx := auth.WithUser(context.Background(), user)

	req := httptest.NewRequest("GET", "/admin", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	h.handleDashboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleListUsers(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	tmplMgr.Start(context.Background())

	h := &Handler{
		authQ: &fakeAuthQ{
			users: []*auth.User{
				{ID: "1", Email: "user1@example.com"},
			},
		},
		tmpl: tmplMgr,
		log:  logger,
	}

	user := &auth.User{ID: "admin1", Email: "admin@example.com", Roles: []string{"admin"}}
	ctx := auth.WithUser(context.Background(), user)

	req := httptest.NewRequest("GET", "/admin/list-users", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	h.handleListUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleGetUser_MissingID(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		log: logger,
	}

	user := &auth.User{ID: "admin1", Email: "admin@example.com", Roles: []string{"admin"}}
	ctx := auth.WithUser(context.Background(), user)

	req := httptest.NewRequest("GET", "/admin/get-user", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	h.handleGetUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleGetUser_Success(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	tmplMgr.Start(context.Background())

	h := &Handler{
		authQ: &fakeAuthQ{
			user: &auth.User{ID: "user1", Email: "user1@example.com", Roles: []string{"user"}},
		},
		tmpl: tmplMgr,
		log:  logger,
	}

	admin := &auth.User{ID: "admin1", Email: "admin@example.com", Roles: []string{"admin"}}
	ctx := auth.WithUser(context.Background(), admin)

	req := httptest.NewRequest("GET", "/admin/get-user?id=user1", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	h.handleGetUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleUpdateRoles_MissingID(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		log: logger,
	}

	form := url.Values{}
	form.Set("roles", "admin")

	req := httptest.NewRequest("POST", "/admin/update-roles", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	h.handleUpdateRoles(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleUpdateRoles_Success(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		authQ: &fakeAuthQ{},
		log:   logger,
	}

	form := url.Values{}
	form.Set("id", "user1")
	form.Set("roles", "admin, user")

	req := httptest.NewRequest("POST", "/admin/update-roles", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	h.handleUpdateRoles(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if hx := rec.Header().Get("HX-Redirect"); !strings.Contains(hx, "/admin/get-user") {
		t.Errorf("expected HX-Redirect to /admin/get-user, got %s", hx)
	}
}

func TestHandleToggleUser_MissingID(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		log: logger,
	}

	admin := &auth.User{ID: "admin1", Email: "admin@example.com", Roles: []string{"admin"}}
	ctx := auth.WithUser(context.Background(), admin)

	form := url.Values{}

	req := httptest.NewRequest("POST", "/admin/toggle-user", strings.NewReader(form.Encode())).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	h.handleToggleUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleToggleUser_SelfDeactivate(t *testing.T) {
	logger := log.NewTestLogger("error")

	h := &Handler{
		log: logger,
	}

	admin := &auth.User{ID: "admin1", Email: "admin@example.com", Roles: []string{"admin"}}
	ctx := auth.WithUser(context.Background(), admin)

	form := url.Values{}
	form.Set("id", "admin1") // trying to toggle self

	req := httptest.NewRequest("POST", "/admin/toggle-user", strings.NewReader(form.Encode())).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	h.handleToggleUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleListEvents(t *testing.T) {
	logger := log.NewTestLogger("error")
	tmplMgr := web.NewTemplateManager(testAssetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	tmplMgr.Start(context.Background())

	h := &Handler{
		auditStore: &fakeAuditStore{
			records: []audit.Record{
				{ID: "1", EventType: "test.event"},
			},
		},
		tmpl: tmplMgr,
		log:  logger,
	}

	user := &auth.User{ID: "admin1", Email: "admin@example.com", Roles: []string{"admin"}}
	ctx := auth.WithUser(context.Background(), user)

	req := httptest.NewRequest("GET", "/admin/list-events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	h.handleListEvents(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// --- Fake implementations that match handler's expected interfaces ---

type fakeListSvc struct {
	list      *list.TodoList
	listErr   error
	item      *list.TodoItem
	itemErr   error
	removeErr error
}

func (f *fakeListSvc) GetOrCreateList(ctx context.Context, userID string) (*list.TodoList, error) {
	return f.list, f.listErr
}

func (f *fakeListSvc) AddItem(ctx context.Context, userID, text string) (*list.TodoItem, error) {
	return f.item, f.itemErr
}

func (f *fakeListSvc) ToggleItem(ctx context.Context, userID, itemID string) (*list.TodoItem, error) {
	return f.item, f.itemErr
}

func (f *fakeListSvc) RemoveItem(ctx context.Context, userID, itemID string) error {
	return f.removeErr
}

type fakeAuthQ struct {
	countUsers int64
	countErr   error
	users      []*auth.User
	listErr    error
	user       *auth.User
	getErr     error
	updateErr  error
}

func (f *fakeAuthQ) CountUsers(ctx context.Context) (int64, error) {
	return f.countUsers, f.countErr
}

func (f *fakeAuthQ) ListUsers(ctx context.Context) ([]*auth.User, error) {
	return f.users, f.listErr
}

func (f *fakeAuthQ) GetUserByID(ctx context.Context, id string) (*auth.User, error) {
	return f.user, f.getErr
}

func (f *fakeAuthQ) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	return f.user, f.getErr
}

func (f *fakeAuthQ) UpdateUserRoles(ctx context.Context, id string, roles []string, updatedAt time.Time) error {
	return f.updateErr
}

func (f *fakeAuthQ) UpdateUserActive(ctx context.Context, id string, active bool, updatedAt time.Time) error {
	return f.updateErr
}

type fakeAuthSvc struct {
	user        *auth.User
	session     *auth.Session
	signupErr   error
	signinErr   error
	signoutErr  error
	validateErr error
}

func (f *fakeAuthSvc) Signup(ctx context.Context, email, password string) (*auth.User, error) {
	return f.user, f.signupErr
}

func (f *fakeAuthSvc) Signin(ctx context.Context, email, password string) (*auth.Session, error) {
	return f.session, f.signinErr
}

func (f *fakeAuthSvc) Signout(ctx context.Context, sessionToken string) error {
	return f.signoutErr
}

func (f *fakeAuthSvc) ValidateSession(ctx context.Context, token string) (*auth.User, error) {
	return f.user, f.validateErr
}
