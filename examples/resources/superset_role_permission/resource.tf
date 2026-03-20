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
