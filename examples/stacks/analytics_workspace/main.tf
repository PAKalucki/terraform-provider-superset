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

variable "warehouse_sqlalchemy_uri" {
  type        = string
  sensitive   = true
  description = "SQLAlchemy URI for the analytics warehouse."
}

resource "superset_database" "warehouse" {
  database_name  = "analytics"
  sqlalchemy_uri = var.warehouse_sqlalchemy_uri

  expose_in_sqllab = true

  extra = jsonencode({
    metadata_cache_timeout = {
      schema_cache_timeout = 600
    }
  })
}

resource "superset_dataset" "events" {
  database_id           = superset_database.warehouse.id
  schema                = "analytics"
  table_name            = "events"
  description           = "Curated events dataset"
  main_dttm_col         = "created_at"
  filter_select_enabled = true

  columns = [
    {
      column_name  = "id"
      verbose_name = "Event ID"
      groupby      = true
      is_active    = true
      type         = "INTEGER"
    },
    {
      column_name = "event_name"
      filterable  = true
      groupby     = true
      is_active   = true
      type        = "TEXT"
    },
    {
      column_name = "created_at"
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

locals {
  datasource_uid = format("%d__table", superset_dataset.events.id)
}

resource "superset_chart" "events_table" {
  slice_name    = "Events Table"
  datasource_id = superset_dataset.events.id
  viz_type      = "table"

  params = jsonencode({
    datasource = local.datasource_uid
    viz_type   = "table"
  })
}

resource "superset_css_template" "branding" {
  template_name = "Branding"
  css           = <<-CSS
    .dashboard {
      background: #f4efe6;
      color: #243447;
    }
  CSS
}

resource "superset_annotation_layer" "deployments" {
  name        = "Deployments"
  description = "Release markers for production deploys"
}

resource "superset_saved_query" "events_preview" {
  database_id = superset_database.warehouse.id
  label       = "Events Preview"
  schema      = "analytics"
  sql         = "select * from events order by created_at desc limit 100"

  template_parameters = jsonencode({
    region = "emea"
  })
}

resource "superset_dashboard" "operations" {
  dashboard_title = "Operations Overview"
  slug            = "operations-overview"
  css             = superset_css_template.branding.css
  published       = true
  chart_ids       = [superset_chart.events_table.id]
}
