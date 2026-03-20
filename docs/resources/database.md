---
page_title: "superset_database Resource - superset"
subcategory: ""
description: |-
  Manages a Superset database connection.
---

# superset_database (Resource)

Manages a Superset database connection.

Use `jsonencode(...)` for `extra` so Terraform and Superset agree on the stored JSON representation. Superset masks stored credentials when a database is read back from the API, so the provider preserves the configured `sqlalchemy_uri` value in Terraform state.

## Example Usage

```terraform
resource "superset_database" "warehouse" {
  database_name  = "analytics"
  sqlalchemy_uri = "postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"

  expose_in_sqllab = true

  extra = jsonencode({
    metadata_cache_timeout = {
      schema_cache_timeout = 600
    }
  })
}
```

## Schema

### Required

- `database_name` (String) Human-readable name for the Superset database connection.
- `sqlalchemy_uri` (String, Sensitive) SQLAlchemy connection URI used by Superset to reach the database.

### Optional

- `allow_ctas` (Boolean) Whether `CREATE TABLE AS` statements are allowed in SQL Lab.
- `allow_cvas` (Boolean) Whether `CREATE VIEW AS` statements are allowed in SQL Lab.
- `allow_dml` (Boolean) Whether non-SELECT statements are allowed in SQL Lab.
- `allow_file_upload` (Boolean) Whether CSV uploads are allowed for this database.
- `allow_run_async` (Boolean) Whether queries on this database run asynchronously.
- `cache_timeout` (Number) Database-level chart cache timeout in seconds.
- `expose_in_sqllab` (Boolean) Whether to expose the database in SQL Lab.
- `extra` (String, Sensitive) Optional database `extra` JSON string.
- `force_ctas_schema` (String) Schema enforced for `CREATE TABLE AS` statements when enabled.
- `impersonate_user` (Boolean) Whether Superset impersonates the current user when querying this database.

### Read-Only

- `backend` (String) Resolved Superset database backend.
- `driver` (String) Resolved SQLAlchemy driver.
- `id` (Number) Superset database identifier.
- `uuid` (String) Superset database UUID.
