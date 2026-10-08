<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# ui

UI kit with dependency injection, emoji support, and HTMX-first components.

## Usage

The Go blocks are independent fragments for the complete page/view composition
in the [guide companion](../examples/guide/main.go), not consecutive declarations
in one function. Supply the application-owned dependencies and view fields named
in each fragment. The [UI reference](../docs/reference/ui/README.md) gives exact
constructor and option contracts. Templates use `ui.FuncMap()` or `kit.FuncMap()`.

```go
kit := ui.New(cfg, log,
    ui.WithSettings(settingsSvc),
    ui.WithOverlay(os.DirFS("./custom")),
    ui.WithCSRFFunc(tokenFromContext),  // integrate with your CSRF middleware
)

// Template functions
tmpl := template.New("page").Funcs(kit.FuncMap())

// Or standalone (no kit instance)
standalone := template.New("page").Funcs(ui.FuncMap())
```

## CSRF Integration

`WithCSRFFunc` accepts `func(context.Context) string`. The application supplies
`tokenFromContext` and the middleware that validates the submitted token.
Installing this callback or rendering a hidden input does not validate CSRF:

```go
// Your CSRF package provides a function to extract token from context
kit := ui.New(cfg, log, ui.WithCSRFFunc(tokenFromContext))

// Then in handlers, get token from kit
token := kit.CSRFToken(r.Context())

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
deleteButton := ui.NewButton("Delete").Emoji(ui.EmojiTrash).Danger()

// With HTMX
loadButton := ui.NewButton("Load").HX().Get("/data").TargetID("result").Done()
```

### Layout

```go
header := ui.NewPageHeader("Settings").
    Subtitle("Configure your app").
    Breadcrumbs(
        ui.Breadcrumb{Label: "Home", Href: "/"},
        ui.Breadcrumb{Label: "Settings"},
    ).
    Actions(saveBtn, cancelBtn)

page := ui.NewPage("Dashboard").Header(header).Content(content).Footer(footer)

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

```

The view carries the form as `.Form`:

```html
{{ .Form.Open }}
  <input type="text" name="email">
  <button type="submit">Submit</button>
{{ .Form.Close }}
```

With HTMX:

```go
form := ui.NewForm().
    HX().Post("/api/submit").TargetID("result").Done().
    CSRFToken(csrfToken)
```

### Delete Button (form-based, not link)

```go
// Native deletion uses a POST form; the application still enforces policy.
deleteBtn := ui.NewDeleteButton("Delete", "/items/123").
    CSRFToken(csrfToken).
    Confirm("Are you sure?").
    Emoji(ui.EmojiTrash)

// With HTMX
htmxDelete := ui.NewDeleteButton("Delete", "").
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

```text
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
savedAlert := ui.AlertSuccess("Saved successfully!")

flash := ui.NewFlash("Changes saved").Success().AutoDismiss(5)

toast := ui.NewToast("New message").
    Title("Notification").
    Position("top-right").
    Duration(3000)
```

### Link

```go
link := ui.NewLink("Click here", "/page")
externalLink := ui.ABlank("External", "https://example.com") // target="_blank"
boostedLink := ui.ABoosted("Navigate", "/page")             // hx-boost

// With HTMX
loadLink := ui.NewLink("Load", "/").HX().Get("/api").TargetID("content").Done()
```

### StatusBadge

```go
badge := ui.StatusBadge("active")   // green, "Active"
draftBadge := ui.StatusBadge("draft")    // yellow, "Draft"
expiredBadge := ui.StatusBadge("expired")  // red, "Expired"

// With icon
iconBadge := ui.StatusBadgeWithIcon("active") // "● Active"

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

Chip and Label supply all seven named variant methods. Button supplies Primary,
Secondary, Success, Warning and Danger. Alert supplies Info, Success, Warning,
Danger and Error. Use `Variant` for an explicit variant value; the renderer and
application styles determine its appearance.
