---
page_title: "superset_saved_query Resource - superset"
subcategory: ""
description: |-
  Manages a Superset saved query.
---

# superset_saved_query (Resource)

Manages a Superset saved query.

Use `jsonencode(...)` for `template_parameters` and `extra_json` so Terraform and Superset store the same normalized JSON. When those attributes are omitted, the provider leaves their current values unchanged.

## Example Usage

```terraform
resource "superset_database" "warehouse" {
  database_name  = "analytics"
  sqlalchemy_uri = "postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"
}

resource "superset_saved_query" "orders" {
  database_id = superset_database.warehouse.id
  label       = "Orders Preview"
  schema      = "analytics"
  sql         = "select * from orders limit 100"

  template_parameters = jsonencode({
    region = "emea"
  })

  extra_json = jsonencode({
    editor_width = 640
  })
}
```

## Schema

### Required

- `database_id` (Number) Superset database identifier used by the saved query.
- `label` (String) Saved query label shown in SQL Lab.
- `sql` (String) SQL text stored in the saved query.

### Optional

- `catalog` (String) Optional catalog for the saved query.
- `description` (String) Optional saved query description.
- `extra_json` (String) Optional JSON string for saved-query metadata. When omitted, the provider leaves the current value unchanged.
- `schema` (String) Optional database schema for the saved query.
- `template_parameters` (String) Optional JSON string for template parameters. When omitted, the provider leaves the current value unchanged.

### Read-Only

- `database_name` (String) Resolved Superset database name.
- `id` (Number) Superset saved query identifier.

## Import

Import a saved query by its numeric Superset id.

```terraform
import {
  to = superset_saved_query.orders
  id = "42"
}
```

```shell
terraform import superset_saved_query.orders 42
```

Superset does not always return `extra_json` on read, so configure `extra_json` explicitly after import if you want Terraform to keep managing that field authoritatively.
