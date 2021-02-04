resource "aws_instance" "vault_instance" {
  count           = var.instances_per_subnet
  ami             = module.enos_infra.ubuntu_ami_id
  instance_type   = var.instance_type
  security_groups = [aws_security_group.enos_vault_sg.id]
  subnet_id       = module.enos_infra.vpc_subnet_ids[count.index]
  key_name        = var.key_name
  tags = merge(
    var.common_tags,
    {
      Name = "${local.name_suffix}-vault"
    },
  )
}