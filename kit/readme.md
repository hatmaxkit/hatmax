# kit

UI kit with dependency injection, emoji support, and HTMX-first components.

## Usage

```go
kit := kit.New(cfg, log,
    kit.WithSettings(settingsSvc),
    kit.WithOverlay(os.DirFS("./custom")),
    kit.WithCSRFFunc(csrf.Token),  // integrate with your CSRF middleware
)

// Template functions
tmpl := template.New("page").Funcs(kit.FuncMap())

// Or standalone (no kit instance)
tmpl := template.New("page").Funcs(kit.FuncMap())
```

## CSRF Integration

The kit integrates with external CSRF middleware via `WithCSRFFunc`:

```go
// Your CSRF package provides a function to extract token from context
kit := kit.New(cfg, log, kit.WithCSRFFunc(csrf.Token))

// Then in handlers, get token from kit
token := kit.CSRFToken(r.Context())

// Pass to forms
form := kit.NewForm().Action("/submit").CSRFToken(token)
deleteBtn := kit.NewDeleteButton("Delete", "/items/1").CSRFToken(token)
```

## Components

### Chip (rounded) and Label (squared)

```go
chip := kit.NewChip("Active").Emoji(kit.EmojiCheck).Success()
label := kit.NewLabel("Important").Emoji(kit.EmojiWarning).Warning()
```

### Button

```go
btn := kit.NewButton("Save").Emoji(kit.EmojiSave).Primary()
btn := kit.NewButton("Delete").Emoji(kit.EmojiTrash).Danger()

// With HTMX
btn := kit.NewButton("Load").HX().Get("/data").TargetID("result").Done()
```

### Layout

```go
page := kit.NewPage("Dashboard").
    Header(header).
    Content(content).
    Footer(footer)

header := kit.NewPageHeader("Settings").
    Subtitle("Configure your app").
    Breadcrumbs(
        kit.Breadcrumb{Label: "Home", Href: "/"},
        kit.Breadcrumb{Label: "Settings"},
    ).
    Actions(saveBtn, cancelBtn)

container := kit.NewContainer().Content(html).Fluid()
```

### Navigation

```go
nav := kit.NewNavGrid().
    AddItem(kit.EmojiHome, "Dashboard", "/").
    AddItem(kit.EmojiSettings, "Settings", "/settings").
    AddItemWithBadge(kit.EmojiMail, "Messages", "/messages", "5").
    Cols(3)

menu := kit.NewNav().
    AddLink("Home", "/", true).
    AddLink("About", "/about", false).
    Vertical()
```

### Table

```go
table := kit.NewTable().
    Columns(
        kit.Col("name", "Name").WithWidth("200px"),
        kit.Col("email", "Email"),
        kit.Col("actions", "").WithAlign("right"),
    ).
    Rows(
        kit.NewRow(kit.Text("Alice"), kit.Text("alice@example.com"), kit.HTML(actions)),
        kit.NewRow(kit.Text("Bob"), kit.Text("bob@example.com"), kit.HTML(actions)),
    ).
    Striped().
    Hoverable().
    EmptyText("No users found")
```

### Form (with CSRF support)

```go
form := kit.NewForm().
    Action("/submit").
    CSRFToken(csrfToken).
    Post()

// In template
{{ form.Open }}
  <input type="text" name="email">
  <button type="submit">Submit</button>
{{ form.Close }}

// With HTMX
form := kit.NewForm().
    HX().Post("/api/submit").TargetID("result").Done().
    CSRFToken(csrfToken)
```

### Delete Button (form-based, not link)

```go
// Safe delete: renders as form, not link (bots can't accidentally trigger)
deleteBtn := kit.NewDeleteButton("Delete", "/items/123").
    CSRFToken(csrfToken).
    Confirm("Are you sure?").
    Emoji(kit.EmojiTrash)

// With HTMX
deleteBtn := kit.NewDeleteButton("Delete", "").
    HX().Delete("/api/items/123").SwapDelete().Confirm("Sure?").Done().
    CSRFToken(csrfToken)
```

### Settings Form

```go
schemas := []settings.Schema{
    {Key: "app.name", Type: settings.String, Label: "App Name", Required: true},
    {Key: "app.debug", Type: settings.Bool, Label: "Debug Mode"},
    {Key: "theme", Type: settings.Enum, Options: []string{"light", "dark"}},
}

form := kit.NewSettingsForm(schemas).
    Values(currentValues).
    Errors(validationErrors).
    Action("/settings").
    SubmitButton("Save Changes")
```

### Assets

```go
assets := kit.NewAssets(embeddedFS).
    WithOverlay(os.DirFS("./custom")).
    WithPrefix("/static")

mux.Handle("/static/", assets.Handler())

// In templates
url := assets.URL("css/style.css") // "/static/css/style.css"
```

## Emoji Presets

```go
kit.EmojiCheck    // ✅
kit.EmojiCross    // ❌
kit.EmojiWarning  // ⚠️
kit.EmojiInfo     // ℹ️
kit.EmojiStar     // ⭐
kit.EmojiHeart    // ❤️
kit.EmojiTrash    // 🗑
kit.EmojiSettings // ⚙
kit.EmojiUser     // 👤
kit.EmojiHome     // 🏠
kit.EmojiSave     // 💾
// ... see emoji.go for full list
```

Custom emojis:

```go
chip := kit.NewChip("Coffee").Emoji("☕")
```

## Variants

Chip, Label, and Button support: Primary, Secondary, Success, Warning, Danger, Info, Muted.
