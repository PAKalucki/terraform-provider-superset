resource "superset_database" "warehouse" {
  database_name  = "analytics"
  sqlalchemy_uri = "postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"
}

resource "superset_dataset" "events" {
  database_id           = superset_database.warehouse.id
  schema                = "analytics"
  table_name            = "events"
  main_dttm_col         = "created_at"
  filter_select_enabled = true
}

locals {
  datasource_uid = format("%d__table", superset_dataset.events.id)
}

resource "superset_chart" "events_table" {
  slice_name    = "Events table"
  datasource_id = superset_dataset.events.id
  viz_type      = "table"

  params = jsonencode({
    datasource = local.datasource_uid
    viz_type   = "table"
  })
}

resource "superset_dashboard" "operations" {
  dashboard_title = "Operations dashboard"
  slug            = "operations-dashboard"
  published       = true
  chart_ids       = [superset_chart.events_table.id]
}
