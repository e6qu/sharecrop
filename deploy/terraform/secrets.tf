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
