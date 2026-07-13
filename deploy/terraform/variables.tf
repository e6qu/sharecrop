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

# Networking: deploy into an existing VPC.
variable "vpc_id" {
  description = "VPC to deploy into."
  type        = string
}

variable "public_subnet_ids" {
  description = "Public subnets for the internet-facing load balancer (>= 2 AZs)."
  type        = list(string)
}

variable "task_subnet_ids" {
  description = "Subnets for the ECS tasks, RDS Proxy, and Aurora. Use private subnets with a NAT gateway (so tasks can pull the image); or public subnets with assign_public_ip = true."
  type        = list(string)
}

variable "assign_public_ip" {
  description = "Give ECS tasks a public IP (required only if task_subnet_ids are public subnets without a NAT gateway)."
  type        = bool
  default     = false
}

variable "certificate_arn" {
  description = "Optional ACM certificate ARN. When set, the ALB serves HTTPS on 443 and redirects 80 -> 443; when null, it serves plain HTTP on 80."
  type        = string
  default     = null
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
  description = "Aurora database name."
  type        = string
  default     = "sharecrop"
}

variable "database_username" {
  description = "Aurora master username."
  type        = string
  default     = "sharecrop"
}

variable "aurora_min_capacity" {
  description = "Aurora Serverless v2 minimum ACUs. 0 enables scale-to-zero (requires a supported engine version)."
  type        = number
  default     = 0
}

variable "aurora_max_capacity" {
  description = "Aurora Serverless v2 maximum ACUs."
  type        = number
  default     = 4
}

variable "aurora_engine_version" {
  description = "Aurora PostgreSQL engine version. Use a version that supports 0 minimum ACUs for scale-to-zero."
  type        = string
  default     = "16.4"
}

variable "tags" {
  description = "Extra tags applied to all resources."
  type        = map(string)
  default     = {}
}
