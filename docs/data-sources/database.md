---
page_title: "superset_database Data Source - superset"
subcategory: ""
description: |-
  Reads an existing Superset database connection.
---

# superset_database (Data Source)

Reads an existing Superset database connection by `id` or `database_name`.

Superset masks stored credentials when it returns connection details, so `sqlalchemy_uri` is returned with masked credentials in the data source.

## Example Usage

```terraform
data "superset_database" "warehouse" {
  database_name = "analytics"
}
```

## Schema

### Optional

- `database_name` (String) Human-readable name for the Superset database connection. Configure this or `id`.
- `id` (Number) Superset database identifier. Configure this or `database_name`.

### Read-Only

- `allow_ctas` (Boolean) Whether `CREATE TABLE AS` statements are allowed in SQL Lab.
- `allow_cvas` (Boolean) Whether `CREATE VIEW AS` statements are allowed in SQL Lab.
- `allow_dml` (Boolean) Whether non-SELECT statements are allowed in SQL Lab.
- `allow_file_upload` (Boolean) Whether CSV uploads are allowed for this database.
- `allow_run_async` (Boolean) Whether queries on this database run asynchronously.
- `backend` (String) Resolved Superset database backend.
- `cache_timeout` (Number) Database-level chart cache timeout in seconds.
- `database_name` (String) Human-readable name for the Superset database connection.
- `driver` (String) Resolved SQLAlchemy driver.
- `expose_in_sqllab` (Boolean) Whether the database is exposed in SQL Lab.
- `extra` (String, Sensitive) Database `extra` JSON string returned by Superset.
- `force_ctas_schema` (String) Schema enforced for `CREATE TABLE AS` statements when enabled.
- `impersonate_user` (Boolean) Whether Superset impersonates the current user when querying this database.
- `sqlalchemy_uri` (String, Sensitive) SQLAlchemy connection URI returned by Superset with masked credentials.
- `uuid` (String) Superset database UUID.
