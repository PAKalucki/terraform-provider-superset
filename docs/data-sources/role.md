---
page_title: "superset_role Data Source - superset"
subcategory: ""
description: |-
  Reads an existing Superset role.
---

# superset_role (Data Source)

Reads an existing Superset role by `id` or `name`.

The data source returns the current `user_ids`, `group_ids`, and `permission_ids` assigned to the role.

## Example Usage

```terraform
data "superset_role" "admin" {
  name = "Admin"
}
```

## Schema

### Optional

- `id` (Number) Superset role identifier. Configure this, or configure `name`.
- `name` (String) Role name used for lookup. Configure this, or configure `id`.

### Read-Only

- `group_ids` (Set of Number) Superset group identifiers currently assigned to the role.
- `permission_ids` (Set of Number) Superset permission-view-menu identifiers currently assigned to the role.
- `user_ids` (Set of Number) Superset user identifiers currently assigned to the role.
