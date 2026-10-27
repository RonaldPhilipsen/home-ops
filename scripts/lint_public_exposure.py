#!/usr/bin/env python3
"""Enforce the `expose: public` label on every route bound to the external gateway.

The `external` gateway (kubernetes/apps/kube-system/cilium/gateway/external.yaml)
exposes its https/minecraft listeners to the internet with `allowedRoutes.from: All`,
so any route binding to it is publicly reachable. Binding to it must therefore be an
explicit, labeled decision: every such route carries the label `expose: public`.

This script fails when:
  - a route binds to the external gateway without the `expose: public` label, or
  - a route carries `expose: public` but does not bind to the external gateway (stale label).

Route definitions are checked in:
  - standalone Gateway API resources (HTTPRoute/TCPRoute/UDPRoute/GRPCRoute)
  - HelmRelease values: app-template `route.<name>` and single-route `httpRoute` charts
  - Grafana operator CRs: `spec.httpRoute`
"""

import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    sys.exit("pyyaml is required: pip install pyyaml")

EXTERNAL_GATEWAY = "external"
EXPOSE_LABEL = "expose"
PUBLIC_VALUE = "public"
ROOT = Path(__file__).resolve().parent.parent / "kubernetes"
ROUTE_KINDS = {"HTTPRoute", "TCPRoute", "UDPRoute", "GRPCRoute"}


def iter_route_definitions(doc):
    """Yield (location, parent_refs, labels) for every route definition in a manifest."""
    kind = doc.get("kind")

    if kind in ROUTE_KINDS:
        spec = doc.get("spec") or {}
        meta = doc.get("metadata") or {}
        yield "spec", spec.get("parentRefs") or [], meta.get("labels")
        return

    if kind == "HelmRelease":
        values = (doc.get("spec") or {}).get("values") or {}

        routes = values.get("route")
        if isinstance(routes, dict):
            for name, route in routes.items():
                if isinstance(route, dict):
                    yield f"values.route.{name}", route.get("parentRefs") or [], route.get("labels")

        http_route = values.get("httpRoute")
        if isinstance(http_route, dict):
            if isinstance(http_route.get("parentRefs"), list):
                yield "values.httpRoute", http_route["parentRefs"], http_route.get("labels")
            elif isinstance(http_route.get("spec"), dict):
                spec = http_route["spec"]
                meta = http_route.get("metadata") or {}
                yield "values.httpRoute.spec", spec.get("parentRefs") or [], meta.get("labels")
        return

    if kind == "Grafana":
        http_route = (doc.get("spec") or {}).get("httpRoute")
        if isinstance(http_route, dict) and isinstance(http_route.get("spec"), dict):
            meta = http_route.get("metadata") or {}
            yield "spec.httpRoute.spec", http_route["spec"].get("parentRefs") or [], meta.get("labels")
            return


def is_external(parent_refs):
    return any(isinstance(ref, dict) and ref.get("name") == EXTERNAL_GATEWAY for ref in parent_refs)


def is_public(labels):
    return isinstance(labels, dict) and labels.get(EXPOSE_LABEL) == PUBLIC_VALUE


def main():
    violations = []
    checked_routes = 0

    for path in sorted(ROOT.rglob("*.yaml")):
        try:
            docs = list(yaml.safe_load_all(path.read_text()))
        except yaml.YAMLError as exc:
            print(f"warning: skipping {path.relative_to(ROOT.parent)}: {exc}", file=sys.stderr)
            continue

        for idx, doc in enumerate(docs):
            if not isinstance(doc, dict):
                continue
            name = (doc.get("metadata") or {}).get("name", f"doc-{idx + 1}")
            for location, parent_refs, labels in iter_route_definitions(doc):
                checked_routes += 1
                external = is_external(parent_refs)
                public = is_public(labels)
                where = f"{path.relative_to(ROOT.parent)}: {doc.get('kind')}/{name} ({location})"
                if external and not public:
                    violations.append(f"{where} binds to the '{EXTERNAL_GATEWAY}' gateway but lacks the '{EXPOSE_LABEL}: {PUBLIC_VALUE}' label")
                elif not external and public:
                    violations.append(f"{where} carries '{EXPOSE_LABEL}: {PUBLIC_VALUE}' but does not bind to the '{EXTERNAL_GATEWAY}' gateway (stale label)")

    if violations:
        print(f"FAIL: {len(violations)} public-exposure label violation(s):", file=sys.stderr)
        for violation in violations:
            print(f"  - {violation}", file=sys.stderr)
        sys.exit(1)

    print(f"OK: {checked_routes} route definition(s) checked, all public exposure is labeled.")


if __name__ == "__main__":
    main()
