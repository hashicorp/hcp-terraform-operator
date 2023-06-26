#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

CHANGELOG_FILE="CHANGELOG.md"
RELEASE_DATE=`date "+%B %d, %Y"`

sed -i "" "s/UNRELEASED/$RELEASE_DATE/" $CHANGELOG_FILE
