---
page_title: "superset_annotation_layer Resource - superset"
subcategory: ""
description: |-
  Manages a Superset annotation layer.
---

# superset_annotation_layer (Resource)

Manages a Superset annotation layer.

This resource manages the annotation layer container itself. It does not yet manage individual annotations within the layer.

## Example Usage

```terraform
resource "superset_annotation_layer" "deployments" {
  name        = "Deployments"
  description = "Release and deployment markers"
}
```

## Schema

### Required

- `name` (String) Superset annotation layer name.

### Optional

- `description` (String) Optional annotation layer description.

### Read-Only

- `id` (Number) Superset annotation layer identifier.

## Import

Import an annotation layer by its numeric Superset id.

```terraform
import {
  to = superset_annotation_layer.deployments
  id = "42"
}
```

```shell
terraform import superset_annotation_layer.deployments 42
```
