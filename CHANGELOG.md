## Unreleased

## 0.0.6 (2026-03-22)

BUG FIXES:

- Preserve managed dashboard `chart_ids` order during state refresh so Terraform does not fail after apply when Superset returns dashboard charts in a different order.

## 0.0.5 (2026-03-22)

FEATURES:

- Add dashboard native filter support with `show_native_filters` and `native_filter_configuration` on the dashboard resource and data source.

BUG FIXES:

- Send a same-origin `Referer` header on CSRF-protected API requests so Superset 6.x accepts authenticated create/update/delete operations.
- Normalize dashboard metadata handling so layout updates preserve unmanaged native filters, and native filter updates preserve existing dashboard layout metadata.

## 0.0.2 (2026-03-21)

FEATURES:

- Add provider environment variable fallbacks for endpoint and authentication settings, including `SUPERSET_URL` as an alias for `endpoint`.

## 0.0.1 (2026-03-21)

FEATURES:

- Initial Apache Superset provider release with authenticated API client support for Superset 6.x.
- Managed resources for databases, datasets, charts, dashboards, roles, role permissions, users, saved queries, CSS templates, and annotation layers.
- Data sources for databases, datasets, charts, dashboards, roles, and permissions.
- Local docker-compose acceptance environment for Superset 6.0.0 with automated readiness checks.
- Import support for all managed resources.
