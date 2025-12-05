#!/bin/bash
# Copyright IBM Corp. 2022, 2025
# SPDX-License-Identifier: MPL-2.0

# Ensure that 'yq' is installed in the bin directory by running 'make yq'
YQ="$ROOT/bin/yq"
CSV_FILE="bundle/manifests/hcp-terraform-operator.clusterserviceversion.yaml"

##### Add 'com.redhat.openshift.versions' annotation.
echo "Add 'com.redhat.openshift.versions' annotation."
OPENSHIFT_VERSIONS="\"v4.12\""
{
    echo ""
    echo "  # OpenShift specific annotations"
    echo "  com.redhat.openshift.versions: $OPENSHIFT_VERSIONS"
} >> bundle/metadata/annotations.yaml

##### Add 'containerImage' annotation.
echo "Add 'containerImage' annotation."
IMAGE=$(yq '.spec.install.spec.deployments[] | select(.name == "hcp-terraform-operator-controller-manager") | .spec.template.spec.containers[] | select(.name == "manager") | .image' $CSV_FILE)
echo $IMAGE
yq -i ".metadata.annotations.containerImage = \"$IMAGE\"" $CSV_FILE
