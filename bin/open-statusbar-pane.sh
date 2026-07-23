#!/bin/bash
set -euo pipefail

HERDR_BIN="${HERDR_BIN_PATH:-herdr}"
STATE_DIR="${HERDR_PLUGIN_STATE_DIR:-${TMPDIR:-/tmp}/herdr-usagebar}"
TAB_KEY="${HERDR_TAB_ID:-default}"
TAB_KEY="$(printf '%s' "$TAB_KEY" | tr ':/' '__')"
STATE_FILE="$STATE_DIR/statusbar-pane-$TAB_KEY"
mkdir -p "$STATE_DIR"

if [[ -s "$STATE_FILE" ]]; then
	pane_id="$(<"$STATE_FILE")"
	if "$HERDR_BIN" pane get "$pane_id" >/dev/null 2>&1; then
		echo "Herdr Usage Bar bar already open in $pane_id"
		exit 0
	fi
	rm -f "$STATE_FILE"
fi

response="$($HERDR_BIN plugin pane open \
	--plugin usagebar \
	--entrypoint statusbar \
	--placement split \
	--direction down \
	--no-focus)"
printf '%s\n' "$response"

pane_id="$(printf '%s' "$response" | sed -En 's/.*"pane_id"[[:space:]]*:[[:space:]]*"([^"]*)".*/\1/p')"
if [[ -z "$pane_id" ]]; then
	echo "Herdr Usage Bar bar opened, but Herdr did not return pane id; resize it manually." >&2
	exit 0
fi

printf '%s' "$pane_id" >"$STATE_FILE"
# plugin.pane.open returns before pane layout settles on some Herdr builds.
for _ in {1..20}; do
	if "$HERDR_BIN" pane get "$pane_id" >/dev/null 2>&1; then
		break
	fi
	sleep 0.1
done
# ponytail: Herdr plugin panes have no split-ratio field; one resize keeps this
# terminal-pane fallback thin. Remove if user prefers a full-height pane.
"$HERDR_BIN" pane resize --pane "$pane_id" --direction down --amount 0.42 >/dev/null 2>&1 || true
