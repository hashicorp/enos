resource "aws_security_group" "consul_sg" {
  name        = "${local.name_suffix}-consul-sg"
  description = "SSH and Consul Traffic"
  vpc_id      = var.vpc_id

  # SSH
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    cidr_blocks      = ["50.248.238.101/32", join(",", data.aws_vpc.infra.cidr_block_associations.*.cidr_block)]
    description      = "value"
    from_port        = 8200
    to_port          = 8600
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    protocol         = "tcp"
    self             = null
    security_groups  = []
  }

  ingress {
    cidr_blocks      = ["50.248.238.101/32", join(",", data.aws_vpc.infra.cidr_block_associations.*.cidr_block)]
    description      = "value"
    from_port        = 8200
    to_port          = 8600
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    protocol         = "udp"
    self             = null
    security_groups  = []
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
      Name = "${local.name_suffix}-consul-sg"
    },
  )
}
