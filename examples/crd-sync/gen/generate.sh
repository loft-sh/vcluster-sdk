#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
SCRIPT_ROOT=${SCRIPT_DIR}/../../..

KUBE_CODEGEN_ROOT="${SCRIPT_ROOT}/vendor/k8s.io/code-generator"
source "${SCRIPT_ROOT}/vendor/k8s.io/code-generator/kube_codegen.sh"

kube::codegen::gen_helpers \
  --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.txt" \
  "examples/crd-sync"

echo "Generate crd ..."
go run gen/main.go >manifests/crds.yaml

echo "Update vendor"
go mod vendor
