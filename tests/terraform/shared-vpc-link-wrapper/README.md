# Shared VPC Link plan contract

This real-provider wrapper creates an environment-owned Amazon API Gateway VPC
Link and security group, then passes their unknown-until-apply IDs into the
Sharecrop module with `create_api_gateway_vpc_link = false`.

Run it with real AWS credentials and real non-secret coordinates:

```sh
cp terraform.tfvars.example terraform.tfvars
terraform init -backend=false
terraform plan
```

The plan must succeed without an invalid-count or invalid-for-each error. It is
a plan-only contract; do not apply the fixture.
