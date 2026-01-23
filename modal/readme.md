# modal

Modal dialog configuration for templates.

## Usage

```go
// In handler
cfg := modal.DefaultConfig("delete-modal", "Confirm Delete")
cfg.Size = modal.SizeLarge

// In template
<div id="{{.Modal.ID}}" class="modal {{.Modal.Size}}">
    <h2>{{.Modal.Title}}</h2>
    ...
</div>
```

Sizes: `SizeSmall`, `SizeMedium`, `SizeLarge`.
