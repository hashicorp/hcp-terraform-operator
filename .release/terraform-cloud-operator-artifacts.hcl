schema = 1
artifacts {
  zip = [
    "hcp-terraform-operator_${version}_linux_amd64.zip",
    "hcp-terraform-operator_${version}_linux_arm64.zip",
  ]
  container = [
    "hcp-terraform-operator_release-default_linux_amd64_${version}_${commit_sha}.docker.dev.tar",
    "hcp-terraform-operator_release-default_linux_amd64_${version}_${commit_sha}.docker.tar",
    "hcp-terraform-operator_release-default_linux_arm64_${version}_${commit_sha}.docker.dev.tar",
    "hcp-terraform-operator_release-default_linux_arm64_${version}_${commit_sha}.docker.tar",
  ]
}
