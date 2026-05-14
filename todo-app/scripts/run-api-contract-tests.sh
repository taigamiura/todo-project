#!/bin/sh
set -eu

cleanup() {
  docker compose -f ../compose.mock.yml down >/dev/null 2>&1 || true
}

trap cleanup EXIT

docker compose -f ../compose.mock.yml up -d --wait bff-mock

ENABLE_CONTRACT_TESTS=1 BFF_BASE_URL=http://127.0.0.1:8080 npx vitest run src/app/api/**/*.contract.test.ts