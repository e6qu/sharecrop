type Violation = {
  path: string;
  message: string;
};

const rootDirectories: string[] = ["cmd", "internal", "tests", "tools", "web"];
const generatedSuffixes: string[] = [
  "/web/static/app.js",
  "/web/static/app.css",
];
const boundarySuffixes: string[] = [
  "/cmd/sharecrop-wasm/main_js_wasm.go",
  // The DB handle adapts pgx / database/sql, whose variadic argument and
  // scan-destination signatures are unavoidably weakly typed. This is the one
  // seam between the typed domain layer and the driver interfaces.
  "/internal/db/handle.go",
  "/internal/db/handle_sqlite.go",
  // StringArray implements the standard-library sql.Scanner interface, whose
  // Scan parameter is weakly typed by that interface.
  "/internal/db/stringarray.go",
  // sqlitex reaches the raw sqlite3.Conn through database/sql's Conn.Raw, whose
  // callback parameter is weakly typed by the standard library.
  "/internal/sqlitex/sqlitex.go",
];

const weakWildcardToken = "a" + "ny";
const weakStructuralToken = "obj" + "ect";
const absentValueToken = "nu" + "ll";

async function collectFiles(directory: string, files: string[]): Promise<void> {
  for await (const entry of Deno.readDir(directory)) {
    const path = `${directory}/${entry.name}`;
    if (entry.isDirectory) {
      await collectFiles(path, files);
      continue;
    }

    if (entry.isFile) {
      files.push(path);
    }
  }
}

function isSkipped(path: string): boolean {
  for (const suffix of generatedSuffixes) {
    if (path.endsWith(suffix.slice(1))) {
      return true;
    }
  }
  for (const suffix of boundarySuffixes) {
    if (path.endsWith(suffix.slice(1))) {
      return true;
    }
  }

  return false;
}

