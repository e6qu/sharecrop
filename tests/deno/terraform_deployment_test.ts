const read = (path: string): Promise<string> => Deno.readTextFile(path);

const assert = (condition: boolean, message: string): void => {
  if (!condition) {
    throw new Error(message);
  }
};

const assertMatch = (source: string, pattern: RegExp): void => {
  assert(pattern.test(source), `source did not match ${pattern}`);
};

Deno.test("Terraform orders a successful standalone migration before service rollout", async () => {
  const deployment = await read("deploy/terraform/deployment.tf");
  const ecs = await read("deploy/terraform/ecs.tf");
  const outputs = await read("deploy/terraform/outputs.tf");

  const migrate = deployment.indexOf(
    'Resource = "arn:aws:states:::ecs:runTask.sync"',
  );
  const rollout = deployment.indexOf(
    'Resource = "arn:aws:states:::aws-sdk:ecs:updateService"',
  );
  assert(migrate >= 0, "the workflow must run a standalone Amazon ECS task");
  assert(
    rollout > migrate,
    "the service update must follow the synchronous migration task",
  );
  assertMatch(
    deployment,
    /"Run database migrations"\s*=\s*\{[\s\S]*?Next\s*=\s*"Roll application service"/,
  );
  assertMatch(
    deployment,
    /aws_scheduler_schedule"\s+"deploy"[\s\S]*?schedule_expression\s*=\s*"at\(/,
  );
  assertMatch(
    deployment,
    /maximum_retry_attempts\s*=\s*0/,
  );
  assertMatch(
    deployment,
    /replace_triggered_by\s*=\s*\[[\s\S]*?aws_sfn_state_machine\.deploy,/,
  );
  assertMatch(
    deployment,
    /Variable\s*=\s*"\$\.service\.Services\[0\]\.Deployments\[1\]"[\s\S]*?IsPresent\s*=\s*false/,
  );

  assertMatch(
    ecs,
    /resource "aws_ecs_service" "serve"[\s\S]*?desired_count\s*=\s*0/,
  );
  assertMatch(
    ecs,
    /ignore_changes\s*=\s*\[[\s\S]*?desired_count,[\s\S]*?task_definition,/,
  );
  assert(
    !outputs.includes("run_migrate_command"),
    "deployment must not depend on an operator evaluating a generated AWS CLI command",
  );
});

Deno.test("Terraform reuses an existing Amazon API Gateway VPC Link when supplied", async () => {
  const variables = await read("deploy/terraform/variables.tf");
  const gateway = await read("deploy/terraform/gateway.tf");
  const network = await read("deploy/terraform/network.tf");

  assertMatch(
    variables,
    /variable "create_api_gateway_vpc_link"[\s\S]*?default\s*=\s*true/,
  );
  assertMatch(
    variables,
    /variable "existing_api_gateway_vpc_link_id"[\s\S]*?default\s*=\s*""/,
  );
  assertMatch(
    variables,
    /variable "existing_api_gateway_vpc_link_security_group_id"[\s\S]*?default\s*=\s*""/,
  );
  assertMatch(
    gateway,
    /resource "aws_apigatewayv2_vpc_link" "this"[\s\S]*?count\s*=\s*var\.create_api_gateway_vpc_link \? 1 : 0/,
  );
  assertMatch(
    network,
    /resource "aws_security_group" "api_gateway_vpc_link"[\s\S]*?count\s*=\s*var\.create_api_gateway_vpc_link \? 1 : 0/,
  );
  assertMatch(
    gateway,
    /api_gateway_vpc_link_id\s*=\s*var\.create_api_gateway_vpc_link \? aws_apigatewayv2_vpc_link\.this\[0\]\.id : var\.existing_api_gateway_vpc_link_id/,
  );
  assertMatch(
    network,
    /resource "aws_vpc_security_group_ingress_rule" "service_from_api_gateway"[\s\S]*?for_each\s*=\s*local\.api_gateway_vpc_link_security_groups[\s\S]*?referenced_security_group_id\s*=\s*each\.value/,
  );
  assertMatch(
    gateway,
    /api_gateway_vpc_link_security_groups\s*=\s*var\.create_api_gateway_vpc_link\s*\?\s*\{[\s\S]*?managed\s*=\s*aws_security_group\.api_gateway_vpc_link\[0\]\.id[\s\S]*?\}\s*:\s*\{[\s\S]*?shared\s*=\s*var\.existing_api_gateway_vpc_link_security_group_id/,
  );
  assertMatch(
    gateway,
    /Dedicated VPC Link mode rejects existing_api_gateway_vpc_link_id and existing_api_gateway_vpc_link_security_group_id/,
  );
  assertMatch(
    gateway,
    /Shared VPC Link mode requires both existing_api_gateway_vpc_link_id and existing_api_gateway_vpc_link_security_group_id/,
  );
  assert(
    !gateway.includes('data "aws_apigatewayv2_vpc_link"'),
    "shared resource-derived coordinates must not be read through a plan-time data lookup",
  );
  assert(
    !gateway.includes("uses_existing_api_gateway_vpc_link"),
    "resource ownership must not be inferred from a possibly unknown ID",
  );
  assert(
    [...gateway.matchAll(/resource "aws_apigatewayv2_vpc_link"/g)].length ===
      1,
    "the default standalone VPC Link remains the only managed VPC Link resource",
  );
});

Deno.test("real-provider wrapper passes unknown shared link coordinates with plan-known ownership", async () => {
  const wrapper = await read(
    "tests/terraform/shared-vpc-link-wrapper/main.tf",
  );

  assertMatch(
    wrapper,
    /resource "aws_apigatewayv2_vpc_link" "shared"/,
  );
  assertMatch(
    wrapper,
    /create_api_gateway_vpc_link\s*=\s*false/,
  );
  assertMatch(
    wrapper,
    /existing_api_gateway_vpc_link_id\s*=\s*aws_apigatewayv2_vpc_link\.shared\.id/,
  );
  assertMatch(
    wrapper,
    /existing_api_gateway_vpc_link_security_group_id\s*=\s*aws_security_group\.shared_api_gateway_vpc_link\.id/,
  );
});

Deno.test("Terraform publishes the application root through a complete default route", async () => {
  const gateway = await read("deploy/terraform/gateway.tf");

  assertMatch(
    gateway,
    /resource "aws_apigatewayv2_integration" "this"[\s\S]*?integration_type\s*=\s*"HTTP_PROXY"[\s\S]*?request_parameters\s*=\s*\{[\s\S]*?"overwrite:path"\s*=\s*"\$request\.path"/,
  );
  assertMatch(
    gateway,
    /resource "aws_apigatewayv2_route" "this"[\s\S]*?route_key\s*=\s*"\$default"[\s\S]*?target\s*=\s*"integrations\/\$\{aws_apigatewayv2_integration\.this\.id\}"/,
  );
  assertMatch(
    gateway,
    /resource "aws_apigatewayv2_stage" "this"[\s\S]*?name\s*=\s*"\$default"[\s\S]*?auto_deploy\s*=\s*true[\s\S]*?depends_on\s*=\s*\[aws_apigatewayv2_route\.this\]/,
  );
  assertMatch(
    gateway,
    /resource "aws_apigatewayv2_api_mapping" "this"[\s\S]*?stage\s*=\s*aws_apigatewayv2_stage\.this\.name/,
  );
});

Deno.test("Terraform uses stable IAM role names without provider-appended prefix limits", async () => {
  const iam = await read("deploy/terraform/iam.tf");
  const deployment = await read("deploy/terraform/deployment.tf");
  const roles = iam + deployment;

  assert(
    !roles.includes("name_prefix"),
    "IAM roles must not depend on provider-appended name-prefix suffixes",
  );
  for (const suffix of ["exec", "task", "deploy", "scheduler"]) {
    assertMatch(
      roles,
      new RegExp(`name\\s*=\\s*"\\$\\{var\\.name\\}-${suffix}"`),
    );
  }
});
