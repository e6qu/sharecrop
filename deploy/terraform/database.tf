# Single-AZ Amazon RDS for PostgreSQL keeps the dev database inexpensive and
# works on AWS accounts that do not permit Aurora or Amazon RDS Proxy.

resource "aws_db_subnet_group" "this" {
  name_prefix = "${var.name}-"
  subnet_ids  = var.task_subnet_ids
  tags        = local.tags
}

resource "aws_db_instance" "this" {
  identifier = var.name

  engine         = "postgres"
  engine_version = var.postgres_engine_version
  instance_class = var.postgres_instance_class

  db_name  = var.database_name
  username = var.database_username
  password = random_password.database.result
  port     = 5432

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.database.id]

  allocated_storage     = var.postgres_allocated_storage_gib
  max_allocated_storage = var.postgres_max_allocated_storage_gib
  storage_type          = "gp3"
  storage_encrypted     = true
  multi_az              = false
  publicly_accessible   = false

  backup_retention_period = 1
  skip_final_snapshot     = true
  deletion_protection     = false
  apply_immediately       = true

  tags = local.tags
}

# DATABASE_URL reaches the private database directly. ECS task deployment waits
# for this version, which also establishes the database creation dependency.
resource "aws_secretsmanager_secret" "database_url" {
  name_prefix = "${var.name}/database-url-"
  tags        = local.tags
}

resource "aws_secretsmanager_secret_version" "database_url" {
  secret_id = aws_secretsmanager_secret.database_url.id
  secret_string = format(
    "postgres://%s:%s@%s:5432/%s?sslmode=require",
    var.database_username,
    random_password.database.result,
    aws_db_instance.this.address,
    var.database_name,
  )
}
