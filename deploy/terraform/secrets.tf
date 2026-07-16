# Generated secrets. The DATABASE_URL secret is assembled in database.tf once the
# Amazon RDS for PostgreSQL instance exists.

resource "random_password" "database" {
  length  = 32
  special = false # keep it URL-safe for DATABASE_URL and RDS
}

resource "random_password" "access_token" {
  length  = 64
  special = false
}

# SHARECROP_ACCESS_TOKEN_SECRET (the task reads this via `secrets`).
resource "aws_secretsmanager_secret" "access_token" {
  name_prefix = "${var.name}/access-token-secret-"
  tags        = local.tags
}

resource "aws_secretsmanager_secret_version" "access_token" {
  secret_id     = aws_secretsmanager_secret.access_token.id
  secret_string = random_password.access_token.result
}
