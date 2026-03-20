#!/usr/bin/env bash
set -euo pipefail

SUPERSET_ENDPOINT="${SUPERSET_ENDPOINT:-http://127.0.0.1:8088}"
SUPERSET_USERNAME="${SUPERSET_USERNAME:-admin}"
SUPERSET_PASSWORD="${SUPERSET_PASSWORD:-admin}"

health_url="${SUPERSET_ENDPOINT%/}/health"
login_url="${SUPERSET_ENDPOINT%/}/api/v1/security/login"
payload="$(printf '{"username":"%s","password":"%s","provider":"db"}' "${SUPERSET_USERNAME}" "${SUPERSET_PASSWORD}")"

for attempt in $(seq 1 90); do
  if curl --silent --show-error --fail "${health_url}" >/dev/null 2>&1 &&
    curl --silent --show-error --fail \
      -X POST "${login_url}" \
      -H "Content-Type: application/json" \
      -d "${payload}" >/dev/null 2>&1; then
    exit 0
  fi

  sleep 2
done

echo "Superset test environment did not become ready at ${SUPERSET_ENDPOINT}" >&2
exit 1
