variable "region" {
  type = string
}

variable "name" {
  type    = string
  default = "sharecrop-wrapper-test"
}

variable "image" {
  type = string
}

variable "release_revision" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "task_subnet_ids" {
  type = list(string)
}

variable "existing_ecs_cluster_arn" {
  type    = string
  default = ""
}

variable "certificate_arn" {
  type = string
}

variable "domain_name" {
  type = string
}

variable "database_url_secret_arn" {
  type = string
}
