---
page_title: "superset_chart Data Source - superset"
subcategory: ""
description: |-
  Reads an existing Superset chart.
---

# superset_chart (Data Source)

Reads an existing Superset chart by `id`, or by `datasource_id` with `slice_name` and optional `datasource_type`.

## Example Usage

```terraform
data "superset_chart" "events_table" {
  datasource_id = superset_dataset.events.id
  slice_name    = "Events table"
}
```

## Schema

### Optional

- `datasource_id` (Number) Superset datasource identifier used for composite lookup. Configure this with `slice_name` and optional `datasource_type`, or configure `id`.
- `datasource_type` (String) Optional datasource type used for composite lookup.
- `id` (Number) Superset chart identifier. Configure this, or configure `datasource_id` with `slice_name` and optional `datasource_type`.
- `slice_name` (String) Human-readable chart name used for composite lookup.

### Read-Only

- `cache_timeout` (Number) Chart cache timeout in seconds.
- `datasource_name` (String) Resolved Superset datasource name.
- `description` (String) Chart description.
- `params` (String) Chart form data JSON string returned by Superset.
- `query_context` (String) Chart query context JSON string returned by Superset.
- `uuid` (String) Superset chart UUID.
- `url` (String) Resolved Superset chart URL.
- `viz_type` (String) Superset visualization type.