function checkGo(path: string, source: string, violations: Violation[]): void {
  if (/\bany\b/.test(source)) {
    violations.push({ path, message: "Go source used weak wildcard type" });
  }

  if (/interface\s*\{\s*\}/.test(source)) {
    violations.push({ path, message: "Go source used weak empty interface" });
  }

  if (path.startsWith("internal/core/") && /\bmap\s*\[/.test(source)) {
    violations.push({ path, message: "core domain source used generic map" });
  }

  if (path.startsWith("internal/core/") && /\sbool\b/.test(source)) {
    violations.push({
      path,
      message: "core domain source used bool field or return value",
    });
  }
}

function checkTypeScript(
  path: string,
  source: string,
  violations: Violation[],
): void {
  if (new RegExp(`\\b${weakWildcardToken}\\b`).test(source)) {
    violations.push({
      path,
      message: "TypeScript source used weak wildcard type",
    });
  }

  if (new RegExp(`\\b${weakStructuralToken}\\b`).test(source)) {
    violations.push({
      path,
      message: "TypeScript source used weak structural type",
    });
  }

  if (new RegExp(`\\b${absentValueToken}\\b`).test(source)) {
    violations.push({
      path,
      message: "TypeScript source used forbidden absent value",
    });
  }

  if (/[A-Za-z0-9_]\?\s*:/.test(source)) {
    violations.push({
      path,
      message: "TypeScript source used optional parameter or property",
    });
  }
}

function checkCoreImports(
  path: string,
  source: string,
  violations: Violation[],
): void {
  if (!path.startsWith("internal/core/")) {
    return;
  }

  const forbiddenImports: string[] = [
    "net/http",
    "github.com/jackc/pgx",
    "os",
    "database/sql",
  ];

  for (const forbiddenImport of forbiddenImports) {
    if (source.includes(`"${forbiddenImport}`)) {
      violations.push({
        path,
        message: `core domain imported ${forbiddenImport}`,
      });
    }
  }
}

const files: string[] = [];
for (const directory of rootDirectories) {
  await collectFiles(directory, files);
}

const violations: Violation[] = [];
for (const path of files) {
  if (isSkipped(path)) {
    continue;
  }

  const source = await Deno.readTextFile(path);
  if (path.endsWith(".go")) {
    checkGo(path, source, violations);
    checkCoreImports(path, source, violations);
  }

  if (path.endsWith(".ts")) {
    checkTypeScript(path, source, violations);
  }
}

const deploymentTerraformFiles: string[] = [];
await collectFiles("deploy/terraform", deploymentTerraformFiles);
let deploymentTerraform = "";
for (const path of deploymentTerraformFiles) {
  if (!path.endsWith(".tf")) {
    continue;
  }
  deploymentTerraform += `\n${await Deno.readTextFile(path)}`;
}

const forbiddenIngressResources: string[] = [
  "aws_lb",
  "aws_lb_listener",
  "aws_lb_target_group",
];
for (const resourceType of forbiddenIngressResources) {
  const declaration = new RegExp(
    `resource\\s+\"${resourceType}\"\\s+\"`,
  );
  if (declaration.test(deploymentTerraform)) {
    violations.push({
      path: "deploy/terraform",
      message:
        `${resourceType} is forbidden; public HTTP ingress uses Amazon API Gateway with an AWS Cloud Map private integration`,
    });
  }
}

for (
  const requiredResource of [
    "aws_apigatewayv2_vpc_link",
    "aws_apigatewayv2_api",
    "aws_service_discovery_service",
  ]
) {
  const declaration = new RegExp(
    `resource\\s+\"${requiredResource}\"\\s+\"`,
  );
  if (!declaration.test(deploymentTerraform)) {
    violations.push({
      path: "deploy/terraform",
      message: `private ingress is missing ${requiredResource}`,
    });
  }
}

for (
  const [pattern, message] of [
    [
      /connection_type\s*=\s*"VPC_LINK"/,
      "Amazon API Gateway integration must use a VPC Link",
    ],
    [
      /integration_uri\s*=\s*aws_service_discovery_service\./,
      "Amazon API Gateway integration must target AWS Cloud Map",
    ],
    [
      /health_check_custom_config\s*\{\s*\}/,
      "AWS Cloud Map must receive Amazon ECS-managed task health",
    ],
    [
      /service_registries\s*\{/,
      "Amazon ECS service must register task addresses and ports in AWS Cloud Map",
    ],
    [
      /"healthcheck",\s*"http:\/\/127\.0\.0\.1:8080\/healthz"/,
      "Amazon ECS task definition must probe the real Sharecrop health endpoint",
    ],
    [
      /disable_execute_api_endpoint\s*=\s*true/,
      "Amazon API Gateway's unmanaged execute-api endpoint must stay disabled",
    ],
    [
      /deployment_circuit_breaker\s*\{[\s\S]*?enable\s*=\s*true[\s\S]*?rollback\s*=\s*true[\s\S]*?\}/,
      "Amazon ECS service must roll back an unhealthy deployment",
    ],
    [
      /wait_for_steady_state\s*=\s*true/,
      "Terraform must wait for the Amazon ECS service to become healthy",
    ],
  ] as const
) {
  if (!pattern.test(deploymentTerraform)) {
    violations.push({ path: "deploy/terraform", message });
  }
}

if (!/assign_public_ip\s*=\s*false/.test(deploymentTerraform)) {
  violations.push({
    path: "deploy/terraform",
    message: "Amazon ECS tasks must run without public IP addresses",
  });
}

// The embedded app guest is a build artifact: it is committed as an empty
// placeholder and rebuilt by `make wasi-app-guest`. Guard against accidentally
// committing the ~12MB built version.
const embeddedGuestPath = "internal/wasiguest/app-guest.wasm";
try {
  const info = await Deno.stat(embeddedGuestPath);
  if (info.size > 0) {
    violations.push({
      path: embeddedGuestPath,
      message:
        "embedded app guest must stay an empty placeholder (it is built by `make wasi-app-guest`, not committed)",
    });
  }
} catch {
  // Absent is fine; the embed simply falls back to the native mux.
}

if (violations.length > 0) {
  for (const violation of violations) {
    console.error(`${violation.path}: ${violation.message}`);
  }

  Deno.exit(1);
}
