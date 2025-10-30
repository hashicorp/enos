# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    enos = {
      source  = "hashicorp-forge/enos"
      version = "0.6.2"
    }

    aws = {
      source = "hashicorp/aws"
    }
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-*-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  owners = ["099720109477"] # Canonical
}

data "aws_ami" "rhel" {
  most_recent = true

  filter {
    name   = "name"
    values = ["RHEL-10.0*HVM-20*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  owners = ["309956199498"] # Redhat
}

locals {
  ami = {
    "ubuntu" = data.aws_ami.ubuntu.id
    "rhel"   = data.aws_ami.rhel.id
  }

  tags = merge(var.tags, {
    Project = "Enos"
    Name    = "Enos CI Target"
  })
  cidr_block = "10.13.0.0/16"
}

data "enos_environment" "localhost" {
}


data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "zone-name"
    values = ["*"]
  }
}

resource "random_string" "cluster_id" {
  length  = 8
  lower   = true
  upper   = false
  numeric = false
  special = false
}

resource "aws_vpc" "vpc" {
  // Always set the ipv4 cidr block as it's required in "dual-stack" VPCs which we create.
  cidr_block                       = local.cidr_block
  enable_dns_hostnames             = true
  enable_dns_support               = true
  assign_generated_ipv6_cidr_block = false
  tags                             = { "Name" = "Enos CI E2E" }
}

resource "aws_subnet" "subnet" {
  count             = length(data.aws_availability_zones.available.names)
  vpc_id            = aws_vpc.vpc.id
  availability_zone = data.aws_availability_zones.available.names[count.index]

  // IPV4, but since we need to support ipv4 connections from the machine running enos, we're
  // always going to need ipv4 available.
  map_public_ip_on_launch = true
  cidr_block              = cidrsubnet(local.cidr_block, 8, count.index)

  // IPV6, only set these when we want to run in ipv6 mode.
  assign_ipv6_address_on_creation = false
  ipv6_cidr_block                 = null

  tags = {
    "Name" = "Enos-e2e-subnet-${data.aws_availability_zones.available.names[count.index]}"
  }
}

resource "aws_internet_gateway" "ipv4" {
  vpc_id = aws_vpc.vpc.id

  tags = {
    "Name" = "enos-e2e-igw"
  }
}

resource "aws_route" "igw_ipv4" {
  route_table_id         = aws_vpc.vpc.default_route_table_id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.ipv4.id
}

resource "aws_security_group" "default" {
  vpc_id = aws_vpc.vpc.id

  ingress {
    description      = "allow_ingress_from_all"
    from_port        = 0
    to_port          = 0
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = null
  }

  egress {
    description      = "allow_egress_from_all"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = null
  }

  tags = {
    "Name" = "enos-e2e-default"
  }
}

module "target_sg" {
  source = "terraform-aws-modules/security-group/aws//modules/ssh"

  name        = "enos_core_example"
  description = "Enos provider core example security group"
  vpc_id      = aws_vpc.vpc.id
  tags        = local.tags

  ingress_cidr_blocks = ["${data.enos_environment.localhost.public_ipv4_addresses[0]}/32"]
}

resource "aws_instance" "target" {
  ami                         = local.ami[var.distro]
  instance_type               = "t3.micro"
  key_name                    = "enos-ci-ssh-key"
  associate_public_ip_address = true
  tags                        = local.tags
  vpc_security_group_ids      = [module.target_sg.security_group_id]
  subnet_id                   = aws_subnet.subnet[0].id
}

resource "enos_file" "from_source" {
  depends_on = [aws_instance.target]

  content     = "content"
  destination = "/tmp/from_source.txt"

  transport = {
    ssh = {
      host = aws_instance.target.public_ip
    }
  }
}
