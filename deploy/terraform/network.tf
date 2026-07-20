# Security groups: Amazon API Gateway VPC Link -> private serve tasks.

resource "aws_security_group" "api_gateway_vpc_link" {
  name_prefix = "${var.name}-api-link-"
  description = "Amazon API Gateway VPC Link"
  vpc_id      = var.vpc_id
  tags        = local.tags

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_egress_rule" "api_gateway_to_service" {
  security_group_id            = aws_security_group.api_gateway_vpc_link.id
  description                  = "Sharecrop HTTP traffic to private tasks"
  ip_protocol                  = "tcp"
  from_port                    = 8080
  to_port                      = 8080
  referenced_security_group_id = aws_security_group.service.id
}

resource "aws_security_group" "service" {
  name_prefix = "${var.name}-service-"
  description = "serve Fargate tasks"
  vpc_id      = var.vpc_id
  tags        = local.tags

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_ingress_rule" "service_from_api_gateway" {
  security_group_id            = aws_security_group.service.id
  description                  = "Application traffic from Amazon API Gateway VPC Link"
  ip_protocol                  = "tcp"
  from_port                    = 8080
  to_port                      = 8080
  referenced_security_group_id = aws_security_group.api_gateway_vpc_link.id
}

resource "aws_vpc_security_group_egress_rule" "service_all" {
  security_group_id = aws_security_group.service.id
  description       = "All egress (image pull, PostgreSQL, AWS APIs)"
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}
