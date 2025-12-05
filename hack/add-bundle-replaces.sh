#!/bin/bash
# Copyright IBM Corp. 2022, 2025
# SPDX-License-Identifier: MPL-2.0

# Ensure that 'yq' is installed in the bin directory by running 'make yq'
YQ="$ROOT/bin/yq"
CSV_FILE="bundle/manifests/hcp-terraform-operator.clusterserviceversion.yaml"

echo "Set 'spec.replaces' field."
PREV_VERSION=`git describe --tags --abbrev=0 $(git rev-list --tags --skip=1 --max-count=1)`
echo "Previous version: $PREV_VERSION"

yq -i ".spec.replaces = \"hcp-terraform-operator.$PREV_VERSION\"" $CSV_FILE
yq -i "sort_keys(..)" $CSV_FILE
