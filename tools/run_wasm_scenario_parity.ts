import {
  type JsonRecord,
  noScenarioBody,
  runSharedScenarioParity,
  type ScenarioBody,
  type ScenarioClient,
  type ScenarioResponse,
} from "../tests/scenario_parity/scenario.ts";
import {
  assertStatus,
  callJSON,
  createHost,
  instantiateWasm,
  parseJSONRecord,
  request,
  requiredString,
  type WasmConfigureResponse,
  wasmFunction,
  type WasmStatus,
} from "./wasm_runtime_loader.ts";

function parseArgs(args: string[]): string {
  const wasmIndex = args.indexOf("--wasm");
  if (wasmIndex < 0 || wasmIndex + 1 >= args.length) {
    throw new Error("--wasm <path> is required");
  }
  return args[wasmIndex + 1];
}

// WasmScenarioClient drives the WASM binary's own real internal/http mux
// directly (no network, no fake actor-id scheme): the Authorization header
// is just forwarded as the 4th sharecropHandleRequest argument, exactly
// the same contract site/demo/wasm-host.js's WasmXHR now uses. The WASM
// binary's own request bridge manages the refresh-token cookie internally,
// so this client never needs to touch cookies.
class WasmScenarioClient implements ScenarioClient {
  constructor(
    private readonly requestExport: (...args: unknown[]) => unknown,
    private readonly accessToken: string,
  ) {}

  request(
    method: string,
    path: string,
    body: ScenarioBody,
  ): Promise<ScenarioResponse> {
    const rawBody = body === noScenarioBody ? "" : JSON.stringify(body);
    const authorization = this.accessToken === ""
      ? ""
      : `Bearer ${this.accessToken}`;
    const response = request(
      this.requestExport,
      method,
      path,
      rawBody,
      `${method} ${path}`,
      authorization,
    );
    const json = response.body.trim() === ""
      ? {}
      : parseJSONRecord<JsonRecord>(response.body, `${method} ${path}`);
    if (response.status >= 400 && response.error !== "") {
      json.error = response.error;
    }
    return Promise.resolve({ status: response.status, json });
  }

  withAccessToken(accessToken: string): ScenarioClient {
    return new WasmScenarioClient(this.requestExport, accessToken);
  }
}

async function main(): Promise<void> {
  const wasmPath = parseArgs(Deno.args);
  const bytes = await Deno.readFile(wasmPath);
  const { runPromise } = await instantiateWasm(bytes);
  await new Promise((resolve) => setTimeout(resolve, 0));

  const statusExport = wasmFunction("sharecropWasmBackendStatus");
  const initialStatus = callJSON<WasmStatus>(
    statusExport,
    "sharecropWasmBackendStatus",
  );
  if (
    requiredString(initialStatus as Record<string, unknown>, "runtime") !==
      "unconfigured"
  ) {
    throw new Error("initial WASM runtime status must be unconfigured");
  }

  const requestExport = wasmFunction("sharecropHandleRequest");
  const unconfigured = request(
    requestExport,
    "POST",
    "/api/tasks",
    "{}",
    "unconfigured request",
  );
  assertStatus(unconfigured, 500, "unconfigured request");
  if (!unconfigured.error.includes("host runtime is not configured")) {
    throw new Error(`unconfigured error = ${unconfigured.error}`);
  }

  const host = createHost();
  const configureExport = wasmFunction("sharecropConfigureHost");
  const configure = callJSON<WasmConfigureResponse>(
    configureExport,
    "sharecropConfigureHost",
    host,
  );
  if (
    requiredString(configure as Record<string, unknown>, "status") !==
      "configured"
  ) {
    throw new Error(`WASM host did not configure: ${configure.error}`);
  }

  const configuredStatus = callJSON<WasmStatus>(
    statusExport,
    "sharecropWasmBackendStatus",
  );
  if (
    requiredString(configuredStatus as Record<string, unknown>, "runtime") !==
      "configured"
  ) {
    throw new Error("configured WASM runtime status must be configured");
  }

  const unsupported = request(
    requestExport,
    "GET",
    "/api/not-implemented",
    "",
    "unsupported route",
  );
  assertStatus(unsupported, 404, "unsupported route");

  // No access token yet: sharecropConfigureHost already seeded the demo
  // scenario and pre-authenticated its admin user, so the shared scenario's
  // first call (POST /api/auth/refresh) succeeds via the WASM binary's own
  // internally-held refresh-token cookie, exactly like the browser demo.
  await runSharedScenarioParity(new WasmScenarioClient(requestExport, ""));

  runPromise.catch((errorValue: unknown) => {
    console.error(errorValue);
    Deno.exit(1);
  });
  console.log(`Executed configured Sharecrop WASM scenario from ${wasmPath}.`);
  Deno.exit(0);
}

if (import.meta.main) {
  main().catch((errorValue: unknown) => {
    console.error(errorValue);
    Deno.exit(1);
  });
}
