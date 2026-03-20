---
page_title: "superset_user Resource - superset"
subcategory: ""
description: |-
  Manages a Superset user.
---

# superset_user (Resource)

Manages a Superset user for Superset auth backends that expose user administration through the REST API. The local acceptance environment uses the built-in DB auth backend, which supports this resource.

`role_ids` is authoritative for the user's role assignments. `password` is required on create; when omitted later, the provider leaves the existing password unchanged.

## Example Usage

```terraform
data "superset_role" "alpha" {
  name = "Alpha"
}

resource "superset_user" "analyst" {
  username   = "analyst"
  first_name = "Analytics"
  last_name  = "User"
  email      = "analyst@example.com"
  password   = "ChangeMe123!"
  role_ids   = [data.superset_role.alpha.id]
}
```

## Schema

### Required

- `email` (String) Superset email address.
- `first_name` (String) Superset first name.
- `last_name` (String) Superset last name.
- `role_ids` (Set of Number) Authoritative set of Superset role identifiers assigned to the user.
- `username` (String) Superset username.

### Optional

- `active` (Boolean) Whether the Superset user is active. Defaults to `true`.
- `password` (String, Sensitive) Superset password. This is required on create. When omitted later, the provider leaves the existing password unchanged.

### Read-Only

- `id` (Number) Superset user identifier.

## Import

Import a user by its numeric Superset id.

```terraform
import {
  to = superset_user.analyst
  id = "42"
}
```

```shell
terraform import superset_user.analyst 42
```

Superset does not return user passwords through the API, so `password` is not restored by import. Omit `password` after import to leave it unchanged, or set a new value if you want Terraform to rotate it on the next apply.
