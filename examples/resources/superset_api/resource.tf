resource "superset_api" "publish_dashboard" {
  method = "PUT"
  path   = "/api/v1/dashboard/42"

  request_body = jsonencode({
    published = true
  })
}
