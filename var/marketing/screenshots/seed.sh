#!/usr/bin/env bash
# Waits for startup builds of the fictional seed configs, and confirms the
# default admin login works. Usage: seed.sh <base-url> <tmpdir>
set -euo pipefail

base="${1:?base url}"
tmp="${2:?tmp dir}"
api="$base/api/StencilBox.clientapi.v1.StencilBoxApiService"
cookies="$tmp/admin.cookies"

rpc() {
	curl -fsS --max-time 10 -b "$cookies" -c "$cookies" \
		-X POST -H 'Content-Type: application/json' -d "$2" "$api/$1"
}

rm -f "$cookies"

echo "Logging in as admin"
rpc LoginWithUsernameAndPassword '{"username":"admin","password":"admin"}' >/dev/null

# Dismiss the default-password banner so marketing images look healthy.
rpc ChangePassword '{"currentPassword":"admin","newPassword":"marketing-demo"}' >/dev/null
# Re-login after password change (session may be cleared).
rm -f "$cookies"
rpc LoginWithUsernameAndPassword '{"username":"admin","password":"marketing-demo"}' >/dev/null

echo "Waiting for startup builds (homepage, sidebar, status)"
for path in homepage/index.html sidebar/index.html status/index.html; do
	ok=0
	for _ in $(seq 120); do
		if [ -f "$tmp/output/$path" ]; then
			ok=1
			break
		fi
		sleep 0.5
	done
	if [ "$ok" -ne 1 ]; then
		echo "Build output missing: $tmp/output/$path" >&2
		echo "--- service.log ---" >&2
		tail -n 80 "$tmp/service.log" >&2 || true
		exit 1
	fi
done

# Favicon fetches for external example.com URLs can finish after index.html exists;
# give the homepage a moment so icons settle before capture.
sleep 2

echo "Seed ready: $(rpc GetBuildConfigs '{}' | jq -r '(.buildConfigs // .build_configs // []) | length') build configs"
