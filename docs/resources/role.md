---
page_title: "superset_role Resource - superset"
subcategory: ""
description: |-
  Manages a Superset role.
---

# superset_role (Resource)

Manages a Superset role.

Use this resource for the role object itself. Manage the role's permission set separately with `superset_role_permission`.

Deleting a role can affect any users or groups currently assigned to it. The provider emits a warning when it can detect those assignments before delete.

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

## Import

Import a role by its numeric Superset id.

```terraform
import {
  to = superset_role.analyst
  id = "42"
}
```

```shell
terraform import superset_role.analyst 42
```
