
resource "aws_iam_role" "vault_instance_role" {
  name               = "vault_instance_role"
  assume_role_policy = file("${path.module}/ec2_assume_role_policy.json")
}

resource "aws_iam_instance_profile" "vault_profile" {
  name = "vault_instance_profile"
  role = aws_iam_role.vault_instance_role.name
}

data "template_file" "vault_policy" {
  template = file("${path.module}/iam_policy.json.tpl")

  vars = {
    aws_account_id = data.aws_caller_identity.current.account_id
    kms_key_arn    = var.kms_key_arn
  }
}

resource "aws_iam_role_policy" "vault_policy" {
  name   = "vault_policy"
  role   = aws_iam_role.vault_instance_role.id
  policy = data.template_file.vault_policy.rendered
}
