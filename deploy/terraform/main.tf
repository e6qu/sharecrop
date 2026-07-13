provider "aws" {
  region = var.region
}

locals {
  tags = merge({
    Application = var.name
    ManagedBy   = "terraform"
  }, var.tags)
}
