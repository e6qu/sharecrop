variable "name" {
  description = "Name prefix for all resources."
  type        = string
  default     = "sharecrop"
}

variable "region" {
  description = "AWS region."
  type        = string
}

variable "image" {
  description = "Immutable container image reference to run (the multi-architecture manifest), e.g. ghcr.io/e6qu/sharecrop:0123456789ab. AWS Fargate pulls the arm64 variant."
  type        = string
}

variable "image_pull_secret_arn" {
  description = "Optional Secrets Manager ARN holding {\"username\",\"password\"} for pulling a PRIVATE ghcr image. Leave null for a public image."
  type        = string
  default     = null
}

variable "shauth_oidc_issuer" {
  description = "Shauth HTTPS OpenID Connect issuer. Set every Shauth coordinate together."
  type        = string
  default     = ""
}
variable "shauth_oidc_client_id" {
  description = "Shauth confidential client ID."
  type        = string
  default     = ""
}
variable "shauth_oidc_client_secret_arn" {
  description = "AWS Secrets Manager ARN containing the Shauth confidential-client secret."
  type        = string
  default     = ""
}
variable "public_url" {
  description = "Canonical HTTPS public origin used to derive the OpenID Connect callback, Back-Channel Logout, and post-logout redirect URLs, e.g. https://sharecrop.dev.e6qu.dev."
  type        = string
  default     = ""
}

# Networking: deploy into an existing VPC.
variable "vpc_id" {
  description = "VPC to deploy into."
  type        = string
}

variable "existing_ecs_cluster_arn" {
  description = "Optional Amazon Elastic Container Service cluster ARN. When set, Sharecrop uses that existing cluster instead of creating one."
  type        = string
  default     = ""

  validation {
    condition     = var.existing_ecs_cluster_arn == "" || can(regex("^arn:[^:]+:ecs:[^:]+:[0-9]+:cluster/.+", var.existing_ecs_cluster_arn))
    error_message = "existing_ecs_cluster_arn must be empty or an Amazon Elastic Container Service cluster ARN."
  }
}

variable "task_subnet_ids" {
  description = "Private subnets for the Amazon ECS tasks and Amazon API Gateway VPC Link. Use subnets with outbound registry access through the environment's NAT path."
  type        = list(string)

  validation {
    condition     = length(var.task_subnet_ids) >= 2
    error_message = "task_subnet_ids must contain at least two private subnets."
  }
}

variable "certificate_arn" {
  description = "AWS Certificate Manager certificate ARN for the regional Amazon API Gateway custom domain."
  type        = string

  validation {
    condition     = can(regex("^arn:[^:]+:acm:[^:]+:[0-9]+:certificate/.+", var.certificate_arn))
    error_message = "certificate_arn must be an AWS Certificate Manager certificate ARN."
  }
}

variable "domain_name" {
  description = "Public DNS name bound to the regional Amazon API Gateway HTTP API, e.g. sharecrop.dev.e6qu.dev."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$", var.domain_name))
    error_message = "domain_name must be a lowercase DNS name."
  }
}

variable "api_throttling_burst_limit" {
  description = "Maximum burst requests admitted by the Amazon API Gateway default route."
  type        = number
  default     = 50

  validation {
    condition     = var.api_throttling_burst_limit >= 1 && floor(var.api_throttling_burst_limit) == var.api_throttling_burst_limit
    error_message = "api_throttling_burst_limit must be a positive integer."
  }
}

variable "api_throttling_rate_limit" {
  description = "Steady requests per second admitted by the Amazon API Gateway default route."
  type        = number
  default     = 25

  validation {
    condition     = var.api_throttling_rate_limit > 0
    error_message = "api_throttling_rate_limit must be greater than zero."
  }
}

# Compute.
variable "desired_count" {
  description = "Number of serve replicas."
  type        = number
  default     = 2
}

variable "cpu" {
  description = "Fargate task CPU units for the serve task."
  type        = string
  default     = "512"
}

variable "memory" {
  description = "Fargate task memory (MiB) for the serve task."
  type        = string
  default     = "1024"
}

variable "database_url_secret_arn" {
  description = "AWS Secrets Manager ARN containing Sharecrop's tenant-specific PostgreSQL URL from fck-rds."
  type        = string
}

variable "tags" {
  description = "Extra tags applied to all resources."
  type        = map(string)
  default     = {}
}
