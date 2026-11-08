# db-bridge: SPIFFE-aware Postgres bridge sidecar.
#
# Runs spiffe-helper (fetches + rotates the pod's X.509 SVID from the SPIRE
# agent via the CSI-mounted Workload API socket) and pgbouncer (presents the
# SVID to CloudNativePG, serves the app on loopback). One container so the
# SVID rotation can SIGHUP pgbouncer in place — no app-visible restart.
#
# The pgbouncer config is app-specific and is mounted per app from a
# ConfigMap (PGBOUNCER_INI). Volumes expected in the pod:
#   spiffe   : csi driver csi.spiffe.io -> /run/spiffe (agent.sock)
#   svid     : emptyDir                 -> /spiffe    (svid.crt/svid.key/bundle.crt)
#   cnpg-ca  : secret (server CA)       -> /cnpg-ca   (ca.crt)
FROM debian:trixie-slim

ARG SPIFFE_HELPER_VERSION=v0.12.3

RUN apt-get update \
 && apt-get install -y --no-install-recommends curl ca-certificates pgbouncer \
 && rm -rf /var/lib/apt/lists/* \
 && curl -fsSL "https://github.com/spiffe/spiffe-helper/releases/download/${SPIFFE_HELPER_VERSION}/spiffe-helper_${SPIFFE_HELPER_VERSION}_Linux-x86_64.tar.gz" \
      | tar xz -C /usr/local/bin \
 && chmod 0755 /usr/local/bin/spiffe-helper

COPY entrypoint.sh /entrypoint.sh
COPY helper.conf /etc/db-bridge/helper.conf
RUN chmod 0755 /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
