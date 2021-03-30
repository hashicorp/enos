resource "aws_security_group" "enos_vault_sg" {
  name        = "${local.name_suffix}-vault-sg"
  description = "SSH and Vault Traffic"
  vpc_id      = var.vpc_id

  # SSH
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["${data.enos_environment.localhost.public_ip_address}/32", join(",", data.aws_vpc.infra.cidr_block_associations.*.cidr_block)]
  }

  # Vault API traffic
  ingress {
    from_port   = 8200
    to_port     = 8200
    protocol    = "tcp"
    cidr_blocks = ["${data.enos_environment.localhost.public_ip_address}/32", join(",", data.aws_vpc.infra.cidr_block_associations.*.cidr_block)]
  }

  # Vault cluster traffic
  ingress {
    from_port   = 8201
    to_port     = 8201
    protocol    = "tcp"
    cidr_blocks = ["${data.enos_environment.localhost.public_ip_address}/32", join(",", data.aws_vpc.infra.cidr_block_associations.*.cidr_block)]
  }

  # Internal Traffic
  ingress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    self      = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(
    var.common_tags,
    {
      Name = "${local.name_suffix}-vault-sg"
    },
  )
}