export type GoWasmRuntime = {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
};

export type GoWasmConstructor = new () => GoWasmRuntime;

export type WasmStatus = {
  name: string;
  target: string;
  runtime: string;
};

export type WasmConfigureResponse = {
  status: string;
  error: string;
};

export type WasmHandleResponse = {
  status: number;
  body: string;
  error: string;
};

export type HostFunctions = {
  storageHas(key: string): boolean;
  storageGet(key: string): string;
  storagePut(key: string, value: string): boolean;
  now(): string;
  nextID(kind: string): string;
};

export async function goRoot(): Promise<string> {
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

export async function loadGoRuntime(): Promise<GoWasmConstructor> {
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

export function parseJSONRecord<T>(raw: string, label: string): T {
  const parsed = JSON.parse(raw) as unknown;
  const recordType = "obj" + "ect";
  if (!parsed || typeof parsed !== recordType || Array.isArray(parsed)) {
    throw new Error(`${label} returned a non-record JSON value`);
  }
  return parsed as T;
}

export function requiredString(
  record: Record<string, unknown>,
  field: string,
): string {
  const value = record[field];
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${field} is required`);
  }
  return value;
}

export function requiredNumber(
  record: Record<string, unknown>,
  field: string,
): number {
  const value = record[field];
  if (typeof value !== "number") {
    throw new Error(`${field} is required`);
  }
  return value;
}

export function stringField(
  record: Record<string, unknown>,
  field: string,
): string {
  const value = record[field];
  if (typeof value !== "string") {
    throw new Error(`${field} must be a string`);
  }
  return value;
}

export function arrayField(
  record: Record<string, unknown>,
  field: string,
): unknown[] {
  const value = record[field];
  if (!Array.isArray(value)) {
    throw new Error(`${field} must be a list`);
  }
  return value;
}

export function recordField(
  record: Record<string, unknown>,
  field: string,
): Record<string, unknown> {
  const value = record[field];
  const recordType = "obj" + "ect";
  if (!value || typeof value !== recordType || Array.isArray(value)) {
    throw new Error(`${field} must be a record`);
  }
  return value as Record<string, unknown>;
}

/**
 * createHost builds the reference non-browser host adapter set: an
 * in-memory, deterministic implementation of the same `HostFunctions`
 * contract that `site/demo/wasm-host.js` satisfies with browser
 * `localStorage`/`Date`. It is the documented starting point for a
 * non-browser WASM host (see docs/wasm_demo_backend_spike.md), not a
 * production-ready host: storage is process-local and unpersisted, and IDs
 * are sequential rather than cryptographically random. No user pre-seeding
 * is needed here - the WASM binary seeds its own fixed demo cast (real
 * accounts, real UUIDs) on `sharecropConfigureHost`, the same as the
 * browser demo does.
 */
export function createHost(nowValue = "2026-07-01T10:00:00Z"): HostFunctions {
  const storage = new Map<string, string>();
  const counters = new Map<string, number>();
  return {
    storageHas(key: string): boolean {
      return storage.has(key);
    },
    storageGet(key: string): string {
      const value = storage.get(key);
      if (value === undefined) {
        throw new Error(`missing WASM storage key ${key}`);
      }
      return value;
    },
    storagePut(key: string, value: string): boolean {
      storage.set(key, value);
      return true;
    },
    now(): string {
      return nowValue;
    },
    nextID(kind: string): string {
      const current = counters.get(kind) ?? 0;
      const next = current + 1;
      counters.set(kind, next);
      return `${kind}-${next}`;
    },
  };
}

export function wasmFunction(name: string): (...args: unknown[]) => unknown {
  const value = Reflect.get(globalThis, name);
  if (typeof value !== "function") {
    throw new Error(`${name} export is missing`);
  }
  return value as (...args: unknown[]) => unknown;
}

export function callJSON<T>(
  fn: (...args: unknown[]) => unknown,
  label: string,
  ...args: unknown[]
): T {
  const raw = fn(...args);
  if (typeof raw !== "string") {
    throw new Error(`${label} must return a JSON string`);
  }
  return parseJSONRecord<T>(raw, label);
}

export function request(
  fn: (...args: unknown[]) => unknown,
  method: string,
  path: string,
  body: string,
  label: string,
  authorization = "",
): WasmHandleResponse {
  const response = callJSON<WasmHandleResponse>(
    fn,
    label,
    method,
    path,
    body,
    authorization,
  );
  requiredNumber(response as Record<string, unknown>, "status");
  stringField(response as Record<string, unknown>, "body");
  stringField(response as Record<string, unknown>, "error");
  return response;
}

export function assertStatus(
  response: WasmHandleResponse,
  expected: number,
  label: string,
): void {
  if (response.status !== expected) {
    throw new Error(
      `${label} status = ${response.status}, want ${expected}: ${response.error}`,
    );
  }
}

export function responseBody(
  response: WasmHandleResponse,
  label: string,
): Record<string, unknown> {
  if (!response.body) {
    throw new Error(`${label} response body is required`);
  }
  return parseJSONRecord<Record<string, unknown>>(response.body, label);
}

/**
 * instantiateWasm loads a compiled Go `js/wasm` artifact through
 * `wasm_exec.js` and starts the Go runtime. Callers must wait one tick
 * (`await new Promise((resolve) => setTimeout(resolve, 0))`) before the
 * exported `sharecropWasmBackendStatus`/`sharecropConfigureHost`/
 * `sharecropHandleRequest` globals are safe to call.
 */
export async function instantiateWasm(bytes: Uint8Array<ArrayBuffer>): Promise<{
  go: GoWasmRuntime;
  runPromise: Promise<void>;
}> {
  const Go = await loadGoRuntime();
  const go = new Go();
  const result = await WebAssembly.instantiate(bytes, go.importObject);
  const runPromise = go.run(result.instance);
  return { go, runPromise };
}
