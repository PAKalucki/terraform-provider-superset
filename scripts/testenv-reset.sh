#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

docker compose -f "${ROOT_DIR}/docker_compose/docker-compose.yaml" down --volumes --remove-orphans
docker compose -f "${ROOT_DIR}/docker_compose/docker-compose.yaml" up -d --wait
"${ROOT_DIR}/scripts/testenv-wait.sh"
