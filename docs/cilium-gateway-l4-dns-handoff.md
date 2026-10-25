# Cilium Gateway — L4 Routes + DNS Migration: STATUS / HANDOFF

_Last updated: 2026-10-25 (session shutdown handoff). Read top to bottom._

## TL;DR — where we are right now
Migrating all ingress/L4 through **Cilium Gateway** (`external`+`internal` in `kube-system`) replacing agentgateway (already done) **and** the per-service LoadBalancer services. L4 TCP/UDP routes are in place and co-located with each app.

**STATUS: core edits DONE (not yet committed).** (1) 11 L4 routes now carry `external-dns.alpha.kubernetes.io/hostname` (bedrock = opt-out, no DNS). (2) 5 LB services converted to ClusterIP + their `external-dns.kubernetes.io/hostname`/`lbipam` annotations removed; `postgres-lb` DELETED (pooler ClusterIP already exists). All 11 routes pass server dry-run. Remaining: commit+push, then post-reconcile verify (external-dns logs, records -> .30/.31, no record conflicts, svc types).
Annotation key used: `external-dns.alpha.kubernetes.io/hostname` (matches v0.13.1 docs; deployed external-dns is v0.23.0; gh/raw blocked in this env so could not re-verify the v0.23.0 key against source. If a record does not appear, check external-dns logs for the exact key it expects.)
1. Map the **internal L4 routes** (and external `minecraft`) to DNS names via the external-dns Gateway API `hostname` annotation → records target the gateway IP. **Skip bedrock.**
2. Turn the per-service **LoadBalancer** services into **ClusterIP** — *only where no other service is already defined* for that backend.
3. **OPEN QUESTION (must verify):** is `external-dns.alpha.kubernetes.io/hostname` still the right annotation in the **latest** external-dns? (v0.13.1 docs say yes for TCP/UDP routes; the deployed version may be newer and may use a non-alpha key.)
4. Commit + push when done.

**Because of the 30-min shutdown: the work below may be PARTIALLY applied. Check `git status` to see what's staged/committed before continuing.**

---

## Objective
- Single Cilium Gateway for all ingress/egress. L4 (TCP/UDP) via `TCPRoute`/`UDPRoute`. Remove per-service `LoadBalancer` services; DNS for those services now points at the **gateway IP** via external-dns (Gateway API source), not the individual LB VIPs.

## Key facts (verified against the live cluster)
- Repo: `/home/ronald245/Documents/Source/home-ops`; Flux GitOps, branch `main`. Cilium 1.20.4, K8s v1.37.
- **Gateways** (`kube-system`): `external` = **172.16.20.30**, `internal` = **172.16.20.31**. Both `Programmed`. L4 listeners already added (see "Listeners").
- `cilium-gateway-external`/`cilium-gateway-internal` LB Services are **Cilium-owned** (ownerRef→Gateways), durable, NOT repo-managed.
- **TCPRoute + UDPRoute = `gateway.networking.k8s.io/v1`.**
- **external-dns instances + sources:**
  - `cloudflare-dns` (public): sources `crd` + `gateway-httproute`, flag `--gateway-name=external` → **only** creates records for routes on the **external** gateway.
  - `external-dns-omada` (LAN): sources `gateway-httproute` + `service` + `crd`, no `--gateway-name` → all gateways.
  - `external-dns-pihole` (LAN): sources `gateway-httproute` + `service`, no `--gateway-name` → all gateways.
  - ⇒ **Internal** routes get LAN DNS (omada/pihole) → `.31`. **External** routes get public (cloudflare) + LAN → `.30`.
- **DNS mechanism for TCP/UDP routes** (per external-dns Gateway API tutorial): TCP/UDP route specs have no hostnames; the hostname is supplied by the annotation. v0.13.1 = `external-dns.alpha.kubernetes.io/hostname`. Target = parent Gateway address. (VERIFY latest key — see Open Question.)
- The `gateway-httproute` source reads Gateways + HTTPRoute + TLSRoute + **TCPRoute + UDPRoute** (RBAC in tutorial lists tcproutes/udproutes).

