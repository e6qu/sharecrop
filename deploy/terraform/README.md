# Sharecrop on Amazon ECS Fargate — Terraform

Provisions the production backend from [docs/deployment.md](../../docs/deployment.md):
an Amazon API Gateway HTTP API with an AWS Cloud Map private integration, an
Amazon ECS Fargate service (arm64) running the container image, a
tenant-specific connection to the shared fck-rds PostgreSQL service, and the
AWS Secrets Manager entries the service reads. It deploys into an **existing
VPC** and creates no Application Load Balancer or Network Load Balancer.

## What it creates

- **Private ingress:** an Amazon API Gateway HTTP API and regional custom domain
  reach the service through a VPC Link. The integration resolves healthy task
  addresses and ports from an AWS Cloud Map SRV service. The public execute-api
  endpoint is disabled.
- **Bounded traffic:** the default route has explicit steady-state and burst
  throttles. API access logs and detailed Amazon CloudWatch metrics are enabled.
- **Amazon ECS:** either a dedicated cluster or the supplied shared cluster,
  the `sharecrop-serve` service (`desired_count` replicas, arm64), and a one-off
  `sharecrop-migrate` task definition (`migrate up`). Tasks always run in the
  supplied private subnets without public IP addresses.
- **Health:** the distroless image runs its own binary as an Amazon ECS
  container health check. Amazon ECS publishes health to AWS Cloud Map, and
  Amazon API Gateway distributes requests only across healthy instances.
  Terraform waits for steady state, and the Amazon ECS deployment circuit
  breaker rolls back an unhealthy rollout.
- **Database:** a tenant-specific fck-rds PostgreSQL URL; fck-rds owns the
  database and role and permits the Sharecrop task security group.
- **Secrets:** generated `SHARECROP_ACCESS_TOKEN_SECRET` plus the supplied
  `DATABASE_URL` secret, injected into the task as Amazon ECS secrets.
- **IAM and security groups:** only the Amazon API Gateway VPC Link security
  group can reach the task's HTTP port. Task egress reaches the registry,
  PostgreSQL, and required AWS APIs through the VPC's existing egress path.

## Usage

```sh
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform apply

# Run migrations once before the first rollout and on every schema change.
eval "$(terraform output -raw run_migrate_command)"

# Create a Route 53 alias for `domain_name` with these exact module outputs.
terraform output api_gateway_domain_name
terraform output api_gateway_hosted_zone_id

# These outputs identify the application and ingress log groups for monitoring.
terraform output serve_log_group_name
terraform output api_gateway_access_log_group_name
```

Deploying a new version means setting `image` to the immutable 12-character
commit-SHA manifest published by the [Release workflow](../../.github/workflows/release.yml),
running the migration task when the release contains migrations, and applying
Terraform.

## Notes

- **Private image:** if the GitHub Container Registry package is private, put a
  `{"username","password"}` secret in AWS Secrets Manager and set
  `image_pull_secret_arn`. A public package needs nothing.
- **Egress:** private tasks and the VPC Link share `task_subnet_ids`. Those task
  subnets need the environment's existing egress path to pull the image and
  reach Shauth; the module never assigns public task IP addresses.
- **TLS and DNS:** supply the regional AWS Certificate Manager certificate ARN
  and exact `domain_name`. The caller owns DNS validation and creates the Route
  53 alias from the two Amazon API Gateway outputs.
- **Shared cluster:** set `existing_ecs_cluster_arn` to use an existing Amazon
  ECS cluster. Leave it unset to create a dedicated cluster.
- **Database capacity:** fck-rds owns shared PostgreSQL capacity and Amazon
  Elastic File System storage at the environment level; this module receives
  only Sharecrop's scoped database URL secret.
- The `deploy/ecs/*.task-definition.json` files are standalone references for a
  non-Terraform deploy; this module defines its own task definitions.
