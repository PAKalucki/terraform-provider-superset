---
page_title: "superset_role Resource - superset"
subcategory: ""
description: |-
  Manages a Superset role.
---

# superset_role (Resource)

Manages a Superset role.

Use this resource for the role object itself. Manage the role's permission set separately with `superset_role_permission`.

## Example Usage

```terraform
resource "superset_role" "analyst" {
  name = "Analyst"
}
```

## Schema

### Required

- `name` (String) Role name in Superset.

### Read-Only

- `id` (Number) Superset role identifier.