## Backend services + intended DNS names
| Route (gateway / section) | Port/Proto | Backend svc (ns) | Intended FQDN | Notes |
|---|---|---|---|---|
| minecraft (external) | 25565/tcp | `minecraft` (games) | `mc.rphilipsen.nl` | |
| minecraft-bedrock (external) | 19132/udp | `minecraft` (games) | — **SKIP** | user: skip for now |
| minecraft-rcon (internal) | 25575/tcp | `minecraft` (games) | `mc-rcon.rphilipsen.nl` | new name (no prior DNS) |
| minecraft-metrics (internal) | 25585/tcp | `minecraft` (games) | `mc-metrics.rphilipsen.nl` | new name (no prior DNS) |
| vernemq (internal) | 1883/tcp | `vernemq` (database) | `mqtt.rphilipsen.nl` | existing name |
| vernemq-tls (internal) | 8883/tcp | `vernemq` (database) | `mqtt.rphilipsen.nl` | same host as 1883 |
| postgres (internal) | 5432/tcp | `pooler` (database) | `postgres.rphilipsen.nl` | existing name |
| esphome-peer-link (internal) | 6052/tcp | `esphome-peer-link` (home-automation) | `esphome-build.rphilipsen.nl` | existing name |
| esphome-peer-link-pairing (internal) | 6055/tcp | `esphome-peer-link` (home-automation) | `esphome-build.rphilipsen.nl` | same host |
| qbittorrent (internal) | 50413/tcp | `qbittorrent` (media) | `qbittorrent.rphilipsen.nl` | new name (no prior DNS); **tcp only, no udp on svc** |
| syslog (internal) | 5514/udp | `syslog-lb` (observability) | `syslog.rphilipsen.nl` | existing name |

## L4 listeners (already added, on the Gateways in kube-system)
- `external.yaml`: `minecraft` 25565/TCP, `minecraft-bedrock` 19132/UDP (each `allowedRoutes.namespaces.from: All`).
- `internal.yaml`: `minecraft-rcon` 25575/TCP, `minecraft-metrics` 25585/TCP, `vernemq` 1883/TCP, `vernemq-tls` 8883/TCP, `postgres` 5432/TCP, `esphome-peer-link` 6052/TCP, `esphome-peer-link-pairing` 6055/TCP, `qbittorrent` 50413/TCP, `syslog` 5514/UDP (each `allowedRoutes.namespaces.from: All`).

## L4 route files (co-located with apps; created this session)
Currently each has `annotations: external-dns.kubernetes.io/controller: none` (a STOPGAP opt-out from the previous instruction). **These need to be replaced** with the Gateway API `hostname` annotation (the active task):
- `kubernetes/apps/games/minecraft/app/gateway-l4.yaml` (4 routes)
- `kubernetes/apps/database/vernemq/app/gateway-l4.yaml` (2)
- `kubernetes/apps/database/cloudnative-pg/cluster/gateway-l4.yaml` (1)
- `kubernetes/apps/media/qbittorrent/app/gateway-l4.yaml` (1)
- `kubernetes/apps/home-automation/esphome/app/gateway-l4.yaml` (2)
- `kubernetes/apps/observability/opentelemetry/collector/gateway-l4.yaml` (1)
Each is wired into its app `kustomization.yaml` (`- ./gateway-l4.yaml`).
Note: qbittorrent app kustomization also references `../../media-nfs.yaml` (pre-existing; isolated `kubectl kustomize` of that app dir fails on the load-restrictor — build from `media/` root or use `kustomize build --load-restrictor=LoadRestrictionsNone`).

