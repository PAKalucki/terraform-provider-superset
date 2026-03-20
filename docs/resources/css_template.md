---
page_title: "superset_css_template Resource - superset"
subcategory: ""
description: |-
  Manages a Superset CSS template.
---

# superset_css_template (Resource)

Manages a Superset CSS template.

## Example Usage

```terraform
resource "superset_css_template" "branding" {
  template_name = "Branding"
  css = <<-CSS
    .dashboard {
      background: #f4efe6;
      color: #243447;
    }
  CSS
}
```

## Schema

### Required

- `css` (String) CSS text stored in the template.
- `template_name` (String) Superset CSS template name.

### Read-Only

- `id` (Number) Superset CSS template identifier.
