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

if (violations.length > 0) {
  for (const violation of violations) {
    console.error(`${violation.path}: ${violation.message}`);
  }

  Deno.exit(1);
}
