resource "aws_instance" "consul_instance" {
  count                  = var.instance_count
  ami                    = var.ubuntu_ami_id
  instance_type          = var.instance_type
  vpc_security_group_ids = [aws_security_group.consul_sg.id]
  subnet_id              = tolist(data.aws_subnet_ids.infra.ids)[count.index]
  key_name               = var.ssh_aws_keypair
  iam_instance_profile   = aws_iam_instance_profile.consul_profile.name
  availability_zone      = var.availability_zone != "" ? var.availability_zone : null

  tags = merge(
    var.common_tags,
    {
      Name = "${local.name_suffix}-consul-${count.index}",
      Type = "consul-server"
    },
  )
}

resource "enos_remote_exec" "install_consul" {
  depends_on = [aws_instance.consul_instance]

  content = templatefile("${path.module}/install-consul.sh.tpl", {
    "package_url"    = var.package_url
    "consul_license" = file("${path.root}/consul.hclic")
  })

  for_each = toset([ for idx in range(var.instance_count): tostring(idx) ])
  transport = {
    ssh = {
      host = aws_instance.consul_instance[tonumber(each.value)].public_ip
    }
  }
}

resource "enos_remote_exec" "start_consul" {
  depends_on = [enos_remote_exec.install_consul]

  inline = [
    "sudo touch /var/log/consul-nohup.log",
    "sudo chmod 777 /var/log/consul-nohup.log",
    "sudo nohup consul agent -retry-join 'provider=aws tag_key=Type tag_value=consul-server' -data-dir=/tmp/consul -server -bootstrap-expect=3 -log-file=/var/log/consul.log -ui -client 0.0.0.0 &> /var/log/consul-nohup.log &",
  ]

  for_each = toset([ for idx in range(var.instance_count): tostring(idx) ])
  transport = {
    ssh = {
      host = aws_instance.consul_instance[tonumber(each.value)].public_ip
    }
  }
}

resource "enos_remote_exec" "wait_for_consul" {
  depends_on = [enos_remote_exec.start_consul]

  content = templatefile("${path.module}/wait-for-consul.sh.tpl", {
    "consul_license" = file("${path.root}/consul.hclic")
  })

  for_each = toset([ for idx in range(var.instance_count): tostring(idx) ])
  transport = {
    ssh = {
      host = aws_instance.consul_instance[tonumber(each.value)].public_ip
    }
  }
}
