#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
CONFIG_FILE="${ROOT_DIR}/talos/talstomize.yaml"
OUTPUT_DIR="${ROOT_DIR}/talos/clusterconfig"

if [[ $# -lt 1 ]]; then
    printf 'Usage: %s <node-name> [talosctl apply-config flags]\n' "$0" >&2
    exit 2
fi

node_name="$1"
shift

node_ip="$(yq -r ".nodes[\"${node_name}\"].ip // \"\"" "${CONFIG_FILE}")"
if [[ -z "${node_ip}" ]]; then
    printf 'Unknown node: %s\n' "${node_name}" >&2
    exit 2
fi

talstomize build talos --output "${OUTPUT_DIR}"

apply_config="${OUTPUT_DIR}/${node_name}.yaml"
if [[ ! -f "${apply_config}" ]]; then
    printf 'Generated config not found: %s\n' "${apply_config}" >&2
    exit 1
fi

tmp_config="$(mktemp)"
trap 'rm -f "${tmp_config}"' EXIT

yq 'select(documentIndex == 0 or .kind == "KubeAPIServerCAConfig")' "${apply_config}" > "${tmp_config}"

talosctl --talosconfig "${OUTPUT_DIR}/talosconfig" apply-config \
    --nodes "${node_ip}" \
    --file "${tmp_config}" \
    "$@"
