resource "aws_instance" "vault_instance" {
  count                  = var.instance_count
  ami                    = var.ubuntu_ami_id
  instance_type          = var.instance_type
  vpc_security_group_ids = [aws_security_group.enos_vault_sg.id]
  subnet_id              = tolist(data.aws_subnet_ids.infra.ids)[count.index]
  key_name               = var.ssh_aws_keypair
  iam_instance_profile   = aws_iam_instance_profile.vault_profile.name
  user_data              = base64encode(data.template_file.user_data_script.rendered)
  tags = merge(
    var.common_tags,
    {
      Name = "${local.name_suffix}-vault-${count.index}"
    },
  )
}

data "template_file" "user_data_script" {
  template = file("${path.module}/user-data.sh.tpl")
  vars = {
    package_url = var.package_url
    consul_ips = join(" ", var.consul_ips)
    kms_key = data.aws_kms_key.kms_key.id
    vault_license = var.vault_license
  }
}
