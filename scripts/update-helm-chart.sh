#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


CHART_DIR="charts/terraform-cloud-operator"
CHART_FILE="Chart.yaml"
VALUES_FILE="values.yaml"
VERSION_FILE="version/VERSION"

# Update the 'Chart.yaml' file with a new version of the Operator image tag.
function update_chart_file {
  C_VERSION=`yq '.appVersion' $CHART_DIR/$CHART_FILE`
  C_CHART_VERSION=`yq '.version' $CHART_DIR/$CHART_FILE`

  if [[ $C_VERSION == $VERSION && $C_CHART_VERSION == $VERSION ]]; then
    echo "No changes in the $CHART_FILE file."
    return 0
  fi

  echo "Updating the $CHART_FILE file."

  yq \
    --inplace \
    '.appVersion = strenv(VERSION) | .version = strenv(VERSION)' $CHART_DIR/$CHART_FILE
}

function main {
  if [[ -z "${VERSION}" ]]; then
    echo "The environment variable VERSION is not set. Read value from ${VERSION_FILE}."
    export VERSION=`cat $VERSION_FILE`
  fi

  GIT_BRANCH=`git rev-parse --abbrev-ref HEAD | sed -e 's/^release\/v//'`

  if [[ "$VERSION" != "$GIT_BRANCH" ]]; then
    echo "The version in the git branch name '${GIT_BRANCH}' does not match with the release version '${VERSION}'."
    echo "Exiting!"
    exit 1
  fi

  echo "Version: ${VERSION}"

  update_chart_file
}

main
