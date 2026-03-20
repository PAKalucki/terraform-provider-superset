data "superset_permission" "dashboard_read" {
  permission_name = "can_read"
  view_menu_name  = "Dashboard"
}
