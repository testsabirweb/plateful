output "queue_url" {
  description = "SQS queue URL for order events (publisher + worker consumers)."
  value       = aws_sqs_queue.order_events.url
}

output "queue_name" {
  description = "SQS queue name (use with AWS SDK or env)."
  value       = aws_sqs_queue.order_events.name
}

output "ecs_cluster_name" {
  description = "ECS cluster name for ops / CodeDeploy."
  value       = aws_ecs_cluster.app.name
}

output "ecs_service_name" {
  description = "ECS service running the API task."
  value       = aws_ecs_service.api.name
}

output "rds_endpoint" {
  description = "RDS hostname for DATABASE_URL (TLS recommended)."
  value       = aws_db_instance.app.endpoint
  sensitive   = false
}
