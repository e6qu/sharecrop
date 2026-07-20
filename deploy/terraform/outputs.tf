output "api_gateway_domain_name" {
  description = "Regional Amazon API Gateway target domain for the public Route 53 alias."
  value       = aws_apigatewayv2_domain_name.this.domain_name_configuration[0].target_domain_name
}

output "api_gateway_hosted_zone_id" {
  description = "Regional Amazon API Gateway hosted-zone ID for the public Route 53 alias."
  value       = aws_apigatewayv2_domain_name.this.domain_name_configuration[0].hosted_zone_id
}

output "api_gateway_access_log_group_name" {
  description = "CloudWatch Logs group receiving Amazon API Gateway access logs."
  value       = aws_cloudwatch_log_group.api_access.name
}

output "ecs_cluster_name" {
  description = "ECS cluster name."
  value       = local.ecs_cluster_name
}

output "ecs_service_name" {
  description = "serve ECS service name."
  value       = aws_ecs_service.serve.name
}

output "serve_log_group_name" {
  description = "CloudWatch Logs group receiving Sharecrop serve-task logs."
  value       = aws_cloudwatch_log_group.serve.name
}

output "migrate_task_definition" {
  description = "Family of the standalone migration task run by the ordered deployment workflow."
  value       = aws_ecs_task_definition.migrate.family
}

output "deployment_state_machine_arn" {
  description = "AWS Step Functions state machine that migrates the database before rolling the Amazon ECS service."
  value       = aws_sfn_state_machine.deploy.arn
}

output "deployment_schedule_arn" {
  description = "One-time Amazon EventBridge Scheduler schedule for the current ordered deployment."
  value       = aws_scheduler_schedule.deploy.arn
}

output "service_security_group_id" {
  description = "Security group attached to Sharecrop Amazon ECS tasks."
  value       = aws_security_group.service.id
}
