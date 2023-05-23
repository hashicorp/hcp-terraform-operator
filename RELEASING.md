# Releasing

The purpose of this document is to outline the release process for the Terraform Cloud Operator.

The Semantic Versioning agreement is being followed by this project. Further details can be found [here](https://semver.org/). During the alpha or beta stages, the pre-release versions are not separated by dots. For example, `2.0.0-alpha1` or `2.0.0-beta5`.

## How To Release

To create a new release, adhere to the following steps:

- Decide on the version number that you intend to release. Throughout the following steps, it will be denoted as <SEMVER>.

- Create a new branch from the `main`. The branch name is required to adhere to the following template: `release/v<SEMVER>`.

- Modify the `version/VERSION` file to reflect the version number that you plan to release. The version number in this file must correspond with the `<SEMVER>` of the release branch name.

- Revise the [`CHANGELOG`](./CHANGELOG.md) file by renaming the `UNRELEASED` section to the version number of the release or creating it if it doesn't already exist. The version number in this file must correspond with the `<SEMVER>` of the release branch name.

- Run script [update-helm-chart.sh](./scripts/update-helm-chart.sh) to update the [`Chart.yaml`](./charts/terraform-cloud-operator/Chart.yaml) and the [`values.yaml`](./charts/terraform-cloud-operator/values.yaml) files to match the desired release number:

  - [`Chart.yaml`](./charts/terraform-cloud-operator/Chart.yaml):
    - The value of `version` will be updated based on the value set in the environment variable `CHART_VERSION`, or if it is not set, it will be derived from the file `version/HELM_CHART_VERSION`.
    - The value of `appVersion` will be updated based on the value set in the environment variable `VERSION`, or if it is not set, it will be derived from the file `version/VERSION`.

  - [`values.yaml`](./charts/terraform-cloud-operator/values.yaml):
    - The value of `operator.image.tag` will be updated based on the value set in the environment variable `VERSION`, or if it is not set, it will be derived from the file `version/VERSION`.

- Create a pull request against the `main` branch and follow the regular code review and merge procedures.

- After merging the release branch into the `main` branch, a git tag should have been automatically created for the new release version number. The version number in the tag must correspond with the `<SEMVER>` of the merged release branch name. Confirm this succeeded by viewing the repository [tags](https://github.com/hashicorp/terraform-cloud-operator/tags).

- Follow the [CRT Usage](https://hashicorp.atlassian.net/wiki/spaces/RELENG/pages/2309390389/Part+3+CRT+Usage) guide to promote the release to the staging and production states.
