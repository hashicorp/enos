api_addr = "http://${local_ipv4}:8200"
cluster_addr = "http://${local_ipv4}:8201"
ui = true

storage "consul" {
  address = "127.0.0.1:8500"
  path    = "vault"
}

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = "true"
}

seal "awskms" {
  kms_key_id = "${kms_key}"
}