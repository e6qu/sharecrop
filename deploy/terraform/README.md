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
  the `sharecrop-serve` service (`desired_count` replicas, arm64), and a
  standalone `sharecrop-migrate` task definition (`migrate up`). Tasks always
  run in the supplied private subnets without public IP addresses.
- **Ordered deployment:** Amazon EventBridge Scheduler starts one AWS Step
  Functions execution when the application workflow definition changes. The
  workflow runs the standalone migration task with `ecs:runTask.sync`, updates
  the service only after that task succeeds, and then waits for the service's
  healthy Amazon ECS deployment. A failed migration never rolls the service.
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

# The one-time deployment schedule starts the migration-and-rollout workflow.
# Inspect its real AWS Step Functions execution until it succeeds.
aws stepfunctions list-executions \
  --state-machine-arn "$(terraform output -raw deployment_state_machine_arn)" \
  --max-results 1

# Create a Route 53 alias for `domain_name` with these exact module outputs.
terraform output api_gateway_domain_name
terraform output api_gateway_hosted_zone_id

# These outputs identify the application and ingress log groups for monitoring.
terraform output serve_log_group_name
terraform output api_gateway_access_log_group_name
```

Deploying a new version means setting `image` to the immutable 12-character
commit-SHA manifest published by the [Release workflow](../../.github/workflows/release.yml)
and setting `release_revision` to that same commit SHA
and applying Terraform. Terraform creates a one-time EventBridge Scheduler
schedule for the changed workflow. AWS Step Functions waits for the standalone
migration task, updates the Amazon ECS service only on success, and waits for
the circuit-breaker-protected rollout. The schedule is retained after its one
invocation so a later Terraform refresh does not recreate and rerun it.

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
- **Shared VPC Link:** set `create_api_gateway_vpc_link = false` and supply both
  `existing_api_gateway_vpc_link_id` and
  `existing_api_gateway_vpc_link_security_group_id`. The explicit boolean is
  plan-known even when both IDs come from resources in an environment wrapper;
  this module then creates only the Sharecrop-specific ingress and egress
  rules. The default `true` creates a dedicated VPC Link and security group and
  rejects external link coordinates.
- **Composition contract:**
  [`tests/terraform/shared-vpc-link-wrapper`](../../tests/terraform/shared-vpc-link-wrapper/)
  creates an environment-owned link and security group and passes their
  unknown-until-apply IDs into this module. Its real-provider plan must succeed
  without unknown `count` or `for_each` keys.
- **Deployment completion:** `terraform apply` creates the one-time schedule;
  the AWS Step Functions execution is asynchronous to Terraform and is the
  authoritative deployment result. It has a bounded
  `deployment_timeout_seconds` and remains failed if the migration task or
  Amazon ECS rollout fails.
- **Database capacity:** fck-rds owns shared PostgreSQL capacity and Amazon
  Elastic File System storage at the environment level; this module receives
  only Sharecrop's scoped database URL secret.
- The `deploy/ecs/*.task-definition.json` files are standalone references for a
  non-Terraform deploy; this module defines its own task definitions.
