# Correcciones para Templates del Generador

## Problemas detectados y correcciones aplicadas

### 1. **Queries para agregados (listqueries.go)**

**Problema**: Las queries no incluían el campo `id` ni los timestamps en las operaciones de agregados.

**Corrección aplicada**:
```sql
-- Antes:
QueryCreateListRoot = `INSERT INTO lists (name, description) VALUES (?, ?)`
QueryGetListRoot = `SELECT name, description FROM lists WHERE id = ?`
QueryUpdateListRoot = `UPDATE lists SET name = ?, description = ? WHERE id = ?`

-- Después:
QueryCreateListRoot = `INSERT INTO lists (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
QueryGetListRoot = `SELECT id, name, description, created_at, updated_at FROM lists WHERE id = ?`
QueryUpdateListRoot = `UPDATE lists SET name = ?, description = ?, updated_at = ? WHERE id = ?`
```

**Template a modificar**: `internal/templates/sqlite/queries.tmpl`

### 2. **Método insertRoot (listrepo.go)**

**Problema**: El método no insertaba el ID del aggregate ni los timestamps.

**Corrección aplicada**:
```go
// Antes:
func (r *{{.AggregateRoot.Name}}SQLiteRepo) insertRoot(ctx context.Context, tx *sql.Tx, aggregate *{{.PackageName}}.{{.AggregateRoot.Name}}) error {
    _, err := tx.ExecContext(ctx, QueryCreate{{.AggregateRoot.Name}}Root, aggregate.Name, aggregate.Description)
    return err
}

// Después:
func (r *{{.AggregateRoot.Name}}SQLiteRepo) insertRoot(ctx context.Context, tx *sql.Tx, aggregate *{{.PackageName}}.{{.AggregateRoot.Name}}) error {
    _, err := tx.ExecContext(ctx, QueryCreate{{.AggregateRoot.Name}}Root, aggregate.ID.String(), aggregate.Name, aggregate.Description, aggregate.CreatedAt, aggregate.UpdatedAt)
    return err
}
```

**Template a modificar**: `internal/templates/sqlite/repository.tmpl`

### 3. **Método getRoot (listrepo.go)**

**Problema**: El Scan no coincidía con los campos SELECT.

**Corrección aplicada**:
```go
// Antes:
err := r.db.QueryRowContext(ctx, QueryGet{{.AggregateRoot.Name}}Root, id.String()).Scan(
    &aggregate.Name, &aggregate.Description,
)

// Después:
err := r.db.QueryRowContext(ctx, QueryGet{{.AggregateRoot.Name}}Root, id.String()).Scan(
    &idStr, &aggregate.Name, &aggregate.Description, &aggregate.CreatedAt, &aggregate.UpdatedAt,
)
```

**Template a modificar**: `internal/templates/sqlite/repository.tmpl`

### 4. **Método updateRoot (listrepo.go)**

**Problema**: No incluía el timestamp `updated_at` en la actualización.

**Corrección aplicada**:
```go
// Antes:
result, err := tx.ExecContext(ctx, QueryUpdate{{.AggregateRoot.Name}}Root, aggregate.Name, aggregate.Description, aggregate.ID.String())

// Después:
result, err := tx.ExecContext(ctx, QueryUpdate{{.AggregateRoot.Name}}Root, aggregate.Name, aggregate.Description, aggregate.UpdatedAt, aggregate.ID.String())
```

**Template a modificar**: `internal/templates/sqlite/repository.tmpl`

### 5. **Métodos de transacción para child entities**

**Problema**: Los métodos `saveItems` y `saveTags` usaban `r.db` en lugar de la transacción `tx`.

**Corrección aplicada**:
- Crear métodos `getItemsWithTx` y `getTagsWithTx` que acepten tanto `*sql.DB` como `*sql.Tx`
- Modificar los métodos `saveItems` y `saveTags` para usar las versiones con transacción

**Template a modificar**: `internal/templates/sqlite/repository.tmpl`

## Patrón genérico aplicado

Estas correcciones siguen un patrón genérico que debe aplicarse a todos los agregados:

1. **IDs y timestamps**: Siempre incluir el ID del aggregate root y los timestamps en todas las operaciones
2. **Consistencia SELECT/Scan**: Los campos en SELECT deben coincidir exactamente con los campos en Scan
3. **Transacciones**: Todos los métodos que manejan child entities en transacciones deben usar la transacción, no la conexión directa
4. **Naming**: Seguir convenciones sin underscores en nombres de métodos y funciones

## Prioridad de implementación

1. **Alta**: Correcciones 1-4 (críticas para funcionalidad básica)
2. **Media**: Corrección 5 (necesaria para operaciones Save complejas)
3. **Baja**: Mejoras de consistencia y naming

## Tests de validación

El archivo `listrepo_test.go` sirve como referencia de tests table-based que validan:
- CRUD completo de agregados
- Manejo correcto de child entities
- Casos de error
- Integridad transaccional

Este patrón de test debe templetizarse para todos los agregados generados.