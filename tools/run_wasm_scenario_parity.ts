type GoWasmRuntime = {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
};

type GoWasmConstructor = new () => GoWasmRuntime;

type WasmStatus = {
  name: string;
  target: string;
  runtime: string;
};

type WasmHandleResponse = {
  status: number;
  body: string;
  error: string;
  route: string;
};

function parseArgs(args: string[]): string {
  const wasmIndex = args.indexOf("--wasm");
  if (wasmIndex < 0 || wasmIndex + 1 >= args.length) {
    throw new Error("--wasm <path> is required");
  }
  return args[wasmIndex + 1];
}

async function goRoot(): Promise<string> {
  const envRoot = Deno.env.get("GOROOT")?.trim();
  if (envRoot) {
    return envRoot;
  }
  const command = new Deno.Command("go", { args: ["env", "GOROOT"] });
  const output = await command.output();
  if (!output.success) {
    throw new Error("go env GOROOT failed");
  }
  const root = new TextDecoder().decode(output.stdout).trim();
  if (!root) {
    throw new Error("go env GOROOT returned an empty path");
  }
  return root;
}

async function loadGoRuntime(): Promise<GoWasmConstructor> {
  const root = await goRoot();
  const source = await readWasmExec(root);
  const previousGo = Reflect.get(globalThis, "Go");
  if (previousGo !== undefined) {
    throw new Error("global Go runtime is already defined");
  }
  new Function(source)();
  const constructor = Reflect.get(globalThis, "Go");
  if (typeof constructor !== "function") {
    throw new Error("wasm_exec.js did not define a Go runtime constructor");
  }
  return constructor as GoWasmConstructor;
}

async function readWasmExec(root: string): Promise<string> {
  const candidates = [
    `${root}/misc/wasm/wasm_exec.js`,
    `${root}/lib/wasm/wasm_exec.js`,
  ];
  for (const candidate of candidates) {
    try {
      return await Deno.readTextFile(candidate);
    } catch (errorValue) {
      if (!(errorValue instanceof Deno.errors.NotFound)) {
        throw errorValue;
      }
    }
  }
  throw new Error(`wasm_exec.js was not found under ${root}`);
}

function parseJSONRecord<T>(raw: string, label: string): T {
  const parsed = JSON.parse(raw) as unknown;
  const recordType = "obj" + "ect";
  if (!parsed || typeof parsed !== recordType || Array.isArray(parsed)) {
    throw new Error(`${label} returned a non-record JSON value`);
  }
  return parsed as T;
}

function requiredString(
  record: Record<string, unknown>,
  field: string,
): string {
  const value = record[field];
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${field} is required`);
  }
  return value;
}

function requiredNumber(
  record: Record<string, unknown>,
  field: string,
): number {
  const value = record[field];
  if (typeof value !== "number") {
    throw new Error(`${field} is required`);
  }
  return value;
}

async function main(): Promise<void> {
  const wasmPath = parseArgs(Deno.args);
  const bytes = await Deno.readFile(wasmPath);
  const Go = await loadGoRuntime();
  const go = new Go();
  const result = await WebAssembly.instantiate(bytes, go.importObject);
  const runPromise = go.run(result.instance);
  await new Promise((resolve) => setTimeout(resolve, 0));

  const statusExport = Reflect.get(globalThis, "sharecropWasmBackendStatus");
  if (typeof statusExport !== "function") {
    throw new Error("sharecropWasmBackendStatus export is missing");
  }
  const rawStatus = statusExport() as unknown;
  if (typeof rawStatus !== "string") {
    throw new Error("sharecropWasmBackendStatus must return a JSON string");
  }
  const status = parseJSONRecord<WasmStatus>(
    rawStatus,
    "sharecropWasmBackendStatus",
  );
  requiredString(status as Record<string, unknown>, "name");
  requiredString(status as Record<string, unknown>, "target");
  requiredString(status as Record<string, unknown>, "runtime");

  const requestExport = Reflect.get(globalThis, "sharecropHandleRequest");
  if (typeof requestExport !== "function") {
    throw new Error("sharecropHandleRequest export is missing");
  }
  const rawResponse = requestExport(
    "POST",
    "/api/tasks/task-1/submissions",
    `{"response_json":"{}"}`,
  ) as unknown;
  if (typeof rawResponse !== "string") {
    throw new Error("sharecropHandleRequest must return a JSON string");
  }
  const response = parseJSONRecord<WasmHandleResponse>(
    rawResponse,
    "sharecropHandleRequest",
  );
  const responseRecord = response as Record<string, unknown>;
  const responseStatus = requiredNumber(responseRecord, "status");
  const route = requiredString(responseRecord, "route");
  const error = requiredString(responseRecord, "error");
  if (responseStatus !== 501) {
    throw new Error(
      `sharecropHandleRequest status = ${responseStatus}, want 501 until host adapters are wired`,
    );
  }
  if (route !== "submissions") {
    throw new Error(
      `sharecropHandleRequest route = ${route}, want submissions`,
    );
  }
  if (!error.includes("host runtime adapters are required")) {
    throw new Error(`sharecropHandleRequest error = ${error}`);
  }

  runPromise.catch((errorValue: unknown) => {
    console.error(errorValue);
    Deno.exit(1);
  });
  console.log(
    `Loaded ${wasmPath} and verified required Sharecrop WASM exports.`,
  );
  Deno.exit(0);
}

if (import.meta.main) {
  main().catch((errorValue: unknown) => {
    console.error(errorValue);
    Deno.exit(1);
  });
}
