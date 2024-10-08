# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

schema = "1"

project "hcp-terraform-operator" {
  team = "terraform"
  slack {
    notification_channel = "C051FAAHL8M" # feed-terraform-ecosystem-kubernetes-releases
  }
  github {
    organization = "hashicorp"
    repository   = "hcp-terraform-operator"
    release_branches = [
      "main",
      "release/**",
    ]
  }
}

event "merge" {
  // "entrypoint" to use if build is not run automatically
  // i.e. send "merge" complete signal to orchestrator to trigger build
}

event "build" {
  depends = ["merge"]
  action "build" {
    organization = "hashicorp"
    repository   = "hcp-terraform-operator"
    workflow     = "build"
  }

  notification {
    on = "fail"
  }
}

event "prepare" {
  depends = ["build"]

  action "prepare" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "prepare"
  }

  notification {
    on = "fail"
  }
}

## These are promotion and post-publish events
## they should be added to the end of the file after the verify event stanza.

event "trigger-staging" {
  // This event is dispatched by the bob trigger-promotion command
  // and is required - do not delete.
}

event "promote-staging" {
  depends = ["trigger-staging"]
  action "promote-staging" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "promote-staging"
    config       = "release-metadata.hcl"
  }

  notification {
    on = "always"
  }
}

event "promote-staging-docker" {
  depends = ["promote-staging"]
  action "promote-staging-docker" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "promote-staging-docker"
  }

  notification {
    on = "always"
  }
}

event "trigger-production" {
  // This event is dispatched by the bob trigger-promotion command
  // and is required - do not delete.
}

event "promote-production" {
  depends = ["trigger-production"]
  action "promote-production" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "promote-production"
  }

  notification {
    on = "always"
  }
}

event "promote-production-docker" {
  depends = ["promote-production"]
  action "promote-production-docker" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "promote-production-docker"
  }

  notification {
    on = "always"
  }
}

event "promote-production-helm" {
  depends = ["promote-production-docker"]
  action "promote-production-helm" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "promote-production-helm"
  }

  notification {
    on = "always"
  }
}
