# Terraform Provider Superset

_This provider is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). See [Which SDK Should I Use?](https://developer.hashicorp.com/terraform/plugin/framework-benefits) in the Terraform documentation for additional information._

This repository contains an in-progress [Terraform](https://www.terraform.io) provider for Apache Superset, including:

- A resource and a data source (`internal/provider/`),
- Examples (`examples/`) and generated documentation (`docs/`),
- Miscellaneous meta files.

Currently implemented resources and data sources:

- `superset_database`
- `superset_dataset`
- `superset_chart`
- `superset_dashboard`
- `superset_role`
- `superset_role_permission`
- `superset_user`
- `superset_saved_query`
- `superset_css_template`
- `superset_annotation_layer`
- `superset_permission` data source

Tutorials for creating Terraform providers can be found on the [HashiCorp Developer](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework) platform. _Terraform Plugin Framework specific guides are titled accordingly._

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Example provider configuration:

```terraform
provider "superset" {
  endpoint = "http://127.0.0.1:8088"
  username = "admin"
  password = "admin"
}

resource "superset_database" "warehouse" {
  database_name  = "analytics"
  sqlalchemy_uri = "postgresql+psycopg2://analytics:analytics@warehouse:5432/analytics"

  extra = jsonencode({
    metadata_cache_timeout = {
      schema_cache_timeout = 600
    }
  })
}
```

Charts can be managed with `superset_chart` once a dataset exists. The chart stores `params` and optional `query_context` as JSON strings; prefer `jsonencode(...)` in Terraform configuration for both.

Dashboards can be managed with `superset_dashboard` using `chart_ids` for simple chart associations, or `position_json` when you need to control the exact Superset layout JSON.

Roles can be managed with `superset_role`, while `superset_role_permission` manages the full authoritative permission set for one role. The `superset_permission` data source resolves stable permission-view-menu identifiers for those assignments.

Users can be managed with `superset_user` when the Superset auth backend exposes user administration through the REST API, as in the local DB-auth acceptance environment.

Additional supported resources include `superset_saved_query`, `superset_css_template`, and `superset_annotation_layer`.

For the local acceptance environment, the sample warehouse Postgres service is available to Superset at `warehouse:5432`.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

## Acceptance Test Environment

The repository includes a local Superset 6.0.0 docker-compose environment for acceptance tests.

Default test environment settings:

- Superset API: `http://127.0.0.1:8088`
- Admin username: `admin`
- Admin password: `admin`
- Sample warehouse Postgres: `localhost:15432`
- Sample warehouse credentials: `analytics` / `analytics`

Useful commands:

```shell
make testenv-up
make testenv-token
make testacc
make testenv-down
make testenv-reset
```

Environment variables used by acceptance tests:

- `SUPERSET_ENDPOINT`
- `SUPERSET_USERNAME`
- `SUPERSET_PASSWORD`
- `SUPERSET_ACCESS_TOKEN` (optional alternative to username/password)

`make testacc` starts the local docker-compose environment if needed, waits for `/health` plus API login readiness, and then runs the Go acceptance suite with the default local credentials.
