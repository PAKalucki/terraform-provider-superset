---
page_title: "superset_chart Resource - superset"
subcategory: ""
description: |-
  Manages a Superset chart.
---

# superset_chart (Resource)

Manages a Superset chart.

Use `jsonencode(...)` for `params` and `query_context` so Terraform and Superset agree on the stored JSON representation. `datasource_type` defaults to `table`.

## Example Usage

```terraform
resource "superset_database" "warehouse" {
  database_name  = "analytics"
  sqlalchemy_uri = "postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"
}

resource "superset_dataset" "events" {
  database_id           = superset_database.warehouse.id
  schema                = "analytics"
  table_name            = "events"
  main_dttm_col         = "created_at"
  filter_select_enabled = true
}

locals {
  datasource_uid = format("%d__table", superset_dataset.events.id)
}

resource "superset_chart" "events_table" {
  slice_name    = "Events table"
  datasource_id = superset_dataset.events.id
  viz_type      = "table"

  params = jsonencode({
    datasource = local.datasource_uid
    viz_type   = "table"
  })

  query_context = jsonencode({
    datasource = {
      id   = superset_dataset.events.id
      type = "table"
    }
    force = false
    queries = [
      {
        filters             = []
        extras              = { having = "", where = "" }
        applied_time_extras = {}
        columns             = ["id"]
        orderby             = [["id", true]]
        annotation_layers   = []
        row_limit           = 1000
        series_limit        = 0
        order_desc          = true
        url_params          = {}
        custom_params       = {}
        custom_form_data    = {}
      }
    ]
    form_data = {
      datasource = local.datasource_uid
      viz_type   = "table"
    }
    result_format = "json"
    result_type   = "full"
  })
}
```

## Schema

### Required

- `datasource_id` (Number) Superset datasource identifier for the chart.
- `params` (String) Chart form data JSON string.
- `slice_name` (String) Human-readable chart name in Superset.
- `viz_type` (String) Superset visualization type, for example `table` or `bar`.

### Optional

- `cache_timeout` (Number) Optional chart cache timeout in seconds.
- `datasource_type` (String) Superset datasource type. Defaults to `table`.
- `description` (String) Optional chart description.
- `query_context` (String) Optional chart query context JSON string.

### Read-Only

- `datasource_name` (String) Resolved Superset datasource name.
- `id` (Number) Superset chart identifier.
- `url` (String) Resolved Superset chart URL.
- `uuid` (String) Superset chart UUID.
