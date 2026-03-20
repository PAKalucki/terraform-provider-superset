resource "superset_database" "warehouse" {
  database_name  = "analytics"
  sqlalchemy_uri = "postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"

  expose_in_sqllab = true

  extra = jsonencode({
    metadata_cache_timeout = {
      schema_cache_timeout = 600
    }
  })
}
