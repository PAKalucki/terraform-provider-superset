# Reference
- https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider
- https://superset.apache.org/developer-docs/api/
- Superset version: 6.0.0+

# Work guidelines
- Follow test driven development practice when working
- Commit when finishing a task slice, use conventional commit message styles
- Prefer one self-contained commit per completed phase or milestone
- Use trivy fs . to scan for vulnerabilities
- Use golangci-lint run to lint new code for issues

# Tasks

## Phase 1: Provider Foundation
- [x] Rename scaffolding provider to superset provider (update type names, metadata)
- [x] Define provider schema with authentication attributes (endpoint, username, password, access_token)
- [x] Create Superset API client package (`internal/client/`)
- [x] Implement API client authentication (login flow, token management)
- [x] Implement API client base methods (GET, POST, PUT, DELETE with error handling)
- [x] Add provider configuration validation
- [x] Commit provider foundation changes with a conventional commit

## Phase 2: Acceptance Test Environment
- [x] Upgrade `docker_compose/` to Apache Superset 6.0.0 for local acceptance testing
- [x] Add deterministic bootstrap/setup for admin credentials and API login
- [x] Add health checks or readiness waiting for the Superset API
- [x] Add scripts or `make` targets to start, stop, and reset the test environment
- [x] Document the acceptance test workflow and required environment variables
- [x] Write provider acceptance tests that run against the docker-compose test environment
- [x] Run provider acceptance tests against the docker-compose test environment
- [x] Commit acceptance test environment changes with a conventional commit

## Phase 3: Database Connection Resource
- [x] Create `superset_database` resource schema
- [x] Implement CRUD operations for database connections
- [x] Add support for database connection parameters (SQLAlchemy URI, extra configs)
- [x] Create `superset_database` data source for reading existing connections
- [x] Write unit tests for database resource
- [x] Write acceptance tests for database resource against the docker-compose test environment
- [x] Run database resource acceptance tests against the docker-compose test environment
- [x] Document database resource with examples
- [x] Commit database resource changes with a conventional commit

## Phase 4: Dataset Resource
- [x] Create `superset_dataset` resource schema
- [x] Implement CRUD operations for datasets (SQL Lab tables)
- [x] Add support for dataset columns and metrics configuration
- [x] Create `superset_dataset` data source
- [x] Write unit tests for dataset resource
- [x] Write acceptance tests for dataset resource against the docker-compose test environment
- [x] Run dataset resource acceptance tests against the docker-compose test environment
- [x] Document dataset resource with examples
- [x] Commit dataset resource changes with a conventional commit

## Phase 5: Chart Resource
- [x] Create `superset_chart` resource schema
- [x] Implement CRUD operations for charts
- [x] Add support for chart parameters (viz_type, query context, etc.)
- [x] Create `superset_chart` data source
- [x] Write unit tests for chart resource
- [x] Write acceptance tests for chart resource against the docker-compose test environment
- [x] Run chart resource acceptance tests against the docker-compose test environment
- [x] Document chart resource with examples
- [x] Commit chart resource changes with a conventional commit

## Phase 6: Dashboard Resource
- [ ] Create `superset_dashboard` resource schema
- [ ] Implement CRUD operations for dashboards
- [ ] Add support for dashboard layout and chart positions
- [ ] Implement dashboard-chart associations
- [ ] Create `superset_dashboard` data source
- [ ] Write unit tests for dashboard resource
- [ ] Write acceptance tests for dashboard resource against the docker-compose test environment
- [ ] Run dashboard resource acceptance tests against the docker-compose test environment
- [ ] Document dashboard resource with examples
- [ ] Commit dashboard resource changes with a conventional commit

## Phase 7: Role & Permission Resources
- [ ] Create `superset_role` resource schema
- [ ] Implement CRUD operations for roles
- [ ] Create `superset_role_permission` resource for role-permission assignments
- [ ] Create data sources for roles and permissions
- [ ] Write unit tests for role and permission resources
- [ ] Write acceptance tests for role and permission resources against the docker-compose test environment
- [ ] Run role and permission acceptance tests against the docker-compose test environment
- [ ] Document role and permission resources
- [ ] Commit role and permission resource changes with a conventional commit

## Phase 8: User Resource (Optional - depends on Superset auth backend)
- [ ] Evaluate user management API availability in Superset 6.0.0
- [ ] Create `superset_user` resource schema if supported
- [ ] Implement CRUD operations for users
- [ ] Add support for user-role assignments
- [ ] Write unit tests for user resource
- [ ] Write acceptance tests for user resource against the docker-compose test environment
- [ ] Run user resource acceptance tests against the docker-compose test environment
- [ ] Document user resource
- [ ] Commit user resource changes with a conventional commit

## Phase 9: Additional Resources
- [ ] Create `superset_saved_query` resource
- [ ] Create `superset_css_template` resource
- [ ] Create `superset_annotation_layer` resource
- [ ] Write unit tests for additional resources
- [ ] Write acceptance tests for additional resources against the docker-compose test environment
- [ ] Run additional resource acceptance tests against the docker-compose test environment
- [ ] Document additional resources
- [ ] Commit additional resource changes with a conventional commit

## Phase 10: Import Support
- [ ] Add import functionality to all resources
- [ ] Document import procedures for each resource
- [ ] Write import acceptance tests against the docker-compose test environment
- [ ] Run import acceptance tests against the docker-compose test environment
- [ ] Commit import support changes with a conventional commit

## Phase 11: Final Polish
- [ ] Remove all scaffolding/example code and files
- [ ] Complete provider documentation (index.md)
- [ ] Add comprehensive usage examples in examples/ directory
- [ ] Run full unit and acceptance test suite against the docker-compose test environment and fix issues
- [ ] Run trivy security scan and fix vulnerabilities
- [ ] Run golangci-lint and fix linting issues
- [ ] Update CHANGELOG.md
- [ ] Commit final polish changes with a conventional commit
- [ ] Tag initial release version
