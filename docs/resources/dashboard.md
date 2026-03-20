---
page_title: "superset_dashboard Resource - superset"
subcategory: ""
description: |-
  Manages a Superset dashboard.
---

# superset_dashboard (Resource)

Manages a Superset dashboard.

Use `chart_ids` for the common case where you want Terraform to associate charts and let the provider generate a simple default layout. Use `position_json` when you need to control the exact dashboard layout. When both are configured, the chart identifiers referenced in `position_json` must match `chart_ids`.

The provider-generated layout is intentionally simple: charts are placed in a single row with fixed Superset dimensions. Switch to `position_json` when you need custom sizing or placement.

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
}

resource "superset_dashboard" "operations" {
  dashboard_title = "Operations dashboard"
  slug            = "operations-dashboard"
  published       = true
  chart_ids       = [superset_chart.events_table.id]

  position_json = jsonencode({
    DASHBOARD_VERSION_KEY = "v2"
    ROOT_ID = {
      children = ["GRID_ID"]
      id       = "ROOT_ID"
      type     = "ROOT"
    }
    GRID_ID = {
      children = ["ROW-1"]
      id       = "GRID_ID"
      parents  = ["ROOT_ID"]
      type     = "GRID"
    }
    HEADER_ID = {
      id   = "HEADER_ID"
      meta = { text = "Operations dashboard" }
      type = "HEADER"
    }
    "ROW-1" = {
      children = ["CHART-1"]
      id       = "ROW-1"
      meta     = { "0" = "ROOT_ID", background = "BACKGROUND_TRANSPARENT" }
      parents  = ["ROOT_ID", "GRID_ID"]
      type     = "ROW"
    }
    "CHART-1" = {
      children = []
      id       = "CHART-1"
      meta = {
        chartId   = superset_chart.events_table.id
        height    = 50
        sliceName = superset_chart.events_table.slice_name
        uuid      = superset_chart.events_table.uuid
        width     = 4
      }
      parents = ["ROOT_ID", "GRID_ID", "ROW-1"]
      type    = "CHART"
    }
  })
}
```

## Schema

### Required

- `dashboard_title` (String) Human-readable dashboard title in Superset.

### Optional

- `chart_ids` (List of Number) Optional list of Superset chart identifiers associated with the dashboard. When configured without `position_json`, the provider generates a simple default layout for those charts.
- `css` (String) Optional custom CSS for the dashboard.
- `position_json` (String) Optional Superset dashboard layout JSON string. When configured, the chart identifiers referenced in the layout become the authoritative dashboard-chart associations.
- `published` (Boolean) Whether the dashboard is published in Superset.
- `slug` (String) Optional dashboard slug used in the Superset URL.

### Read-Only

- `id` (Number) Superset dashboard identifier.
- `url` (String) Resolved Superset dashboard URL.
- `uuid` (String) Superset dashboard UUID.
