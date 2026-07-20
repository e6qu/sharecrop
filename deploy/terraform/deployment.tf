data "aws_partition" "current" {}

data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "step_functions_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "aws:SourceAccount"
      values   = [data.aws_caller_identity.current.account_id]
    }

    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values = [
        "arn:${data.aws_partition.current.partition}:states:${var.region}:${data.aws_caller_identity.current.account_id}:stateMachine:${var.name}-deploy",
      ]
    }
  }
}

resource "aws_iam_role" "deployment" {
  name               = "${var.name}-deploy"
  assume_role_policy = data.aws_iam_policy_document.step_functions_assume.json
  tags               = local.tags
}

locals {
  ecs_service_arn = "arn:${data.aws_partition.current.partition}:ecs:${var.region}:${data.aws_caller_identity.current.account_id}:service/${local.ecs_cluster_name}/${aws_ecs_service.serve.name}"
}

data "aws_iam_policy_document" "deployment" {
  statement {
    actions   = ["ecs:RunTask"]
    resources = [aws_ecs_task_definition.migrate.arn]
  }

  statement {
    actions = [
      "ecs:DescribeTasks",
      "ecs:StopTask",
    ]
    resources = ["*"]
  }

  statement {
    actions = [
      "events:DescribeRule",
      "events:PutRule",
      "events:PutTargets",
    ]
    resources = [
      "arn:${data.aws_partition.current.partition}:events:${var.region}:${data.aws_caller_identity.current.account_id}:rule/StepFunctionsGetEventsForECSTaskRule",
    ]
  }

  statement {
    actions = [
      "ecs:DescribeServices",
      "ecs:UpdateService",
    ]
    resources = [local.ecs_service_arn]
  }

  statement {
    actions = ["iam:PassRole"]
    resources = [
      aws_iam_role.execution.arn,
      aws_iam_role.task.arn,
    ]

    condition {
      test     = "StringEquals"
      variable = "iam:PassedToService"
      values   = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy" "deployment" {
  role   = aws_iam_role.deployment.id
  policy = data.aws_iam_policy_document.deployment.json
}

resource "aws_sfn_state_machine" "deploy" {
  name     = "${var.name}-deploy"
  role_arn = aws_iam_role.deployment.arn
  type     = "STANDARD"

  definition = jsonencode({
    Comment        = "Run Sharecrop database migrations before rolling the application service"
    StartAt        = "Run database migrations"
    TimeoutSeconds = var.deployment_timeout_seconds
    States = {
      "Run database migrations" = {
        Type     = "Task"
        Resource = "arn:aws:states:::ecs:runTask.sync"
        Parameters = {
          Cluster        = local.ecs_cluster_arn
          TaskDefinition = aws_ecs_task_definition.migrate.arn
          LaunchType     = "FARGATE"
          NetworkConfiguration = {
            AwsvpcConfiguration = {
              Subnets        = var.task_subnet_ids
              SecurityGroups = [aws_security_group.service.id]
              AssignPublicIp = "DISABLED"
            }
          }
        }
        ResultPath = "$.migration"
        Next       = "Roll application service"
      }
      "Roll application service" = {
        Type     = "Task"
        Resource = "arn:aws:states:::aws-sdk:ecs:updateService"
        Parameters = {
          Cluster            = local.ecs_cluster_arn
          Service            = aws_ecs_service.serve.name
          TaskDefinition     = aws_ecs_task_definition.serve.arn
          DesiredCount       = var.desired_count
          ForceNewDeployment = true
        }
        ResultPath = "$.deployment"
        Next       = "Wait for service"
      }
      "Wait for service" = {
        Type    = "Wait"
        Seconds = 15
        Next    = "Inspect service"
      }
      "Inspect service" = {
        Type     = "Task"
        Resource = "arn:aws:states:::aws-sdk:ecs:describeServices"
        Parameters = {
          Cluster  = local.ecs_cluster_arn
          Services = [aws_ecs_service.serve.name]
        }
        ResultPath = "$.service"
        Next       = "Service healthy"
      }
      "Service healthy" = {
        Type = "Choice"
        Choices = [
          {
            And = [
              {
                Variable     = "$.service.Services[0].TaskDefinition"
                StringEquals = aws_ecs_task_definition.serve.arn
              },
              {
                Variable      = "$.service.Services[0].RunningCount"
                NumericEquals = var.desired_count
              },
              {
                Variable      = "$.service.Services[0].PendingCount"
                NumericEquals = 0
              },
              {
                Variable     = "$.service.Services[0].Deployments[0].RolloutState"
                StringEquals = "COMPLETED"
              },
              {
                Variable  = "$.service.Services[0].Deployments[1]"
                IsPresent = false
              },
            ]
            Next = "Deployment complete"
          },
          {
            Variable     = "$.service.Services[0].Deployments[0].RolloutState"
            StringEquals = "FAILED"
            Next         = "Deployment failed"
          },
        ]
        Default = "Wait for service"
      }
      "Deployment complete" = {
        Type = "Succeed"
      }
      "Deployment failed" = {
        Type  = "Fail"
        Error = "SharecropDeploymentFailed"
        Cause = "The Amazon ECS deployment circuit breaker rolled back the Sharecrop service."
      }
    }
  })

  depends_on = [aws_iam_role_policy.deployment]
  tags       = local.tags
}

data "aws_iam_policy_document" "scheduler_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["scheduler.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "aws:SourceAccount"
      values   = [data.aws_caller_identity.current.account_id]
    }

    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values = [
        "arn:${data.aws_partition.current.partition}:scheduler:${var.region}:${data.aws_caller_identity.current.account_id}:schedule/default/${var.name}-deploy",
      ]
    }
  }
}

resource "aws_iam_role" "scheduler" {
  name               = "${var.name}-scheduler"
  assume_role_policy = data.aws_iam_policy_document.scheduler_assume.json
  tags               = local.tags
}

data "aws_iam_policy_document" "scheduler" {
  statement {
    actions   = ["states:StartExecution"]
    resources = [aws_sfn_state_machine.deploy.arn]
  }
}

resource "aws_iam_role_policy" "scheduler" {
  role   = aws_iam_role.scheduler.id
  policy = data.aws_iam_policy_document.scheduler.json
}

resource "aws_scheduler_schedule" "deploy" {
  name                         = "${var.name}-deploy"
  description                  = "Run database migrations, then roll the Sharecrop service"
  schedule_expression          = "at(${formatdate("YYYY-MM-DD'T'hh:mm:ss", timeadd(timestamp(), "2m"))})"
  schedule_expression_timezone = "UTC"
  action_after_completion      = "NONE"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = aws_sfn_state_machine.deploy.arn
    role_arn = aws_iam_role.scheduler.arn
    input    = "{}"

    retry_policy {
      maximum_event_age_in_seconds = 60
      maximum_retry_attempts       = 0
    }
  }

  lifecycle {
    ignore_changes = [schedule_expression]
    replace_triggered_by = [
      aws_sfn_state_machine.deploy,
    ]
  }

  depends_on = [aws_iam_role_policy.scheduler]
}
