#!/bin/sh
# db-bridge entrypoint: spiffe-helper (SVID lifecycle) + pgbouncer (DB bridge).
# The container exits if either process dies.
set -e

INI="${PGBOUNCER_INI:-/etc/db-bridge/pgbouncer.ini}"
CERT_DIR="${CERT_DIR:-/spiffe}"

# 1. spiffe-helper in the background: fetches the SVID for this pod's identity
#    and renews it. On every update it runs CMD (SIGHUP to pgbouncer); the
#    first run happens before pgbouncer exists and fails harmlessly.
spiffe-helper &
HELPER_PID=$!

# 2. Wait for the initial SVID before starting pgbouncer (pgbouncer needs the
#    client cert at startup).
i=0
while [ ! -s "$CERT_DIR/svid.crt" ] || [ ! -s "$CERT_DIR/svid.key" ]; do
  i=$((i + 1))
  if [ "$i" -ge 90 ]; then
    echo "ERROR: no SVID after 90s" >&2
    exit 1
  fi
  sleep 1
done

# 3. pgbouncer daemonized; it reads the client cert and the CNPG server CA.
pgbouncer -d "$INI"
PGB_PID=$(cat /tmp/pgbouncer.pid)

# 4. Stay alive while both run; exit (and let k8s restart the container) if
#    either dies. tail exits when the watched pid disappears.
tail --pid="$PGB_PID" -f /dev/null &
TAIL_PID=$!
wait "$HELPER_PID"
H=$?
kill "$TAIL_PID" 2>/dev/null || true
exit "$H"