## LoadBalancer services to convert → ClusterIP
Each currently carries `external-dns.kubernetes.io/hostname` (via the `service` source). **When converting to ClusterIP, REMOVE the hostname annotation** (else the `service` source creates a record pointing at a non-routable ClusterIP). Also check the "no other service defined" condition per backend:
- `games/minecraft/app/helmrelease.yaml` — LB svc `minecraft`, line ~105-107 (`mc.rphilipsen.nl`).
- `database/vernemq/app/helmrelease.yaml` — LB svc `vernemq`, line ~102-104 (`mqtt.rphilipsen.nl`).
- `database/cloudnative-pg/cluster/service.yaml` — LB svc `postgres-lb` (`postgres.rphilipsen.nl`, `lbipam.cilium.io/ips: 172.16.20.101`). **A `pooler` ClusterIP service already exists** (route already targets `pooler`) → likely just **delete** `postgres-lb`, don't create a new ClusterIP.
- `home-automation/esphome/app/helmrelease.yaml` — LB svc `esphome-peer-link`, line ~71-73 (`esphome-build.rphilipsen.nl`). A separate `esphome-app` ClusterIP exists (different purpose/ports) → convert `esphome-peer-link` to ClusterIP.
- `observability/opentelemetry/collector/syslog-lb.yaml` — LB svc `syslog-lb` (`syslog.rphilipsen.nl`, `lbipam.cilium.io/ips: 172.16.20.104`) → convert to ClusterIP.
- `media/qbittorrent` — **verify** its service type in `app/helmrelease.yaml` (was not in the hostname grep; check whether it even has an LB svc / `lbipam` IP before converting).

"where there's no other service defined" = if a ClusterIP for the same backend already exists, don't add a second one (remove the LB svc / point the route at the existing ClusterIP).

## Decisions (this session)
- Routes co-located with apps (not central) per user. Listeners stay on Gateways (kube-system). Cross-ns `parentRef` works because Gateways allow routes from `All` namespaces.
- DNS for L4 services moves from per-LB-VIP to the gateway IP (via route hostname annotation).
- qbittorrent: **TCP only** (its service has no UDP port).
- rcon + metrics exposed on **internal** gateway only (LAN), with new subdomains.
- Skip bedrock DNS.

## OPEN QUESTION (resolve before finalizing annotation)
- Is `external-dns.alpha.kubernetes.io/hostname` still valid in the **deployed/latest** external-dns, or is there a GA (non-`alpha`) key now? Check the deployed image tag (`kubectl -n <ns> get deploy/<external-dns> -o jsonpath='{.spec.template.spec.containers[0].image}'` for cloudflare/omada/pihole) and the matching docs.
  - v0.13.1 (fetched): TCP/UDP routes → `external-dns.alpha.kubernetes.io/hostname` (annotation), RBAC includes tcproutes/udproutes.
  - If a newer GA key exists, use it. If unsure, keep `external-dns.alpha.kubernetes.io/hostname` (widely supported) and confirm via external-dns logs after apply.

## Verification (run after changes)
1. `kubectl kustomize <each app dir>` (or `kustomize build --load-restrictor=LoadRestrictionsNone` for qbittorrent) → routes present with the hostname annotation.
2. `kubectl apply --dry-run=server -f <each gateway-l4.yaml>` → schema OK.
3. After push + Flux reconcile: `kubectl get tcproute,udproute -A` → `Programmed`; `kubectl get svc -A | grep LoadBalancer` → the 5 converted ones now `ClusterIP`.
4. Check external-dns logs (cloudflare/omada/pihole) for the new records (`mc.rphilipsen.nl→.30`, `mqtt/postgres/esphome-build/syslog/mc-rcon/mc-metrics/qbittorrent .rphilipsen.nl→.31`) and **no conflicts**.
5. `dig`/`nslookup` a couple of names to confirm public vs LAN resolution.

## Remaining (core edits are DONE + validated)
- [ ] Verify latest external-dns hostname annotation key (Open Question).
- [ ] Replace `controller: none` on the 6 `gateway-l4.yaml` files with the `hostname` annotation (per table above; bedrock → keep opt-out/no DNS).
- [ ] Convert the 5 LB services → ClusterIP (+ remove their hostname annotations); handle `postgres-lb` (delete, `pooler` exists) and verify qbittorrent svc first.
- [ ] Commit + push (message e.g. "map L4 routes to DNS via gateway; LB→ClusterIP").
- [ ] Reconcile + verify (see Verification).

## Untracked (do not accidentally commit unrelated files)
`.worktrees/`, `Security-audit.md`, `docs/` (this handoff is in `docs/`).
