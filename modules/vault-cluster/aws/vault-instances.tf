resource "aws_instance" "vault_instance" {
  count                  = var.instance_count
  ami                    = var.ubuntu_ami_id
  instance_type          = var.instance_type
  vpc_security_group_ids = [aws_security_group.enos_vault_sg.id]
  subnet_id              = tolist(data.aws_subnet_ids.infra.ids)[count.index]
  key_name               = var.ssh_aws_keypair
  tags = merge(
    var.common_tags,
    {
      Name = "${local.name_suffix}-vault-${count.index}"
    },
  )
}
