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
  description = "Container image reference to run (the multi-arch manifest), e.g. ghcr.io/e6qu/sharecrop:v1.4.0. Fargate pulls the arm64 variant."
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
  description = "Canonical HTTPS public URL used for OIDC callbacks, e.g. https://sharecrop.dev.e6qu.dev."
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

variable "public_subnet_ids" {
  description = "Public subnets for the internet-facing load balancer (>= 2 AZs)."
  type        = list(string)
}

variable "task_subnet_ids" {
  description = "Subnets for the ECS tasks and Amazon RDS for PostgreSQL. Use private subnets with a NAT gateway (so tasks can pull the image); or public subnets with assign_public_ip = true."
  type        = list(string)
}

variable "assign_public_ip" {
  description = "Give ECS tasks a public IP (required only if task_subnet_ids are public subnets without a NAT gateway)."
  type        = bool
  default     = false
}

variable "certificate_arn" {
  description = "ACM certificate ARN used by the HTTPS listener when enable_https is true."
  type        = string
  default     = null
}

variable "enable_https" {
  description = "Whether to create the HTTPS listener and redirect HTTP to HTTPS. This must be known while Terraform plans, so callers provisioning a certificate in the same apply set it explicitly."
  type        = bool
  default     = false

  validation {
    condition     = !var.enable_https || var.certificate_arn != null
    error_message = "certificate_arn must be set when enable_https is true."
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

# Database.
variable "database_name" {
  description = "PostgreSQL database name."
  type        = string
  default     = "sharecrop"
}

variable "database_username" {
  description = "PostgreSQL master username."
  type        = string
  default     = "sharecrop"
}

variable "postgres_engine_version" {
  description = "Amazon RDS for PostgreSQL engine version."
  type        = string
  default     = "18.3"
}

variable "postgres_instance_class" {
  description = "Amazon RDS DB instance class. db.t4g.micro is the smallest arm64 class available in eu-west-1."
  type        = string
  default     = "db.t4g.micro"
}

variable "postgres_allocated_storage_gib" {
  description = "Initial gp3 storage in GiB for the Amazon RDS PostgreSQL instance."
  type        = number
  default     = 20
}

variable "postgres_max_allocated_storage_gib" {
  description = "Maximum gp3 storage in GiB allowed by Amazon RDS storage autoscaling."
  type        = number
  default     = 100
}

variable "tags" {
  description = "Extra tags applied to all resources."
  type        = map(string)
  default     = {}
}
