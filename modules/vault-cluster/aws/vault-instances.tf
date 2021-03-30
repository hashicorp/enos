resource "aws_instance" "vault_instance" {
  count                  = var.instance_count
  ami                    = var.ubuntu_ami_id
  instance_type          = var.instance_type
  vpc_security_group_ids = [aws_security_group.enos_vault_sg.id]
  subnet_id              = tolist(data.aws_subnet_ids.infra.ids)[count.index]
  key_name               = var.ssh_aws_keypair
  iam_instance_profile   = aws_iam_instance_profile.vault_profile.name
  tags = merge(
    var.common_tags,
    {
      Name = "${local.name_suffix}-vault-${count.index}"
    },
  )
}

resource "enos_file" "vault_systemd" {
  depends_on = [
    aws_instance.vault_instance
  ]
  source      = "${path.module}/files/vault.service"
  destination = "/tmp/vault.service"

  for_each = toset([for idx in range(var.instance_count) : tostring(idx)])
  transport = {
    ssh = {
      host = aws_instance.vault_instance[tonumber(each.value)].public_ip
    }
  }
}

resource "enos_file" "vault_hcl" {
  depends_on = [
    aws_instance.vault_instance
  ]
  content     = data.template_file.server_hcl_template[tonumber(each.value)].rendered
  destination = "/tmp/server.hcl"

  for_each = toset([for idx in range(var.instance_count) : tostring(idx)])
  transport = {
    ssh = {
      host = aws_instance.vault_instance[tonumber(each.value)].public_ip
    }
  }
}

resource "enos_remote_exec" "install_vault" {
  depends_on = [
    enos_file.vault_systemd
  ]
  content = data.template_file.install_template[tonumber(each.value)].rendered

  for_each = toset([for idx in range(var.instance_count) : tostring(idx)])
  transport = {
    ssh = {
      host = aws_instance.vault_instance[tonumber(each.value)].public_ip
    }
  }
}

resource "enos_remote_exec" "configure_consul_agent" {
  depends_on = [
    enos_remote_exec.install_vault
  ]
  content = data.template_file.configure_consul_agent[tonumber(each.value)].rendered

  for_each = toset([for idx in range(var.instance_count) : tostring(idx)])
  transport = {
    ssh = {
      host = aws_instance.vault_instance[tonumber(each.value)].public_ip
    }
  }
}

resource "enos_remote_exec" "configure_vault" {
  depends_on = [
    enos_file.vault_hcl,
    enos_remote_exec.configure_consul_agent
  ]
  content = data.template_file.configure_template[tonumber(each.value)].rendered

  for_each = toset([for idx in range(var.instance_count) : tostring(idx)])
  transport = {
    ssh = {
      host = aws_instance.vault_instance[tonumber(each.value)].public_ip
    }
  }
}

resource "enos_remote_exec" "secure_vault_logs" {
  depends_on = [
    enos_remote_exec.configure_vault
  ]

  inline = [
    "sudo mkdir /var/log/vault.d",
    "sudo mv /tmp/vault_install.log  /var/log/vault.d/vault_install.log",
    "sudo mv /tmp/vault_consul_agent.log  /var/log/vault.d/vault_consul_agent.log",
    "sudo mv /tmp/vault_config.log  /var/log/vault.d/vault_config.log",
    "sudo chmod 600 /var/log/vault.d/*.log",
    "sudo chown -R root:root /var/log/vault.d"
  ]

  for_each = toset([for idx in range(var.instance_count) : tostring(idx)])
  transport = {
    ssh = {
      host = aws_instance.vault_instance[tonumber(each.value)].public_ip
    }
  }
}
