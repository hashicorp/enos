data "aws_vpc" "infra" {
  id = var.vpc_id
}

data "aws_subnet_ids" "infra" {
  vpc_id = var.vpc_id
}

data "aws_subnet" "infra" {
  for_each = data.aws_subnet_ids.infra.ids
  id       = each.value
}
