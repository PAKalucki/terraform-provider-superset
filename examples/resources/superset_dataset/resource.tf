resource "superset_database" "warehouse" {
  database_name  = "analytics"
  sqlalchemy_uri = "postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"
}

resource "superset_dataset" "events" {
  database_id           = superset_database.warehouse.id
  schema                = "analytics"
  table_name            = "events"
  description           = "Warehouse events dataset"
  main_dttm_col         = "created_at"
  filter_select_enabled = true

  columns = [
    {
      column_name  = "id"
      verbose_name = "Event ID"
      filterable   = true
      groupby      = true
      is_active    = true
      type         = "INTEGER"
    },
    {
      column_name = "created_at"
      filterable  = true
      groupby     = true
      is_active   = true
      is_dttm     = true
      type        = "TIMESTAMP"
    }
  ]

  metrics = [
    {
      metric_name  = "event_count"
      expression   = "COUNT(*)"
      metric_type  = "count"
      verbose_name = "Event Count"
    }
  ]
}
