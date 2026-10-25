# Cilium Gateway — L4 Routes + DNS Migration: STATUS / HANDOFF

_Last updated: 2026-10-25 (post L4-DNS ship). Read top to bottom._

## TL;DR — where we are right now
All ingress/L4 runs through **Cilium Gateway** (`external` + `internal` in `kube-system`). L4 TCP/UDP routes are in place, co-located with each app, and **resolved to the gateway IPs via DNSEndpoints published by external-dns to the LAN resolvers**. Committed + pushed; Flux reconciled.

**Final DNS design (shipped in `563fdd050` + `8aa75bd65`):**
1. **L7 (HTTPRoutes)** — cloudflare-dns (public, `--source=gateway-httproute --gateway-name=external`) + omada + pihole (LAN). Route `target` annotations CNAME app names → `external./internal.rphilipsen.nl`; the gateway LB services (`cilium-gateway-external`/`-internal`, **Cilium-owned, not repo-managed**) carry `external-dns.kubernetes.io/hostname: external./internal.rphilipsen.nl` on their `service` source → A `172.16.20.30` / `172.16.20.31`.
2. **L4 (TCP/UDP routes)** — each route carries `external-dns.kubernetes.io/controller: none` (opt-out of every external-dns Gateway source, so no broken CNAME-to-tunnel records for L4). A records are plain `DNSEndpoint`s in `kubernetes/apps/network/external-dns-omada/app/dnsendpoint-l4.yaml`:
   - `l4-external`: `mc.rphilipsen.nl` A 172.16.20.30
   - `l4-internal`: `mc-rcon`, `mc-metrics`, `mqtt`, `postgres`, `qbittorrent`, `esphome-build`, `syslog` `.rphilipsen.nl` A 172.16.20.31
   - Published by the **pihole** instance (stable resolver, `crd` source) and the **omada** instance (best-effort; omada server not guaranteed up).
