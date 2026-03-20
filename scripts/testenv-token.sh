#!/usr/bin/env bash
set -euo pipefail

SUPERSET_ENDPOINT="${SUPERSET_ENDPOINT:-http://127.0.0.1:8088}"
SUPERSET_USERNAME="${SUPERSET_USERNAME:-admin}"
SUPERSET_PASSWORD="${SUPERSET_PASSWORD:-admin}"

curl --silent --show-error --fail \
  -X POST "${SUPERSET_ENDPOINT%/}/api/v1/security/login" \
  -H "Content-Type: application/json" \
  -d "$(printf '{"username":"%s","password":"%s","provider":"db"}' "${SUPERSET_USERNAME}" "${SUPERSET_PASSWORD}")"
