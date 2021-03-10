resource "aws_instance" "consul_instance" {
  count                  = var.instance_count
  ami                    = var.ubuntu_ami_id
  instance_type          = var.instance_type
  vpc_security_group_ids = [aws_security_group.consul_sg.id]
  subnet_id              = tolist(data.aws_subnet_ids.infra.ids)[count.index]
  key_name               = var.ssh_aws_keypair
  user_data              = base64encode(data.template_file.user_data_script.rendered)
  iam_instance_profile   = aws_iam_instance_profile.consul_profile.name

  tags = merge(
    var.common_tags,
    {
      Name = "${local.name_suffix}-consul-${count.index}",
      Type = "consul-server"
    },
  )
}
data "template_file" "user_data_script" {
  template = file("${path.module}/user-data.sh.tpl")
  vars = {
    "package_url"    = var.package_url
    "consul_license" = file("${path.root}/consul.hclic")
  }
}
