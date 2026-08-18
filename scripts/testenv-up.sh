#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/docker_compose/docker-compose.yaml"

docker compose -f "${COMPOSE_FILE}" up -d --wait
"${ROOT_DIR}/scripts/testenv-wait.sh"
