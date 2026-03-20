---
page_title: "superset_role_permission Resource - superset"
subcategory: ""
description: |-
  Manages the full Superset permission set for one role.
---

# superset_role_permission (Resource)

Manages the full Superset permission set for one role.

This resource is authoritative for the role's permissions. Any permission-view-menu identifiers omitted from `permission_ids` are removed from the role.

## Example Usage

```terraform
data "superset_permission" "dashboard_read" {
  permission_name = "can_read"
  view_menu_name  = "Dashboard"
}

data "superset_permission" "log_read" {
  permission_name = "can_read"
  view_menu_name  = "Log"
}

resource "superset_role" "analyst" {
  name = "Analyst"
}

resource "superset_role_permission" "analyst" {
  role_id = superset_role.analyst.id
  permission_ids = [
    data.superset_permission.dashboard_read.id,
    data.superset_permission.log_read.id,
  ]
}
```

## Schema

### Required

- `permission_ids` (Set of Number) Authoritative set of permission-view-menu identifiers assigned to the role.
- `role_id` (Number) Superset role identifier whose permissions are managed by this resource.

### Read-Only

- `id` (Number) Terraform resource identifier. This matches `role_id`.
- `role_name` (String) Resolved Superset role name.
