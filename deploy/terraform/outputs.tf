output "alb_dns_name" {
  description = "Public DNS name of the load balancer (point your domain's DNS at this)."
  value       = aws_lb.this.dns_name
}

output "alb_zone_id" {
  description = "Canonical hosted-zone ID of the Application Load Balancer, for a Route 53 alias record."
  value       = aws_lb.this.zone_id
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
  description = "Family of the one-off migration task. Run it before the first deploy and on every schema change."
  value       = aws_ecs_task_definition.migrate.family
}

output "run_migrate_command" {
  description = "Example command to run the one-off migration task."
  value = format(
    "aws ecs run-task --cluster %s --task-definition %s --launch-type FARGATE --network-configuration 'awsvpcConfiguration={subnets=[%s],securityGroups=[%s],assignPublicIp=%s}'",
    local.ecs_cluster_name,
    aws_ecs_task_definition.migrate.family,
    join(",", var.task_subnet_ids),
    aws_security_group.service.id,
    var.assign_public_ip ? "ENABLED" : "DISABLED",
  )
}

output "service_security_group_id" {
  description = "Security group attached to Sharecrop Amazon ECS tasks."
  value       = aws_security_group.service.id
}
