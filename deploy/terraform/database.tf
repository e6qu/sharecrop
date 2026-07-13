# Aurora Serverless v2 (PostgreSQL) fronted by RDS Proxy for connection pooling.

resource "aws_db_subnet_group" "this" {
  name_prefix = "${var.name}-"
  subnet_ids  = var.task_subnet_ids
  tags        = local.tags
}

resource "aws_rds_cluster" "this" {
  cluster_identifier = var.name
  engine             = "aurora-postgresql"
  engine_mode        = "provisioned" # required for Serverless v2
  engine_version     = var.aurora_engine_version

  database_name   = var.database_name
  master_username = var.database_username
  master_password = random_password.database.result

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.database.id]

  serverlessv2_scaling_configuration {
    min_capacity = var.aurora_min_capacity
    max_capacity = var.aurora_max_capacity
  }

  storage_encrypted   = true
  skip_final_snapshot = true

  tags = local.tags
}

resource "aws_rds_cluster_instance" "this" {
  identifier         = "${var.name}-1"
  cluster_identifier = aws_rds_cluster.this.id
  instance_class     = "db.serverless"
  engine             = aws_rds_cluster.this.engine
  engine_version     = aws_rds_cluster.this.engine_version
  tags               = local.tags
}

# IAM role that lets RDS Proxy read the database credentials secret.
data "aws_iam_policy_document" "rds_proxy_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["rds.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "rds_proxy" {
  name_prefix        = "${var.name}-rdsproxy-"
  assume_role_policy = data.aws_iam_policy_document.rds_proxy_assume.json
  tags               = local.tags
}

data "aws_iam_policy_document" "rds_proxy_secret" {
  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.database_credentials.arn]
  }
}

resource "aws_iam_role_policy" "rds_proxy_secret" {
  role   = aws_iam_role.rds_proxy.id
  policy = data.aws_iam_policy_document.rds_proxy_secret.json
}

resource "aws_db_proxy" "this" {
  name                   = var.name
  engine_family          = "POSTGRESQL"
  role_arn               = aws_iam_role.rds_proxy.arn
  vpc_subnet_ids         = var.task_subnet_ids
  vpc_security_group_ids = [aws_security_group.rds_proxy.id]
  require_tls            = true

  auth {
    auth_scheme = "SECRETS"
    iam_auth    = "DISABLED"
    secret_arn  = aws_secretsmanager_secret.database_credentials.arn
  }

  tags = local.tags
}

resource "aws_db_proxy_default_target_group" "this" {
  db_proxy_name = aws_db_proxy.this.name
}

resource "aws_db_proxy_target" "this" {
  db_proxy_name         = aws_db_proxy.this.name
  target_group_name     = aws_db_proxy_default_target_group.this.name
  db_cluster_identifier = aws_rds_cluster.this.cluster_identifier
}

# DATABASE_URL, assembled from the RDS Proxy endpoint (the task reads this via
# `secrets`). sslmode=require because RDS Proxy enforces TLS.
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
    aws_db_proxy.this.endpoint,
    var.database_name,
  )
}
