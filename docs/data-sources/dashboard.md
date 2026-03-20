---
page_title: "superset_dashboard Data Source - superset"
subcategory: ""
description: |-
  Reads an existing Superset dashboard.
---

# superset_dashboard (Data Source)

Reads an existing Superset dashboard by `id`, `slug`, or `dashboard_title`.

## Example Usage

```terraform
data "superset_dashboard" "operations" {
  slug = "operations-dashboard"
}
```

## Schema

### Optional

- `dashboard_title` (String) Human-readable dashboard title used for lookup. Configure this, or configure `id` or `slug`.
- `id` (Number) Superset dashboard identifier. Configure this, or configure `slug` or `dashboard_title`.
- `slug` (String) Dashboard slug used for lookup. Configure this, or configure `id` or `dashboard_title`.

### Read-Only

- `chart_ids` (List of Number) Superset chart identifiers associated with the dashboard.
- `css` (String) Custom dashboard CSS.
- `position_json` (String) Superset dashboard layout JSON string.
- `published` (Boolean) Whether the dashboard is published in Superset.
- `url` (String) Resolved Superset dashboard URL.
- `uuid` (String) Superset dashboard UUID.
