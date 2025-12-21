#!/usr/bin/env bash
set -euo pipefail

go test -tags=e2e ./tests/e2e -run TestE2E -count=1
