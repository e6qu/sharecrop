# Sharecrop on ECS Fargate — Terraform

Provisions the production backend from [docs/deployment.md](../../docs/deployment.md):
an internet-facing ALB, an ECS Fargate service (arm64) running the container
image, Aurora Serverless v2 (PostgreSQL) behind RDS Proxy, and the Secrets
Manager entries the service reads. It deploys into an **existing VPC** (you
provide the VPC and subnet ids); it does not create networking.

## What it creates

- **ALB** (public) → target group health-checked on `/healthz` → the serve
  service. Plain HTTP on `:80`, or HTTPS on `:443` (+ 80→443 redirect) when
  `enable_https = true` and `certificate_arn` is supplied. Set
  `enable_https` explicitly when the certificate is created in the same apply.
- **ECS**: either a dedicated cluster or the supplied shared Amazon Elastic
  Container Service cluster, the `sharecrop-serve` service (`desired_count`
  replicas, arm64), and a one-off `sharecrop-migrate` task definition
  (`migrate up`).
- **Database**: an Aurora Serverless v2 PostgreSQL cluster (scale-to-zero when
  `aurora_min_capacity = 0`) fronted by RDS Proxy for pooling.
- **Secrets**: a generated `SHARECROP_ACCESS_TOKEN_SECRET`, the database
  credentials, and the assembled `DATABASE_URL` (pointing at the RDS Proxy,
  `sslmode=require`) — injected into the task as `secrets`.
- **IAM + security groups** wiring internet → ALB → tasks → RDS Proxy → Aurora.

## Usage

```sh
cp terraform.tfvars.example terraform.tfvars   # fill in region, image, vpc/subnets
terraform init
terraform apply

# Run migrations once before the first rollout (and on every schema change).
# `terraform output run_migrate_command` prints the exact aws-cli command.
eval "$(terraform output -raw run_migrate_command)"

# Point your domain at the load balancer:
terraform output alb_dns_name
# Use `alb_zone_id` with the DNS name when creating a Route 53 alias record.
# `serve_log_group_name` identifies the application log group for monitoring.
```

Deploying a new version = the [Release workflow](../../.github/workflows/release.yml)
publishes `ghcr.io/<owner>/<repo>:<version>`; set `image` to that tag and
`terraform apply` (or update the service to the new task definition), running the
migrate task first if the release includes migrations.

## Notes

- **Private image:** if the ghcr package is private, put a
  `{"username","password"}` secret in Secrets Manager and set
  `image_pull_secret_arn`. A public package needs nothing.
- **Egress:** tasks must reach the internet to pull the image. Use private
  `task_subnet_ids` with a NAT gateway (`assign_public_ip = false`), or public
  subnets with `assign_public_ip = true`.
- **Shared cluster:** set `existing_ecs_cluster_arn` to use an existing Amazon
  Elastic Container Service cluster. Leave it unset to create a dedicated
  cluster.
- **Scale-to-zero:** `aurora_min_capacity = 0` requires an engine version that
  supports it; bump `aurora_engine_version` if your region rejects it.
- The `deploy/ecs/*.task-definition.json` files are standalone references for a
  non-Terraform deploy; this module defines its own task definitions.
