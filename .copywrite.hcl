schema_version = 1

project {
  license        = "MPL-2.0"
  copyright_year = 2022
  header_ignore = [
    ".github/**",
    # All files within the config directory are generated automatically by the operator-sdk CLI tool
    # Some files were scaffolded during the first run of the operator-sdk CLI tool and never changed
    # The dev team is not working with these files they all serve a supporting role
    "config/**",
    # Changie is a tool to manage changelog entries
    ".changes/unreleased/*.yaml",
    ".changie.yaml",
  ]
}
