# Auto-loaded by Terraform for `terraform validate` / CI. Placeholder ARNs and password —
# do not apply to a real account without replacing variables via a proper tfvars + backend.

private_subnet_ids = ["subnet-aaaaaaaaaaaaaaaaa", "subnet-bbbbbbbbbbbbbbbbb"]
ecs_security_group_ids = ["sg-00000000000000000"]
db_security_group_ids = ["sg-00000000000000001"]
ecs_execution_role_arn = "arn:aws:iam::123456789012:role/ecsTaskExecutionRole"
ecs_task_role_arn      = "arn:aws:iam::123456789012:role/ecsTaskRole"
container_image        = "123456789012.dkr.ecr.us-east-1.amazonaws.com/plateful:latest"
db_password            = "ci-validate-not-for-production"
