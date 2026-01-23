# model

Base types for domain models: IDs, timestamps, passwords, roles.

## Usage

```go
// IDs
id := model.NewID()                    // "550e8400-e29b-41d4-a716-446655440000"
model.GenerateID(&user.ID)             // sets in place
parsed, err := model.ParseID(str)

// Nullable UUIDs (for optional foreign keys)
nullID := model.NullUUID(&parentID)    // string ptr -> uuid.NullUUID
strPtr := model.FromNullUUID(nullID)   // uuid.NullUUID -> string ptr

// Timestamps
now := model.Now()                     // time.Time truncated to seconds

// Passwords
hash, err := model.HashPassword(password)
ok := model.ComparePassword(hash, password)

// Roles
role := model.RoleAdmin
if role.HasPermission(model.PermWrite) { ... }
```

See individual files: `id.go`, `time.go`, `password.go`, `roles.go`.
