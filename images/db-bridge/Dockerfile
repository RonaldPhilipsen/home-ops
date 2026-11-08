# db-bridge: SPIFFE-aware Postgres bridge sidecar.
#
# Runs spiffe-helper (fetches + rotates the pod's X.509 SVID from the SPIRE
# agent via the CSI-mounted Workload API socket) and the db-bridge binary
# (a small mTLS Postgres bridge: plain TCP on loopback, SVID client cert to
# CloudNativePG). One container so SVID rotation is absorbed in place — the
# bridge re-reads the SVID files on every new connection, no app-visible
# restart.
#
# The bridge config is identical for every app (same CNPG rw upstream; the
# SVID carries the identity), so nothing app-specific is mounted.
# Volumes expected in the pod:
#   spiffe   : csi driver csi.spiffe.io -> /run/spiffe (agent socket)
#   svid     : emptyDir                 -> /spiffe    (svid.crt/svid.key/bundle.crt)
#   cnpg-ca  : secret (server CA)       -> /cnpg-ca   (ca.crt)
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY bridge/ ./
RUN CGO_ENABLED=0 go build -o /out/db-bridge .

FROM debian:trixie-slim

# spiffe-helper v0.12.3 (static binary, vendored for reproducible builds)
COPY spiffe-helper /usr/local/bin/spiffe-helper
COPY --from=build /out/db-bridge /usr/local/bin/db-bridge
COPY entrypoint.sh /entrypoint.sh
COPY helper.conf /etc/spiffe-helper/helper.conf
RUN chmod 0755 /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
