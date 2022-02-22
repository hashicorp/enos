terraform {
  required_providers {
    enos = {
      version = ">= 0.1.13"
      source  = "hashicorp.com/qti/enos"
    }

    aws = {
      source = "hashicorp/aws"
    }
  }
}

data "aws_vpc" "default" {
  default = true
}

data "aws_ami" "ubuntu" {
  most_recent = true

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  owners = ["099720109477"] # Canonical
}

data "aws_ami" "rhel" {
  most_recent = true

  # Currently latest latest point release-1
  filter {
    name   = "name"
    values = ["RHEL-8.2*HVM-20*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  owners = ["309956199498"] # Redhat
}

locals {
  ami = {
    "ubuntu" = data.aws_ami.ubuntu.id
    "rhel"   = data.aws_ami.rhel.id
  }

  tags = {
    Project = "Enos"
    Name    = "Enos CI Target"
  }
}

data "enos_environment" "localhost" {
}

module "target_sg" {
  source = "terraform-aws-modules/security-group/aws//modules/ssh"

  name        = "enos_core_example"
  description = "Enos provider core example security group"
  vpc_id      = data.aws_vpc.default.id
  tags        = local.tags

  ingress_cidr_blocks = ["${data.enos_environment.localhost.public_ip_address}/32"]
}

resource "aws_instance" "target" {
  ami                         = local.ami[var.distro]
  instance_type               = "t3.micro"
  key_name                    = "enos-ci-ssh-keypair"
  associate_public_ip_address = true
  tags                        = local.tags
  security_groups             = [module.target_sg.security_group_name]
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
