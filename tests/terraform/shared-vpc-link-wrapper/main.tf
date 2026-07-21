resource "aws_security_group" "shared_api_gateway_vpc_link" {
  name_prefix = "${var.name}-shared-link-"
  description = "Plan-time ownership contract for a shared Amazon API Gateway VPC Link"
  vpc_id      = var.vpc_id
}

resource "aws_apigatewayv2_vpc_link" "shared" {
  name               = "${var.name}-shared"
  subnet_ids         = var.task_subnet_ids
  security_group_ids = [aws_security_group.shared_api_gateway_vpc_link.id]
}

module "sharecrop" {
  source = "../../../deploy/terraform"

  name                                            = var.name
  region                                          = var.region
  image                                           = var.image
  release_revision                                = var.release_revision
  vpc_id                                          = var.vpc_id
  task_subnet_ids                                 = var.task_subnet_ids
  existing_ecs_cluster_arn                        = var.existing_ecs_cluster_arn
  certificate_arn                                 = var.certificate_arn
  domain_name                                     = var.domain_name
  database_url_secret_arn                         = var.database_url_secret_arn
  create_api_gateway_vpc_link                     = false
  existing_api_gateway_vpc_link_id                = aws_apigatewayv2_vpc_link.shared.id
  existing_api_gateway_vpc_link_security_group_id = aws_security_group.shared_api_gateway_vpc_link.id
}
