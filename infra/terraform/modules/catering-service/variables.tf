# Input parameters for a production-like deploy. This module is illustrative (`terraform plan`
# only); wire real subnets, security groups, and IAM roles before any apply.

variable "project" {
  type        = string
  description = "Short name prefix for resources (e.g. plateful)."
  default     = "plateful"
}

variable "vpc_id" {
  type        = string
  description = "VPC hosting ECS tasks and RDS."
}

variable "private_subnet_ids" {
  type        = list(string)
  description = "Private subnets for Fargate tasks and RDS subnet group (typically >= 2 AZs)."
}

variable "ecs_security_group_ids" {
  type        = list(string)
  description = "Security groups for ECS tasks (ingress from ALB or internal mesh)."
}

variable "db_security_group_ids" {
  type        = list(string)
  description = "Security groups for RDS (ingress tcp/5432 from ECS SGs only, in production)."
}

variable "ecs_execution_role_arn" {
  type        = string
  description = "Task execution role (ECR pull, CloudWatch logs). Replace placeholder before apply."
}

variable "ecs_task_role_arn" {
  type        = string
  description = "Task role for app (SQS send, Secrets Manager). Replace placeholder before apply."
}

variable "container_image" {
  type        = string
  description = "ECR image URI for the API container."
}

variable "db_password" {
  type        = string
  sensitive   = true
  description = "RDS master password (use Secrets Manager + random_password in production)."
}

variable "db_instance_class" {
  type        = string
  default     = "db.t3.micro"
  description = "Size is UNSPECIFIED for real workloads; micro is a dev default."
}

variable "sqs_visibility_timeout_seconds" {
  type        = number
  default     = 60
  description = "Should be >= worst-case worker processing time."
}
