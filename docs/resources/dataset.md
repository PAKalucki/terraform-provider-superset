---
page_title: "superset_dataset Resource - superset"
subcategory: ""
description: |-
  Manages a Superset physical dataset.
---

# superset_dataset (Resource)

Manages a Superset physical dataset backed by a Superset database connection.

When `columns` or `metrics` are configured, they are authoritative and replace the corresponding collection in Superset. Omit either attribute to leave that collection unmanaged by Terraform.

## Example Usage

```terraform
resource "superset_database" "warehouse" {
  database_name  = "analytics"
  sqlalchemy_uri = "postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"
}

resource "superset_dataset" "events" {
  database_id          = superset_database.warehouse.id
  schema               = "analytics"
  table_name           = "events"
  description          = "Warehouse events dataset"
  main_dttm_col        = "created_at"
  filter_select_enabled = true

  columns = [
    {
      column_name  = "id"
      verbose_name = "Event ID"
      filterable   = true
      groupby      = true
      is_active    = true
      type         = "INTEGER"
    },
    {
      column_name = "created_at"
      filterable  = true
      groupby     = true
      is_active   = true
      is_dttm     = true
      type        = "TIMESTAMP"
    }
  ]

  metrics = [
    {
      metric_name  = "event_count"
      expression   = "COUNT(*)"
      metric_type  = "count"
      verbose_name = "Event Count"
    }
  ]
}
```

## Schema

### Required

- `database_id` (Number) Superset database identifier that owns the dataset.
- `table_name` (String) Dataset table name.

### Optional

- `always_filter_main_dttm` (Boolean) Whether the main datetime column is always filtered.
- `cache_timeout` (Number) Dataset cache timeout in seconds.
- `columns` (Attributes List) Authoritative list of dataset columns when configured.
- `description` (String) Dataset description.
- `filter_select_enabled` (Boolean) Whether filter select is enabled for the dataset.
- `main_dttm_col` (String) Main datetime column used by Superset.
- `metrics` (Attributes List) Authoritative list of dataset metrics when configured.
- `normalize_columns` (Boolean) Whether Superset should normalize columns on create or update.
- `schema` (String) Dataset schema name.

### Read-Only

- `database_name` (String) Resolved Superset database name.
- `id` (Number) Superset dataset identifier.
- `uuid` (String) Superset dataset UUID.
