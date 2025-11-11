# Releasing

The purpose of this document is to outline the release process for the HCP Terraform Operator.

The Semantic Versioning agreement is being followed by this project. Further details can be found [here](https://semver.org/). During the alpha or beta stages, the pre-release versions are not separated by dots. For example, `2.0.0-alpha1` or `2.0.0-beta5`.

## How To Release

To create a new release, adhere to the following steps:

- Ensure that all checks of the latest merge are successful; if any checks fail, address the issues before proceeding.

- Switch to the `main` branch and fetch the latest changes.

  ```console
  $ git switch main
  $ git pull
  ```

- Generate the version number that will be released. Depending on the changes, the `auto` option will typically create a new minor release version. If you want to specify the release type explicitly, use the `patch` or `minor` option. Throughout the following steps, it will be denoted as `<SEMVER>`.

  ```console
  $ export HCP_TF_OPERATOR_RELEASE_VERSION=`changie next auto`
  $ echo $HCP_TF_OPERATOR_RELEASE_VERSION
  ```

  If the output does not match the desired version, repeat the step by specifying the release type.

- Create a new branch from the `main`. The branch name is required to adhere to the following template: `release/v<SEMVER>`.

  ```console
  $ git checkout -b release/v$HCP_TF_OPERATOR_RELEASE_VERSION
  ```

- Modify the `version/VERSION` file to reflect the version number that you plan to release.

  ```console
  $ echo $HCP_TF_OPERATOR_RELEASE_VERSION > version/VERSION
  $ cat version/VERSION
  ```

- Update the [`CHANGELOG`](./CHANGELOG.md) file with the change that were made since the last release. It is always a good practice to use the `--dry-run` option to verify that the final changelog entry appears as expected before proceeding.

  ```console
  $ changie batch auto --dry-run
  ```

  Note that the version option (`auto` in the example above) must match the option you selected earlier. If there are any issues, address them before proceeding.

  Once the log entry appears as expected, proceed with the generation.

  ```console
  $ changie batch auto
  ```

  _This is the time to review the corresponding changelog for the version in the `.changie/<SEMVER>.md` file and update it if necessary. Please include any community contribution acknowledgements here._

  Merge the release changes into the [`CHANGELOG`](./CHANGELOG.md) file.

  ```console
  $ changie merge
  ```

  A new file with the changelog entry will be added to the `.changes` directory: `.changes/<SEMVER>.md`, and the same entry will be prepended to the [CHANGELOG.md](./CHANGELOG.md) file. Ensure both look correct before proceeding.

- Execute the script [update-helm-chart.sh](./scripts/update-helm-chart.sh) to update the [`Chart.yaml`](./charts/hcp-terraform-operator/Chart.yaml) file and match the desired release number. _The values of `version` and `appVersion` will be updated accordingly to the <SEMVER> value._


  ```console
  $ scripts/update-helm-chart.sh
  ```

- Update the Helm Chart [`README.md`](./charts/hcp-terraform-operator/README.md) file.

  ```console
  $ make helm-docs
  ```

- Bump the release version in the installation instructions of the [README.md](./README.md) file to match the desired release number.

- Bump the `newTag` version in the [kustomization.yaml](./config/manager/kustomization.yaml) file to match the desired release number.

- Commit and push all changes that were made.

  ```console
  $ git add -A
  $ git commit -m "v$HCP_TF_OPERATOR_RELEASE_VERSION"
  $ git push
  ```

- Create a pull request against the `main` branch and follow the standard code review and merge procedures. Ensure that E2E tests are attached and pass. It is always a good idea to [refer](https://github.com/hashicorp/hcp-terraform-operator/pulls?q=is%3Apr+label%3Arelease) to one of the previous relese PRs for the format and structure.

- After merging the release branch into the `main` branch, a git tag should have been automatically created for the new release version number. The version number in the tag must correspond with the `<SEMVER>` of the merged release branch name. Confirm this success by viewing the repository [tags](https://github.com/hashicorp/hcp-terraform-operator/tags).

- Follow the [CRT Usage](https://hashicorp.atlassian.net/wiki/spaces/RELENG/pages/2309390389/Part+3+CRT+Usage) guide to promote the release to the staging and production states.

## How To Release Red Hat OpenShift Bundle

Ensure that your GitHub username is added to the HCP Terraform Operator Bundle before proceeding.

  - Log in to the [Red Hat Partner Connect portal](https://connect.redhat.com/manage/products).

  - Navigate to **HashiCorp Terraform** > **Components & Testing** > **HCP Terraform Operator Bundle** > **Repository Information**.

  - Append your GitHub username to **Authorized GitHub user accounts**.

  - Save changes.

*The following steps apply once the CRT release is completed and the Red Hat UBI image is successfully released. Navigate to the **HCP Terraform Operator Image** on the Red Hat Partner Connect portal to ensure the desired image version is available.*

- Fork the Red Hat Certified Operators production catalog [repository](https://github.com/redhat-openshift-ecosystem/certified-operators).

- Generate a new bundle in the HCP Terraform operator repository:

```console
$ export HCP_TF_OPERATOR_RELEASE_VERSION=`cat version/VERSION`
$ make bundle VERSION=$HCP_TF_OPERATOR_RELEASE_VERSION
```

- Create a new branch in the Red Hat Certified Operators resoitory following the `hcp-terraform-operator-v$HCP_TF_OPERATOR_RELEASE_VERSION` pattern:

```console
$ git switch main
$ git pull
$ git checkout -b hcp-terraform-operator-v$HCP_TF_OPERATOR_RELEASE_VERSION
```

- Copy the generated bundle from the HCP Terraform operator repository to the Red Hat Certified Operators reposiroty:

```console
$ cp -R <HCP_OPERATOR_REPO>/bundle/* <RED_HAT_CERT_OPERATOR_REPO>/certified-operators/operators/hcp-terraform-operator/$HCP_TF_OPERATOR_RELEASE_VERSION/`
```

- Review, commit and push changes in the Red Hat Certified Operators resoitory:

```console
$ git add -A
$ git commit -m "operator hcp-terraform-operator ($HCP_TF_OPERATOR_RELEASE_VERSION)"
$ git push
```

  Ensure that the `spec.replaces` filed in the `hcp-terraform-operator.clusterserviceversion.yaml` file points to the previous release. For example:

```yaml
spec:
  replaces: hcp-terraform-operator.v2.8.0
  version: 2.8.1
```

- Make a PR in the Red Hat Certified Operators repository on GitHub and wait until it gets merged. If there are any issues, address them.

- Validate that the bundle is now available on the Red Hat Partner Connect portal.
