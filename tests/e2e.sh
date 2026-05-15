#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for e2e tests" >&2
  exit 2
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required for e2e JSON checks" >&2
  exit 2
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

step() {
  printf '\n==> %s\n' "$*"
}

request() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local output="$4"
  local status

  if [ -n "$body" ]; then
    status="$(curl -sS -o "$output" -w "%{http_code}" \
      -X "$method" \
      -H "Content-Type: application/json" \
      --data "$body" \
      "$BASE_URL$path")"
  else
    status="$(curl -sS -o "$output" -w "%{http_code}" \
      -X "$method" \
      "$BASE_URL$path")"
  fi

  printf '%s' "$status"
}

expect_status() {
  local got="$1"
  local want="$2"
  local body_file="$3"

  if [ "$got" != "$want" ]; then
    echo "Expected HTTP $want, got HTTP $got" >&2
    echo "Response body:" >&2
    cat "$body_file" >&2
    exit 1
  fi
}

expect_not_status() {
  local got="$1"
  local unwanted="$2"
  local body_file="$3"

  if [ "$got" = "$unwanted" ]; then
    echo "Expected HTTP status different from $unwanted, got $got" >&2
    echo "Response body:" >&2
    cat "$body_file" >&2
    exit 1
  fi
}

json_value() {
  local file="$1"
  local expr="$2"

  python3 - "$file" "$expr" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

value = eval(sys.argv[2], {"__builtins__": {}}, {"bool": bool, "data": data, "len": len})
print(value)
PY
}

assert_json() {
  local file="$1"
  local expr="$2"
  local message="$3"

  python3 - "$file" "$expr" "$message" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

ok = bool(eval(sys.argv[2], {"__builtins__": {}}, {"bool": bool, "data": data, "len": len}))
if not ok:
    print(sys.argv[3], file=sys.stderr)
    print(json.dumps(data, ensure_ascii=False, indent=2), file=sys.stderr)
    sys.exit(1)
PY
}

step "Waiting for $BASE_URL/healthz"
health_body="$tmpdir/health.json"
for attempt in $(seq 1 60); do
  status="$(request GET /healthz "" "$health_body" || true)"
  if [ "$status" = "200" ]; then
    break
  fi

  if [ "$attempt" = "60" ]; then
    echo "Application did not become healthy; last status: $status" >&2
    cat "$health_body" >&2 || true
    exit 1
  fi

  sleep 2
done
expect_status "$status" 200 "$health_body"

step "POST /api/v1/parse"
parse_body="$tmpdir/parse.json"
status="$(request POST /api/v1/parse '{"path":"log.zip"}' "$parse_body")"
expect_status "$status" 200 "$parse_body"
log_id="$(json_value "$parse_body" "data['log_id']")"
assert_json "$parse_body" "data['log_id'] > 0" "parse response must contain positive log_id"

step "GET /api/v1/log/$log_id"
log_body="$tmpdir/log.json"
status="$(request GET "/api/v1/log/$log_id" "" "$log_body")"
expect_status "$status" 200 "$log_body"
assert_json "$log_body" "data['status'] == 'parsed'" "log status must be parsed"
assert_json "$log_body" "data['nodes_count'] > 0" "log nodes_count must be positive"
assert_json "$log_body" "data['ports_count'] > 0" "log ports_count must be positive"
assert_json "$log_body" "bool(data['uploaded_at'])" "log uploaded_at must be present"

step "GET /api/v1/topology/$log_id"
topology_body="$tmpdir/topology.json"
status="$(request GET "/api/v1/topology/$log_id" "" "$topology_body")"
expect_status "$status" 200 "$topology_body"
assert_json "$topology_body" "len(data['nodes']) > 0" "topology nodes must not be empty"
assert_json "$topology_body" "len(data['groups']) > 0" "topology groups must not be empty"
assert_json "$topology_body" "data['summary']['nodes_count'] > 0" "topology summary nodes_count must be positive"
assert_json "$topology_body" "data['summary']['ports_count'] > 0" "topology summary ports_count must be positive"
node_id="$(json_value "$topology_body" "data['nodes'][0]['id']")"

step "GET /api/v1/node/$node_id"
node_body="$tmpdir/node.json"
status="$(request GET "/api/v1/node/$node_id" "" "$node_body")"
expect_status "$status" 200 "$node_body"
assert_json "$node_body" "data['id'] == $node_id" "node id must match"
assert_json "$node_body" "bool(data['node_guid'])" "node_guid must be present"

step "GET /api/v1/port/$node_id"
ports_body="$tmpdir/ports.json"
status="$(request GET "/api/v1/port/$node_id" "" "$ports_body")"
expect_status "$status" 200 "$ports_body"
assert_json "$ports_body" "len(data['ports']) > 0" "node ports must not be empty"

step "Negative cases"
bad_body="$tmpdir/bad.json"

status="$(request GET /api/v1/log/0 "" "$bad_body")"
expect_status "$status" 400 "$bad_body"

status="$(request GET /api/v1/node/0 "" "$bad_body")"
expect_status "$status" 400 "$bad_body"

status="$(request GET /api/v1/topology/0 "" "$bad_body")"
expect_status "$status" 400 "$bad_body"

status="$(request POST /api/v1/parse '' "$bad_body")"
expect_status "$status" 400 "$bad_body"

status="$(request POST /api/v1/parse '{"path":' "$bad_body")"
expect_status "$status" 400 "$bad_body"

status="$(request POST /api/v1/parse '{"path":"missing.zip"}' "$bad_body")"
expect_status "$status" 404 "$bad_body"

step "E2E tests passed"
