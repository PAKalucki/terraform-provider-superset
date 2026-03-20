provider "superset" {
  endpoint = var.superset_endpoint
  username = var.superset_username
  password = var.superset_password
}

variable "superset_endpoint" {
  type        = string
  description = "Superset API endpoint."
}

variable "superset_username" {
  type        = string
  description = "Superset username."
}

variable "superset_password" {
  type        = string
  sensitive   = true
  description = "Superset password."
}

data "superset_permission" "dashboard_read" {
  permission_name = "can_read"
  view_menu_name  = "Dashboard"
}

data "superset_permission" "chart_read" {
  permission_name = "can_read"
  view_menu_name  = "Chart"
}

resource "superset_role" "analyst" {
  name = "Analytics Analyst"
}

resource "superset_role_permission" "analyst" {
  role_id = superset_role.analyst.id
  permission_ids = [
    data.superset_permission.dashboard_read.id,
    data.superset_permission.chart_read.id,
  ]
}

resource "superset_user" "analyst" {
  username   = "analyst"
  first_name = "Analytics"
  last_name  = "User"
  email      = "analyst@example.com"
  password   = "ChangeMe123!"
  role_ids   = [superset_role.analyst.id]
}
