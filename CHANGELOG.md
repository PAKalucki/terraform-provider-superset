## Unreleased

BUG FIXES:

- Send a same-origin `Referer` header on CSRF-protected API requests so Superset 6.x accepts authenticated create/update/delete operations.

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
