# ui

UI kit with dependency injection, emoji support, and HTMX-first components.

## Usage

```go
ui := ui.New(cfg, log,
    ui.WithSettings(settingsSvc),
    ui.WithOverlay(os.DirFS("./custom")),
    ui.WithCSRFFunc(csrf.Token),  // integrate with your CSRF middleware
)

// Template functions
tmpl := template.New("page").Funcs(ui.FuncMap())

// Or standalone (no kit instance)
tmpl := template.New("page").Funcs(ui.FuncMap())
```

## CSRF Integration

The kit integrates with external CSRF middleware via `WithCSRFFunc`:

```go
// Your CSRF package provides a function to extract token from context
ui := ui.New(cfg, log, ui.WithCSRFFunc(csrf.Token))

// Then in handlers, get token from kit
token := ui.CSRFToken(r.Context())

// Pass to forms
form := ui.NewForm().Action("/submit").CSRFToken(token)
deleteBtn := ui.NewDeleteButton("Delete", "/items/1").CSRFToken(token)
```

## Components

### Chip (rounded) and Label (squared)

```go
chip := ui.NewChip("Active").Emoji(ui.EmojiCheck).Success()
label := ui.NewLabel("Important").Emoji(ui.EmojiWarning).Warning()
```

### Button

```go
btn := ui.NewButton("Save").Emoji(ui.EmojiSave).Primary()
btn := ui.NewButton("Delete").Emoji(ui.EmojiTrash).Danger()

// With HTMX
btn := ui.NewButton("Load").HX().Get("/data").TargetID("result").Done()
```

### Layout

```go
page := ui.NewPage("Dashboard").
    Header(header).
    Content(content).
    Footer(footer)

header := ui.NewPageHeader("Settings").
    Subtitle("Configure your app").
    Breadcrumbs(
        ui.Breadcrumb{Label: "Home", Href: "/"},
        ui.Breadcrumb{Label: "Settings"},
    ).
    Actions(saveBtn, cancelBtn)

container := ui.NewContainer().Content(html).Fluid()
```

### Navigation

```go
nav := ui.NewNavGrid().
    AddItem(ui.EmojiHome, "Dashboard", "/").
    AddItem(ui.EmojiSettings, "Settings", "/settings").
    AddItemWithBadge(ui.EmojiMail, "Messages", "/messages", "5").
    Cols(3)

menu := ui.NewNav().
    AddLink("Home", "/", true).
    AddLink("About", "/about", false).
    Vertical()
```

### Table

```go
table := ui.NewTable().
    Columns(
        ui.Col("name", "Name").WithWidth("200px"),
        ui.Col("email", "Email"),
        ui.Col("actions", "").WithAlign("right"),
    ).
    Rows(
        ui.NewRow(ui.Text("Alice"), ui.Text("alice@example.com"), ui.HTML(actions)),
        ui.NewRow(ui.Text("Bob"), ui.Text("bob@example.com"), ui.HTML(actions)),
    ).
    Striped().
    Hoverable().
    EmptyText("No users found")
```

### Form (with CSRF support)

```go
form := ui.NewForm().
    Action("/submit").
    CSRFToken(csrfToken).
    Post()

// In template
{{ form.Open }}
  <input type="text" name="email">
  <button type="submit">Submit</button>
{{ form.Close }}

// With HTMX
form := ui.NewForm().
    HX().Post("/api/submit").TargetID("result").Done().
    CSRFToken(csrfToken)
```

### Delete Button (form-based, not link)

```go
// Safe delete: renders as form, not link (bots can't accidentally trigger)
deleteBtn := ui.NewDeleteButton("Delete", "/items/123").
    CSRFToken(csrfToken).
    Confirm("Are you sure?").
    Emoji(ui.EmojiTrash)

// With HTMX
deleteBtn := ui.NewDeleteButton("Delete", "").
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

form := ui.NewSettingsForm(schemas).
    Values(currentValues).
    Errors(validationErrors).
    Action("/settings").
    SubmitButton("Save Changes")
```

### Assets

```go
assets := ui.NewAssets(embeddedFS).
    WithOverlay(os.DirFS("./custom")).
    WithPrefix("/static")

mux.Handle("/static/", assets.Handler())

// In templates
url := assets.URL("css/style.css") // "/static/css/style.css"
```

## Emoji Presets

```go
ui.EmojiCheck    // ✅
ui.EmojiCross    // ❌
ui.EmojiWarning  // ⚠️
ui.EmojiInfo     // ℹ️
ui.EmojiStar     // ⭐
ui.EmojiHeart    // ❤️
ui.EmojiTrash    // 🗑
ui.EmojiSettings // ⚙
ui.EmojiUser     // 👤
ui.EmojiHome     // 🏠
ui.EmojiSave     // 💾
// ... see emoji.go for full list
```

Custom emojis:

```go
chip := ui.NewChip("Coffee").Emoji("☕")
```

### Alerts, Flash, and Toast

```go
alert := ui.NewAlert("Something happened").Info().Dismissible()
alert := ui.AlertSuccess("Saved successfully!")

flash := ui.NewFlash("Changes saved").Success().AutoDismiss(5)

toast := ui.NewToast("New message").
    Title("Notification").
    Position("top-right").
    Duration(3000)
```

### Link

```go
link := ui.NewLink("Click here", "/page")
link := ui.ABlank("External", "https://example.com") // target="_blank"
link := ui.ABoosted("Navigate", "/page")             // hx-boost

// With HTMX
link := ui.NewLink("Load", "/").HX().Get("/api").TargetID("content").Done()
```

### StatusBadge

```go
badge := ui.StatusBadge("active")   // green, "Active"
badge := ui.StatusBadge("draft")    // yellow, "Draft"
badge := ui.StatusBadge("expired")  // red, "Expired"

// With icon
badge := ui.StatusBadgeWithIcon("active") // "● Active"

// Register custom status
ui.RegisterStatus("pending", ui.StatusConfig{
    Variant: ui.VariantWarning,
    Label:   "Pending Review",
    Icon:    "⏳",
})
```

## Format Integration

Formatting functions from `format` package are available in templates:

```html
<span class="price">{{ formatPrice .Amount .Currency }}</span>
<span class="count">{{ formatNumber .Count }}</span>
```

## Variants

Chip, Label, Button, and Alert support: Primary, Secondary, Success, Warning, Danger, Info, Muted.
