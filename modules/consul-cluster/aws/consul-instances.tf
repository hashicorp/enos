locals {
  infra_subnet_blocks = [for s in data.aws_subnet.infra : s.cidr_block]
}

resource "aws_instance" "consul_instance" {
  count                  = var.instance_count
  ami                    = var.ubuntu_ami_id
  instance_type          = var.instance_type
  vpc_security_group_ids = [aws_security_group.enos_consul_sg.id]
  subnet_id              = local.infra_subnet_blocks[count.index]
  key_name               = var.ssh_aws_keypair
  tags = merge(
    var.common_tags,
    {
      Name = "${local.name_suffix}-consul-${count.index}"
    },
  )
}
