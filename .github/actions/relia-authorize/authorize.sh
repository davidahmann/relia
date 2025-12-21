#!/usr/bin/env bash
set -euo pipefail

RELIA_URL="${1:-}"
ACTION="${2:-}"
RESOURCE="${3:-}"
ENVNAME="${4:-}"
INTENT_JSON="${5:-{} }"
PLAN_DIGEST="${6:-}"
DIFF_URL="${7:-}"

if [[ -z "$RELIA_URL" || -z "$ACTION" || -z "$RESOURCE" || -z "$ENVNAME" ]]; then
  echo "usage: authorize.sh <relia_url> <action> <resource> <env> <intent_json> [plan_digest] [diff_url]" >&2
  exit 2
fi

if [[ -z "${ACTIONS_ID_TOKEN_REQUEST_URL:-}" || -z "${ACTIONS_ID_TOKEN_REQUEST_TOKEN:-}" ]]; then
  echo "missing GitHub OIDC environment variables; ensure id-token: write permissions" >&2
  exit 2
fi

OIDC_RESP=$(curl -sS -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
  "${ACTIONS_ID_TOKEN_REQUEST_URL}&audience=relia")

JWT=$(echo "$OIDC_RESP" | python - <<'PY'
import json
import sys
payload = json.load(sys.stdin)
value = payload.get("value", "")
if not value:
    raise SystemExit("missing id-token value")
print(value)
PY
)

REQ=$(python - <<PY
import json

req = {
  "action": "$ACTION",
  "resource": "$RESOURCE",
  "env": "$ENVNAME",
  "intent": json.loads(r'''$INTENT_JSON'''),
  "evidence": {
    "plan_digest": "$PLAN_DIGEST" if "$PLAN_DIGEST" else None,
    "diff_url": "$DIFF_URL" if "$DIFF_URL" else None,
  },
}
req["evidence"] = {k: v for k, v in req["evidence"].items() if v}
print(json.dumps(req))
PY
)

RESP=$(curl -sS -X POST "$RELIA_URL/v1/authorize" \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d "$REQ")

VERDICT=$(echo "$RESP" | python - <<'PY'
import json
import sys
data = json.load(sys.stdin)
print(data.get("verdict", ""))
PY
)

RECEIPT_ID=$(echo "$RESP" | python - <<'PY'
import json
import sys
data = json.load(sys.stdin)
print(data.get("receipt_id", ""))
PY
)

echo "receipt_id=$RECEIPT_ID" >> "$GITHUB_OUTPUT"

if [[ "$VERDICT" == "require_approval" ]]; then
  APPROVAL_ID=$(echo "$RESP" | python - <<'PY'
import json
import sys
data = json.load(sys.stdin)
approval = data.get("approval") or {}
print(approval.get("approval_id", ""))
PY
)

  if [[ -z "$APPROVAL_ID" ]]; then
    echo "missing approval_id in response" >&2
    exit 3
  fi

  POLL_URL="$RELIA_URL/v1/approvals/$APPROVAL_ID"
  echo "Approval required. Polling: $POLL_URL"

  for _ in {1..60}; do
    S=$(curl -sS "$POLL_URL" -H "Authorization: Bearer $JWT")
    STATUS=$(echo "$S" | python - <<'PY'
import json
import sys
data = json.load(sys.stdin)
print(data.get("status", ""))
PY
)

    if [[ "$STATUS" == "approved" ]]; then
      echo "Approved."
      break
    elif [[ "$STATUS" == "denied" ]]; then
      echo "Denied."
      exit 2
    fi
    sleep 5
  done

  RESP=$(curl -sS -X POST "$RELIA_URL/v1/authorize" \
    -H "Authorization: Bearer $JWT" \
    -H "Content-Type: application/json" \
    -d "$REQ")

  VERDICT=$(echo "$RESP" | python - <<'PY'
import json
import sys
data = json.load(sys.stdin)
print(data.get("verdict", ""))
PY
)

  RECEIPT_ID=$(echo "$RESP" | python - <<'PY'
import json
import sys
data = json.load(sys.stdin)
print(data.get("receipt_id", ""))
PY
)
  echo "receipt_id=$RECEIPT_ID" >> "$GITHUB_OUTPUT"
fi

if [[ "$VERDICT" == "allow" ]]; then
  AKID=$(echo "$RESP" | python - <<'PY'
import json
import sys
data = json.load(sys.stdin)
creds = data.get("aws_credentials") or {}
print(creds.get("access_key_id", ""))
PY
)
  SAK=$(echo "$RESP" | python - <<'PY'
import json
import sys
data = json.load(sys.stdin)
creds = data.get("aws_credentials") or {}
print(creds.get("secret_access_key", ""))
PY
)
  STK=$(echo "$RESP" | python - <<'PY'
import json
import sys
data = json.load(sys.stdin)
creds = data.get("aws_credentials") or {}
print(creds.get("session_token", ""))
PY
)

  echo "AWS_ACCESS_KEY_ID=$AKID" >> "$GITHUB_ENV"
  echo "AWS_SECRET_ACCESS_KEY=$SAK" >> "$GITHUB_ENV"
  echo "AWS_SESSION_TOKEN=$STK" >> "$GITHUB_ENV"
else
  echo "Authorization failed: $VERDICT"
  exit 3
fi
