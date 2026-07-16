resource "aws_ecs_cluster" "this" {
  name = var.name
  tags = local.tags
}

resource "aws_cloudwatch_log_group" "serve" {
  name_prefix       = "/ecs/${var.name}-serve-"
  retention_in_days = 30
  tags              = local.tags
}

resource "aws_cloudwatch_log_group" "migrate" {
  name_prefix       = "/ecs/${var.name}-migrate-"
  retention_in_days = 30
  tags              = local.tags
}

locals {
  # Injected into every container as `secrets`.
  secrets = concat([
    { name = "DATABASE_URL", valueFrom = aws_secretsmanager_secret.database_url.arn },
    { name = "SHARECROP_ACCESS_TOKEN_SECRET", valueFrom = aws_secretsmanager_secret.access_token.arn },
  ], var.shauth_oidc_client_secret_arn == "" ? [] : [{ name = "SHARECROP_SHAUTH_CLIENT_SECRET", valueFrom = var.shauth_oidc_client_secret_arn }])

  # Private-image pull credentials, only when an image pull secret is configured.
  repository_credentials = var.image_pull_secret_arn == null ? {} : {
    repositoryCredentials = { credentialsParameter = var.image_pull_secret_arn }
  }

  serve_container = merge({
    name         = "sharecrop"
    image        = var.image
    essential    = true
    command      = ["serve"]
    portMappings = [{ containerPort = 8080, protocol = "tcp" }]
    environment  = concat([{ name = "SHARECROP_HTTP_ADDR", value = ":8080" }], var.shauth_oidc_issuer == "" ? [] : [{ name = "SHARECROP_SHAUTH_ISSUER", value = var.shauth_oidc_issuer }, { name = "SHARECROP_SHAUTH_CLIENT_ID", value = var.shauth_oidc_client_id }, { name = "SHARECROP_PUBLIC_URL", value = var.public_url }])
    secrets      = local.secrets
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.serve.name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "sharecrop"
      }
    }
  }, local.repository_credentials)

  migrate_container = merge({
    name      = "sharecrop-migrate"
    image     = var.image
    essential = true
    command   = ["migrate", "up"]
    # migrate only needs the database; the access-token secret is harmless but
    # omitted to keep the one-off task minimal.
    secrets = [local.secrets[0]]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.migrate.name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "sharecrop"
      }
    }
  }, local.repository_credentials)
}

resource "aws_ecs_task_definition" "serve" {
  family                   = "${var.name}-serve"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.cpu
  memory                   = var.memory
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn

  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }

  container_definitions = jsonencode([local.serve_container])
  tags                  = local.tags
  lifecycle {
    precondition {
      condition     = (var.shauth_oidc_issuer == "" && var.shauth_oidc_client_id == "" && var.shauth_oidc_client_secret_arn == "" && var.public_url == "") || (var.shauth_oidc_issuer != "" && var.shauth_oidc_client_id != "" && var.shauth_oidc_client_secret_arn != "" && var.public_url != "")
      error_message = "All Shauth OIDC coordinates and public_url must be configured together."
    }
  }
}

resource "aws_ecs_task_definition" "migrate" {
  family                   = "${var.name}-migrate"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn

  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }

  container_definitions = jsonencode([local.migrate_container])
  tags                  = local.tags
}

resource "aws_ecs_service" "serve" {
  name            = "${var.name}-serve"
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.serve.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  # Give a fresh task time to pass health checks (the guest pool warms from the
  # baked cache, so this is generous).
  health_check_grace_period_seconds = 60

  network_configuration {
    subnets          = var.task_subnet_ids
    security_groups  = [aws_security_group.service.id]
    assign_public_ip = var.assign_public_ip
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.this.arn
    container_name   = "sharecrop"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.http]
  tags       = local.tags
}
