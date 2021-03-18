{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "ec2:DescribeInstances",
        "secretsmanager:*"
      ],
      "Effect": "Allow",
      "Resource": [ "*" ]
    },
    {
      "Action": [
        "kms:DescribeKey",
        "kms:ListKeys",
        "kms:Encrypt",
        "kms:Decrypt",
        "kms:GenerateDataKey"
      ],
      "Effect": "Allow",
      "Resource": [ "${kms_key_arn}" ]
    }
  ]
}
