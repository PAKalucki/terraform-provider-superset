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
