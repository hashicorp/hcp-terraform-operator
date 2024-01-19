# Releasing

The purpose of this document is to outline the release process for the Terraform Cloud Operator.

The Semantic Versioning agreement is being followed by this project. Further details can be found [here](https://semver.org/). During the alpha or beta stages, the pre-release versions are not separated by dots. For example, `2.0.0-alpha1` or `2.0.0-beta5`.

## How To Release

To create a new release, adhere to the following steps:

- Switch to the `main` branch and fetch the latest changes.

  ```console
  $ git switch main
  $ git pull
  ```

- Generate the version number that will be released. Throughout the following steps, it will be denoted as `<SEMVER>`.

  ```console
  $ export TFC_OPERATOR_RELEASE_VERSION=`changie next auto`
  ```

- Create a new branch from the `main`. The branch name is required to adhere to the following template: `release/v<SEMVER>`.

  ```console
  $ git checkout -b release/v$TFC_OPERATOR_RELEASE_VERSION
  ```

- Modify the `version/VERSION` file to reflect the version number that you plan to release.

  ```console
  $ echo $TFC_OPERATOR_RELEASE_VERSION > version/VERSION
  ```

- Update the [`CHANGELOG`](./CHANGELOG.md) file with the change that were made since the last release.

  ```console
  $ changie batch auto
  $ changie marge
  ```

- Execute the script [update-helm-chart.sh](./scripts/update-helm-chart.sh) to update the [`Chart.yaml`](./charts/terraform-cloud-operator/Chart.yaml) file and match the desired release number. _The values of `version` and `appVersion` will be updated accordingly to the <SEMVER> value._


  ```console
  $ scripts/update-helm-chart.sh
  ```

- Commit and push all changes that were made.

  ```console
  $ git add -A
  $ git push
  ```

- Create a pull request against the `main` branch and follow the standard code review and merge procedures.

- After merging the release branch into the `main` branch, a git tag should have been automatically created for the new release version number. The version number in the tag must correspond with the `<SEMVER>` of the merged release branch name. Confirm this success by viewing the repository [tags](https://github.com/hashicorp/terraform-cloud-operator/tags).

- Follow the [CRT Usage](https://hashicorp.atlassian.net/wiki/spaces/RELENG/pages/2309390389/Part+3+CRT+Usage) guide to promote the release to the staging and production states.
