data "aws_iam_policy_document" "consul_instance_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "consul_profile" {
  statement {
    resources = ["*"]

    actions   = ["ec2:DescribeInstances"]
  }

  statement {
    resources = [var.kms_key_arn]

    actions = [
      "kms:DescribeKey",
      "kms:ListKeys",
      "kms:Encrypt",
      "kms:Decrypt",
    ]
  }
}

resource "random_string" "iam_suffix" {
  length = 8
  lower = true
  upper = false
  number = false
  special = false
}

resource "aws_iam_role" "consul_instance_role" {
  name               = "consul_instance_role-${random_string.iam_suffix.result}"
  assume_role_policy = data.aws_iam_policy_document.consul_instance_role.json
}

resource "aws_iam_instance_profile" "consul_profile" {
  name = "consul_instance_profile-${random_string.iam_suffix.result}"
  role = aws_iam_role.consul_instance_role.name
}

resource "aws_iam_role_policy" "consul_policy" {
  name   = "consul_policy-${random_string.iam_suffix.result}"
  role   = aws_iam_role.consul_instance_role.id
  policy = data.aws_iam_policy_document.consul_profile.json
}
