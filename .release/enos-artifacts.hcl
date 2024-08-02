# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

schema = 1
artifacts {
  zip = [
    "enos_${version}_darwin_amd64.zip",
    "enos_${version}_darwin_arm64.zip",
    "enos_${version}_linux_amd64.zip",
    "enos_${version}_linux_arm64.zip",
  ]
  rpm = [
    "enos-${version_linux}-1.aarch64.rpm",
    "enos-${version_linux}-1.x86_64.rpm",
  ]
  deb = [
    "enos_${version_linux}-1_amd64.deb",
    "enos_${version_linux}-1_arm64.deb",
  ]
  container = [
    "enos_default_linux_amd64_${version}_${commit_sha}.docker.tar",
    "enos_default_linux_arm64_${version}_${commit_sha}.docker.tar",
  ]
}
