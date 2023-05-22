#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


CHART_DIR="charts/terraform-cloud-operator"
CHART_FILE="Chart.yaml"
CHART_VERSION_FILE="version/HELM_CHART_VERSION"
VALUES_FILE="values.yaml"
VERSION_FILE="version/VERSION"

# Update the 'Chart.yaml' file with a new version of the Operator image tag.
function update_chart_file {
  C_VERSION=`yq '.appVersion' $CHART_DIR/$CHART_FILE`
  C_CHART_VERSION=`yq '.version' $CHART_DIR/$CHART_FILE`

  if [[ $C_VERSION == $VERSION && $C_CHART_VERSION == $CHART_VERSION ]]; then
    echo "No changes in the $CHART_FILE file."
    return 0
  fi

  echo "Updating the $CHART_FILE file."

  yq \
    --inplace \
    '.appVersion = strenv(VERSION) | .version = strenv(CHART_VERSION)' $CHART_DIR/$CHART_FILE
}

# Update the 'values.yaml' file with a new version of the Operator image tag.
function update_values_file {
  C_VERSION=`yq '.operator.image.tag' $CHART_DIR/$VALUES_FILE`

  if [[ $C_VERSION == $VERSION ]]; then
    echo "No changes in the $VALUES_FILE file."
    return 0
  fi

  echo "Updating the $VALUES_FILE file."

  diff \
    -U0 \
    -w \
    --ignore-blank-lines \
    $CHART_DIR/$VALUES_FILE <(yq '.operator.image.tag = strenv(VERSION)' $CHART_DIR/$VALUES_FILE) > $CHART_DIR/$VALUES_FILE.diff

  patch --silent $CHART_DIR/$VALUES_FILE < $CHART_DIR/$VALUES_FILE.diff
  rm $CHART_DIR/$VALUES_FILE.diff
}

function main {
  if [[ -z "${VERSION}" ]]; then
    echo "The environment variable VERSION is not set. Read value from ${VERSION_FILE}."
    export VERSION=`cat $VERSION_FILE`
  fi

  if [[ -z "${CHART_VERSION}" ]]; then
    echo "The environment variable CHART_VERSION is not set. Read value from ${CHART_VERSION_FILE}."
    export CHART_VERSION=`cat $CHART_VERSION_FILE`
  fi

  GIT_BRANCH=`git rev-parse --abbrev-ref HEAD | sed -e 's/^release\/v//'`

  if [[ "$VERSION" != "$GIT_BRANCH" ]]; then
    echo "The version in the git branch name '${GIT_BRANCH}' does not match with the release version '${VERSION}'."
    echo "Exiting!"
    exit 1
  fi

  echo "Version: ${VERSION}"
  echo "Chart Version: ${CHART_VERSION}"

  update_values_file
  update_chart_file
}

main
