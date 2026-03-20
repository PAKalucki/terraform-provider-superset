data "superset_chart" "events_table" {
  datasource_id = superset_dataset.events.id
  slice_name    = "Events table"
}
