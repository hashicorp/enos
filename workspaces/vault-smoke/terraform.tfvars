aws_region               = "us-east-1"
aws_availability_zone    = "us-east-1a"
aws_ssh_key_pair_name    = "<your-aws-ssh-key-pair-name>"
aws_ssh_private_key_path = "</path/to/your/ssh/private/key.pem>"

vault_install_dir         = "/opt/vault/bin"
vault_instance_count      = 3
vault_license_path        = null
vault_artifactory_release = {
  username   = "<your-email@hashicorp.com>"
  token      = "<your-artifactory-token>"
  host       = "https://artifactory.hashicorp.engineering/artifactory"
  repo       = "hashicorp-packagespec-buildcache-local*"
  path       = "cache-v1/vault-enterprise/*"
  name       = "*.zip"
  properties = {
    "EDITION"         = "ent"
    "GOARCH"          = "amd64"
    "GOOS"            = "linux"
    "artifactType"    = "package"
    # productRevision should be the git SHA of the staged release
    "productRevision" = "f45845666b4e552bfc8ca775834a3ef6fc097fe0"
    "productVersion"  = "1.7.0" # this is required for the smoke test
  }
}

# Consul cluster
consul_license_path = null # only required for "ent" edition
consul_release = {
  version = "1.9.5"
  edition = "oss"
}
