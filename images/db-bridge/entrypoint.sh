#!/bin/sh
# db-bridge entrypoint:
#  1. spiffe-helper: fetches the SVID for this pod's identity and renews it
#     in /spiffe (svid.crt/svid.key/bundle.crt).
#  2. waits for the SVID to land.
#  3. db-bridge: 127.0.0.1:5432 (plain Postgres) -> mTLS Postgres with the
#     SVID. New connections re-read the SVID files, so rotation needs no
#     restart.
set -eu

# 1. spiffe-helper in the background
spiffe-helper -config /etc/spiffe-helper/helper.conf &
HELPER_PID=$!
trap 'kill "$HELPER_PID" 2>/dev/null' TERM INT

# 2. wait for the SVID (up to ~60s)
i=0
while [ ! -s /spiffe/svid.crt ] || [ ! -s /spiffe/svid.key ]; do
  i=$((i + 1))
  if [ "$i" -ge 60 ]; then
    echo "db-bridge: SVID never appeared in /spiffe" >&2
    exit 1
  fi
  sleep 1
done

# 3. the bridge (exec: SIGTERM reaches it directly)
exec db-bridge
