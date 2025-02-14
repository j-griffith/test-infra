#!/bin/bash
# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

TESTINFRA_ROOT=$(git rev-parse --show-toplevel)

for output in gs://k8s-testgrid-canary/config gs://k8s-testgrid/config; do
  dir="$(dirname "${BASH_SOURCE}")"
  bazel run //testgrid/cmd/configurator -- \
    --yaml="${TESTINFRA_ROOT}/config/testgrids" \
    --prow-config="${TESTINFRA_ROOT}/prow/config.yaml" \
    --prow-job-config="${TESTINFRA_ROOT}/config/jobs/" \
    --output="${output}" \
    --oneshot \
    --world-readable
done
