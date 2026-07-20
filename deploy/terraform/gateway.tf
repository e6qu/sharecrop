resource "aws_service_discovery_private_dns_namespace" "this" {
  name = "${var.name}.internal"
  vpc  = var.vpc_id
  tags = local.tags
}

resource "aws_service_discovery_service" "this" {
  name = "serve"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.this.id
    dns_records {
      ttl  = 10
      type = "SRV"
    }
    routing_policy = "MULTIVALUE"
  }

  # Amazon ECS publishes task and container-health state to AWS Cloud Map.
  # Amazon API Gateway's DiscoverInstances calls then select only healthy tasks.
  health_check_custom_config {}

  tags = local.tags
}

resource "aws_apigatewayv2_vpc_link" "this" {
  count = var.create_api_gateway_vpc_link ? 1 : 0

  name               = var.name
  subnet_ids         = var.task_subnet_ids
  security_group_ids = [aws_security_group.api_gateway_vpc_link[0].id]
  tags               = local.tags
}

locals {
  api_gateway_vpc_link_id = var.create_api_gateway_vpc_link ? aws_apigatewayv2_vpc_link.this[0].id : var.existing_api_gateway_vpc_link_id
  api_gateway_vpc_link_security_groups = var.create_api_gateway_vpc_link ? {
    managed = aws_security_group.api_gateway_vpc_link[0].id
    } : {
    shared = var.existing_api_gateway_vpc_link_security_group_id
  }
}

resource "aws_apigatewayv2_api" "this" {
  name                         = var.name
  protocol_type                = "HTTP"
  disable_execute_api_endpoint = true
  tags                         = local.tags

  lifecycle {
    precondition {
      condition = !var.create_api_gateway_vpc_link || (
        var.existing_api_gateway_vpc_link_id == "" &&
        var.existing_api_gateway_vpc_link_security_group_id == ""
      )
      error_message = "Dedicated VPC Link mode rejects existing_api_gateway_vpc_link_id and existing_api_gateway_vpc_link_security_group_id."
    }

    precondition {
      condition = var.create_api_gateway_vpc_link || (
        var.existing_api_gateway_vpc_link_id != "" &&
        var.existing_api_gateway_vpc_link_security_group_id != ""
      )
      error_message = "Shared VPC Link mode requires both existing_api_gateway_vpc_link_id and existing_api_gateway_vpc_link_security_group_id."
    }
  }
}

resource "aws_apigatewayv2_integration" "this" {
  api_id                 = aws_apigatewayv2_api.this.id
  integration_type       = "HTTP_PROXY"
  integration_uri        = aws_service_discovery_service.this.arn
  integration_method     = "ANY"
  connection_type        = "VPC_LINK"
  connection_id          = local.api_gateway_vpc_link_id
  payload_format_version = "1.0"
  timeout_milliseconds   = 30000

  request_parameters = {
    "overwrite:path" = "$request.path"
  }
}

resource "aws_apigatewayv2_route" "this" {
  api_id    = aws_apigatewayv2_api.this.id
  route_key = "$default"
  target    = "integrations/${aws_apigatewayv2_integration.this.id}"
}

resource "aws_cloudwatch_log_group" "api_access" {
  name              = "/aws/apigateway/${var.name}"
  retention_in_days = 30
  tags              = local.tags
}

resource "aws_apigatewayv2_stage" "this" {
  api_id      = aws_apigatewayv2_api.this.id
  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api_access.arn
    format = jsonencode({
      requestId        = "$context.requestId"
      sourceIp         = "$context.identity.sourceIp"
      requestMethod    = "$context.httpMethod"
      requestPath      = "$context.path"
      responseStatus   = "$context.status"
      responseBytes    = "$context.responseLength"
      integrationError = "$context.integrationErrorMessage"
      totalLatencyMs   = "$context.responseLatency"
    })
  }

  default_route_settings {
    detailed_metrics_enabled = true
    throttling_burst_limit   = var.api_throttling_burst_limit
    throttling_rate_limit    = var.api_throttling_rate_limit
  }

  tags = local.tags

  depends_on = [aws_apigatewayv2_route.this]
}

resource "aws_apigatewayv2_domain_name" "this" {
  domain_name = var.domain_name

  domain_name_configuration {
    certificate_arn = var.certificate_arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }

  tags = local.tags
}

resource "aws_apigatewayv2_api_mapping" "this" {
  api_id      = aws_apigatewayv2_api.this.id
  domain_name = aws_apigatewayv2_domain_name.this.id
  stage       = aws_apigatewayv2_stage.this.name
}
