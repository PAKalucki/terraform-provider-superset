---
page_title: "superset_permission Data Source - superset"
subcategory: ""
description: |-
  Reads an existing Superset permission-view-menu resource.
---

# superset_permission (Data Source)

Reads an existing Superset permission-view-menu resource by `id`, or by the exact `permission_name` and `view_menu_name` pair.

## Example Usage

```terraform
data "superset_permission" "dashboard_read" {
  permission_name = "can_read"
  view_menu_name  = "Dashboard"
}
```

## Schema

### Optional

- `id` (Number) Superset permission-view-menu identifier. Configure this, or configure `permission_name` with `view_menu_name`.
- `permission_name` (String) Permission name used for lookup, for example `can_read`.
- `view_menu_name` (String) View menu name used for lookup, for example `Dashboard`.
