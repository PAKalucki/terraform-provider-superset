---
page_title: "superset_dataset Data Source - superset"
subcategory: ""
description: |-
  Reads an existing Superset dataset.
---

# superset_dataset (Data Source)

Reads an existing Superset dataset by `id`, or by `database_id` with `table_name` and optional `schema`.

When `schema` is omitted, the lookup matches a unique dataset by `database_id` and `table_name`. Configure `schema` to disambiguate duplicate table names.

## Example Usage

```terraform
data "superset_dataset" "events" {
  database_id = superset_database.warehouse.id
  schema      = "analytics"
  table_name  = "events"
}
```

## Schema

### Optional

- `database_id` (Number) Superset database identifier used for composite lookup. Configure this with `table_name` and optional `schema`, or configure `id`.
- `id` (Number) Superset dataset identifier. Configure this, or configure `database_id` with `table_name` and optional `schema`.
- `schema` (String) Dataset schema name used for composite lookup.
- `table_name` (String) Dataset table name used for composite lookup.

### Read-Only

- `always_filter_main_dttm` (Boolean) Whether the main datetime column is always filtered.
- `cache_timeout` (Number) Dataset cache timeout in seconds.
- `columns` (Attributes List) Dataset columns returned by Superset.
- `database_name` (String) Resolved Superset database name.
- `description` (String) Dataset description.
- `filter_select_enabled` (Boolean) Whether filter select is enabled for the dataset.
- `main_dttm_col` (String) Main datetime column used by Superset.
- `metrics` (Attributes List) Dataset metrics returned by Superset.
- `normalize_columns` (Boolean) Whether Superset normalizes columns for the dataset.
- `uuid` (String) Superset dataset UUID.
