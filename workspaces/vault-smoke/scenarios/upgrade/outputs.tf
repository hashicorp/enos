output "clusters" {
  value = {for ed in var.vault_product_editions_to_test: ed => {
    consul_instance_private_ips  = module.consul[index(var.vault_product_editions_to_test, ed)].instance_private_ips
    consul_instance_public_ips   = module.consul[index(var.vault_product_editions_to_test, ed)].instance_public_ips
    consul_instance_instance_ids = module.consul[index(var.vault_product_editions_to_test, ed)].instance_ids
    vault_instance_private_ips   = module.vault[index(var.vault_product_editions_to_test, ed)].instance_private_ips
    vault_instance_public_ips    = module.vault[index(var.vault_product_editions_to_test, ed)].instance_public_ips
    vault_instance_instance_ids  = module.vault[index(var.vault_product_editions_to_test, ed)].instance_ids
    vault_root_token             = module.vault[index(var.vault_product_editions_to_test, ed)].vault_root_token
    vault_recovery_keys_b64      = module.vault[index(var.vault_product_editions_to_test, ed)].vault_recovery_keys_b64
    vault_artifactory_bundle     = {
      url    = data.enos_artifactory_item.vault[index(var.vault_product_editions_to_test, ed)].results[0].url
      sha256 = data.enos_artifactory_item.vault[index(var.vault_product_editions_to_test, ed)].results[0].sha256
    }
  }}
}
