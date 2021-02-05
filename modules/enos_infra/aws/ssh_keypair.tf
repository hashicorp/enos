resource "aws_key_pair" "enos_aws_keypair" {
  key_name   = "enos-aws-keypair"
  public_key = var.ssh_pub_key
}