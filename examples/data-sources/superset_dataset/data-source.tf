data "superset_dataset" "events" {
  database_id = superset_database.warehouse.id
  schema      = "analytics"
  table_name  = "events"
}
