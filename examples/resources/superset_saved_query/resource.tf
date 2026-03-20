resource "superset_database" "warehouse" {
  database_name  = "analytics"
  sqlalchemy_uri = "postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"
}

resource "superset_saved_query" "orders" {
  database_id = superset_database.warehouse.id
  label       = "Orders Preview"
  schema      = "analytics"
  sql         = "select * from orders limit 100"

  template_parameters = jsonencode({
    region = "emea"
  })

  extra_json = jsonencode({
    editor_width = 640
  })
}