3. **cloudflare-dns dropped its `crd` source** — no more DNSEndpoint churn in the public zone, and the pre-existing `omada.rphilipsen.nl` A→192.168.0.2 400 (proxied private IP) loop is gone. `external.rphilipsen.nl` + `internal.rphilipsen.nl` CNAMEs in CF are now **manually managed** (the cloudflare-tunnel DNSEndpoints no longer feed cloudflare-dns; their TXT owner lines become orphaned, so sync won't delete them).

**Known, accepted duplication (user: "I'll clear the duplicate out later"):** the 5 ex-LB app services (`minecraft`, `vernemq`, `pooler`-via-`postgres-lb` history, `esphome-peer-link`, `syslog-lb`) still carry `external-dns.kubernetes.io/hostname` annotations, so the `service` source emits A records for `mc`/`mqtt`/`postgres`/`esphome-build`/`syslog` pointing at their **old LB VIPs** (.150/.102/.101/.103/.104) — which still work (same backend). For those 5 names the service-source record currently **wins** over the DNSEndpoint in the LAN resolvers; the 3 new names (mc-rcon, mc-metrics, qbittorrent) resolve to .31 cleanly. To finish the migration: remove the `external-dns.kubernetes.io/hostname` annotations from those 5 service manifests (grep `kubernetes/apps` for the key; keep it only on the two `kube-system/cilium/gateway/*.yaml` gateway services). `upsert-only` policies will then update (not delete) the LAN records to the gateway IPs. Note: `upsert-only` never deletes, so any leftover record is rewritten in place on the next cycle.

## Verification state (all green, 2026-10-25)
- All 9 TCPRoutes + both UDPRoutes `Programmed`; all 11 carry `controller: none` (bedrock included — intentionally no DNS).
- LAN dig via pi-hole (172.16.20.7, queried from a hostNetwork pod in kube-system — plain pod→LAN DNS queries time out on return route, that's expected): `mc-rcon`/`mc-metrics`/`qbittorrent` → .31; `mc`→.150, `mqtt`→.102, `postgres`→.101, `esphome-build`→.103, `syslog`→.104 (accepted duplication, see above); `grafana` → .30 (L7 path intact).
- `external-dns-pihole` + `external-dns-omada`: "All records are already up to date", no errors. Recurring benign warning on both: `Domain external.rphilipsen.nl contains conflicting record type candidates; discarding CNAME record` (service-source A .30 beats cloudflare-tunnel CNAME — desired, LAN skips the tunnel loopback).
- `cloudflare-dns`: "All records are already up to date", zero errors/400s in last 10m.

## external-dns v0.23.0 gotchas (learned the hard way)
- `--policy` must be set explicitly (validation is fatal without it).
- **No `--crd-source-namespaces` flag exists** (only `--crd-source-apiversion` / `--crd-source-kind`) — per-provider DNSEndpoint scoping is impossible; `--annotation-filter` is global (affects all sources) → the only clean scoping lever is the `controller: none` opt-out annotation.
- `controller: none` is honored by the gateway sources at the informer-index level (log: "Skipping ... controller value does not match, found: 'none'").
- The chart builds RBAC from the `sources` list — adding `crd` auto-grants DNSEndpoint list/watch.
- cloudflare-dns SA has **no** RBAC to list Service/Node/Pod/EndpointSlice → inmemory/dry-run pods emulating `--source=service` will fail; dry-runs must drop that source.
- Same name+type from two sources with different targets: one candidate wins (empirically the `service` source beat `crd`); no merge, no warning.

## Key facts
- Repo: `/home/ronald245/Documents/Source/home-ops`; Flux GitOps, branch `main`. Cilium 1.20.4, K8s v1.36.2, Talos 1.14.
- Gateways (`kube-system`): `external` = **172.16.20.30**, `internal` = **172.16.20.31**, both `Programmed`. LB services are Cilium-owned (durable, not repo-managed).
- external-dns instances (ns `network`, image v0.23.0): `cloudflare-dns` (public zone, `gateway-httproute` only), `external-dns-omada` (webhook `ghcr.io/ronaldphilipsen/external-dns-omada:v0.1.2`, sources httproute/tcproute/udproute/service/crd, upsert-only), `external-dns-pihole` (server 172.16.20.7:81, same sources, upsert-only).
- Tunnel: cloudflared in `cloudflare-tunnel` is **HTTPS-only** (no L4 ingress) — the reason L4 can't use the CNAME-to-tunnel shape.
- LB IP pool 172.16.20.0/24; `CiliumL2AnnouncementPolicy l2-policy` announces the LB IPs (LAN reachability).

## L4 routes (co-located, all `controller: none`)
| File | Routes |
|---|---|
| `kubernetes/apps/games/minecraft/app/gateway-l4.yaml` | mc 25565/tcp (ext), bedrock 19132/udp (ext, no DNS), mc-rcon 25575 (int), mc-metrics 25585 (int) |
| `kubernetes/apps/database/vernemq/app/gateway-l4.yaml` | mqtt 1883, mqtt 8883 (int) |
| `kubernetes/apps/database/cloudnative-pg/cluster/gateway-l4.yaml` | postgres 5432 (int) |
| `kubernetes/apps/media/qbittorrent/app/gateway-l4.yaml` | qbittorrent 50413 (int, tcp only) |
| `kubernetes/apps/home-automation/esphome/app/gateway-l4.yaml` | esphome-build 6052 + 6055 (int) |
| `kubernetes/apps/observability/opentelemetry/collector/gateway-l4.yaml` | syslog 5514/udp (int) |

Listeners: `external.yaml` (minecraft 25565/TCP, minecraft-bedrock 19132/UDP); `internal.yaml` (the other 9). Each `allowedRoutes.namespaces.from: All`.

## Open items
1. **User cleanup**: remove `external-dns.kubernetes.io/hostname` from the 5 ex-LB app services (see TL;DR); then re-dig the 5 names → gateway IPs.
2. **Manual CF cleanup** (console/API, unowned records): old LB A records `mc`→.150, `mqtt`→.102, `postgres`→.101, `esphome-build`→.103, `syslog`→.104 (dead private IPs); `mc` CNAME → `home.rphilipsen.nl` (kept by design: public mc name, see below); orphaned TXT owner lines for `external./internal.rphilipsen.nl`.
3. **Manual router work**: home-router 25565 forward re-pointed from 172.16.20.150 to 172.16.20.30 (public mc traffic).
4. **8883/mqtts gap (pre-existing, app-level)**: vernemq chart has no TLS — listener is Programmed but nothing listens; `openssl s_client -connect 172.16.20.31:8883` → handshake failure. Needs a TLS cert + vernemq config, separate task.
5. **L7 301-flap investigation (stale)**: 301 bursts on .30:443 reported pre-migration; L2 pin experiment (`l2-policy` pinned to k8s-neo-alpha) showed 0 bad / 270 reqs and was reverted; not reproducible since. If it recurs, re-run the pin test.

## Public `mc.rphilipsen.nl` — manual, by design
Public CF record is a pre-existing **unowned** CNAME → `home.rphilipsen.nl` (A 95.98.157.200, home box). Sync won't touch it (no ownership TXT). After the router-forward re-point (open item 3), 25565 lands on 172.16.20.30 → gateway → mc pods. Do NOT let external-dns claim this name (that's what burned us before: tunnel CNAME shape breaks L4).

## Files
- `kubernetes/apps/network/external-dns-omada/app/dnsendpoint-l4.yaml` — the L4 A records (l4-external / l4-internal).
- `kubernetes/apps/network/external-dns-omada/app/kustomization.yaml` — includes it.
- `kubernetes/apps/network/cloudflare-dns/app/helmrelease.yaml` — L7-only sources.
- `kubernetes/apps/network/external-dns-{omada,pihole}/app/helmrelease.yaml` — both with `crd` source.
- `docs/superpowers/specs/2026-10-24-cilium-gateway-migration-design.md` — design history (untracked, local only).

## Untracked (do not commit)
`.worktrees/`, `Security-audit.md`, `docs/superpowers/`.

## Env quirks
- In this environment the lowercase kubectl resource token for DNSEndpoint gets mangled in output (`dnseンドpoint`); work around with `R="dn""se""n""dpoint"; kubectl get "$R"`.
- Plain pod → LAN DNS (UDP 53) queries time out (return route); query from `hostNetwork: true` pods in `kube-system` (default ns is PSA baseline).
