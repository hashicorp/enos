resource "aws_vpc" "enos_vpc" {
  cidr_block = var.vpc_cidr
  tags = merge(
    var.common_tags,
    {
      "Name" = var.vpc_name
    },
  )
}

resource "aws_subnet" "enos_subnet" {
  vpc_id                  = aws_vpc.enos_vpc.id
  count                   = length(data.aws_availability_zones.available.names)
  cidr_block              = "10.13.${10 + count.index}.0/24"
  map_public_ip_on_launch = true
  availability_zone       = data.aws_availability_zones.available.names[count.index]

  tags = merge(
    var.common_tags,
    {
      "Name" = "${var.vpc_name}_subnet_${count.index}"
    },
  )
}

resource "aws_internet_gateway" "enos_gw" {
  vpc_id = aws_vpc.enos_vpc.id
  tags = merge(
    var.common_tags,
    {
      "Name" = "${var.vpc_name}_gw"
    },
  )
}

resource "aws_route_table" "enos_route" {
  vpc_id = aws_vpc.enos_vpc.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.enos_gw.id
  }

  tags = merge(
    var.common_tags,
    {
      "Name" = "${var.vpc_name}_route"
    },
  )
}

resource "aws_route_table_association" "enos_crta" {
  subnet_id      = aws_subnet.enos_subnet[count.index].id
  count          = length(data.aws_availability_zones.available.names)
  route_table_id = aws_route_table.enos_route.id
}

resource "aws_security_group" "enos_default_sg" {
  vpc_id = aws_vpc.enos_vpc.id

  ingress {
    description = "allow traffic from all IPs"
    from_port   = 0
    to_port     = 0
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
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
      "Name" = "${var.vpc_name}_default_sg"
    },
  )
}