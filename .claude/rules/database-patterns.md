# Database Patterns (PocketBase)

## Collection Setup
Collections are created idempotently in `collections/setup.go`:
```go
col := ensureCollection(app, "collection_name", func(c *core.Collection) {
    c.Fields.Add(&core.TextField{Name: "name", Required: true})
    c.Fields.Add(&core.NumberField{Name: "qty", Required: true})
    c.Fields.Add(&core.BoolField{Name: "active"})
    c.Fields.Add(&core.SelectField{
        Name: "status", Required: true,
        Values: []string{"active", "completed"},
        MaxSelect: 1,
    })
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
```

## Adding Fields to Existing Collections
```go
ensureField(app, "collection_name", &core.RelationField{
    Name:          "parent",
    Required:      false,
    CollectionId:  parentCol.Id,
    CascadeDelete: true,
    MaxSelect:     1,
})
```

## Relations
- Use `core.RelationField` with `CascadeDelete: true` for parent-child
- Hierarchy: `projects` → `boqs` → `main_boq_items` → `sub_items` → `sub_sub_items`
- Self-referential: `ship_to_parent` on addresses

## Record CRUD
```go
// Find
record, err := app.FindRecordById("collection", id)
records, err := app.FindRecordsByFilter("collection", "field = {:val}", "sort", limit, offset, params)
records, err := app.FindAllRecords(collection)

// Create
col, err := app.FindCollectionByNameOrId("collection")
record := core.NewRecord(col)
record.Set("field", value)
err := app.Save(record)

// Update
record.Set("field", newValue)
err := app.Save(record)

// Delete
err := app.Delete(record)
```

## Field Access
```go
record.GetString("field")
record.GetBool("field")
record.GetFloat64("field")
record.GetInt("field")
record.GetDateTime("field")
```

## Filter Parameters
Always use parameterized filters to prevent injection:
```go
app.FindRecordsByFilter("projects", "name = {:name}", "", 1, 0,
    map[string]any{"name": name})
```
