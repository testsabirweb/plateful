# Catering / Plateful — illustrative AWS module (ECS Fargate + SQS + RDS Postgres).
#
# Omitted before production: IAM policies, ALB + TLS, Secrets Manager, VPC endpoints,
# RDS backup/Multi-AZ, autoscaling, DLQ, observability, WAF.

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

resource "aws_sqs_queue" "order_events" {
  name                       = "${var.project}-order-events"
  visibility_timeout_seconds = var.sqs_visibility_timeout_seconds
}

resource "aws_db_subnet_group" "app" {
  name       = "${var.project}-db-subnets"
  subnet_ids = var.private_subnet_ids
}

resource "aws_db_instance" "app" {
  identifier              = "${var.project}-postgres"
  engine                  = "postgres"
  engine_version          = "16"
  instance_class          = var.db_instance_class
  allocated_storage       = 20
  db_subnet_group_name    = aws_db_subnet_group.app.name
  username                = "plateful"
  password                = var.db_password
  vpc_security_group_ids  = var.db_security_group_ids
  skip_final_snapshot     = true
  publicly_accessible     = false
  backup_retention_period = 0
  deletion_protection     = false
}

resource "aws_ecs_cluster" "app" {
  name = "${var.project}-cluster"
}

resource "aws_ecs_task_definition" "api" {
  family                   = "${var.project}-api"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = var.ecs_execution_role_arn
  task_role_arn            = var.ecs_task_role_arn

  container_definitions = jsonencode([
    {
      name      = "api"
      image     = var.container_image
      essential = true
      portMappings = [
        { containerPort = 8080, protocol = "tcp" }
      ]
    }
  ])
}

resource "aws_ecs_service" "api" {
  name            = "${var.project}-api"
  cluster         = aws_ecs_cluster.app.id
  task_definition = aws_ecs_task_definition.api.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = var.ecs_security_group_ids
    assign_public_ip = false
  }
}
