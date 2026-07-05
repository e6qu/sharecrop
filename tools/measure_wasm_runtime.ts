import {
  assertStatus,
  callJSON,
  createHost,
  instantiateWasm,
  request,
  requiredString,
  responseBody,
  type WasmConfigureResponse,
  wasmFunction,
  type WasmStatus,
} from "./wasm_runtime_loader.ts";

type LatencySample = {
  route: string;
  millis: number;
};

type LatencyStats = {
  route: string;
  count: number;
  minMillis: number;
  meanMillis: number;
  p50Millis: number;
  p95Millis: number;
  maxMillis: number;
};

function parseArgs(
  args: string[],
): { wasmPath: string; requestsPerRoute: number } {
  const wasmIndex = args.indexOf("--wasm");
  if (wasmIndex < 0 || wasmIndex + 1 >= args.length) {
    throw new Error("--wasm <path> is required");
  }
  const requestsIndex = args.indexOf("--requests-per-route");
  const requestsPerRoute = requestsIndex < 0 || requestsIndex + 1 >= args.length
    ? 200
    : Number.parseInt(args[requestsIndex + 1], 10);
  if (!Number.isFinite(requestsPerRoute) || requestsPerRoute <= 0) {
    throw new Error("--requests-per-route must be a positive integer");
  }
  return { wasmPath: args[wasmIndex + 1], requestsPerRoute };
}

function percentile(sortedMillis: number[], fraction: number): number {
  if (sortedMillis.length === 0) {
    throw new Error("percentile requires at least one sample");
  }
  const index = Math.min(
    sortedMillis.length - 1,
    Math.floor(fraction * (sortedMillis.length - 1)),
  );
  return sortedMillis[index];
}

function summarize(route: string, samples: LatencySample[]): LatencyStats {
  const millis = samples.filter((sample) => sample.route === route).map((
    sample,
  ) => sample.millis).sort((a, b) => a - b);
  if (millis.length === 0) {
    throw new Error(`no latency samples recorded for ${route}`);
  }
  const sum = millis.reduce((total, value) => total + value, 0);
  return {
    route,
    count: millis.length,
    minMillis: millis[0],
    meanMillis: sum / millis.length,
    p50Millis: percentile(millis, 0.5),
    p95Millis: percentile(millis, 0.95),
    maxMillis: millis[millis.length - 1],
  };
}

function formatMillis(value: number): string {
  return `${value.toFixed(3)}ms`;
}

function formatMebibytes(bytes: number): string {
  return `${(bytes / (1024 * 1024)).toFixed(2)} MiB`;
}

async function main(): Promise<void> {
  const { wasmPath, requestsPerRoute } = parseArgs(Deno.args);

  const artifactSize = (await Deno.stat(wasmPath)).size;
  const memoryBeforeLoad = Deno.memoryUsage();

  const bytes = await Deno.readFile(wasmPath);
  const instantiateStart = performance.now();
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
  const instantiateMillis = performance.now() - instantiateStart;

  const requestExport = wasmFunction("sharecropHandleRequest");
  const host = createHost();
  const configureExport = wasmFunction("sharecropConfigureHost");
  const configureStart = performance.now();
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
  const configureMillis = performance.now() - configureStart;
  const memoryAfterConfigure = Deno.memoryUsage();

  // sharecropConfigureHost already seeded the demo scenario and
  // pre-authenticated its admin user, so refresh (relying on the WASM
  // binary's own internally-held cookie) is enough to get a real access
  // token for the protected routes measured below.
  const refresh = request(
    requestExport,
    "POST",
    "/api/auth/refresh",
    "",
    "refresh",
  );
  assertStatus(refresh, 200, "refresh");
  const accessToken = requiredString(
    responseBody(refresh, "refresh"),
    "access_token",
  );
  const authorization = `Bearer ${accessToken}`;

  const organization = request(
    requestExport,
    "POST",
    "/api/organizations",
    JSON.stringify({ name: "Measurement Org" }),
    "create organization",
    authorization,
  );
  assertStatus(organization, 201, "create organization");

  const taskBody = JSON.stringify({
    owner: {
      kind: "user",
      user_id: requiredString(
        responseBody(refresh, "refresh"),
        "subject_id",
      ),
    },
    title: "Measurement task",
    description:
      "Exercise configured Go WASM request handling for measurement.",
    reward: { kind: "none" },
    participation: {
      policy: "approval_required",
      assignee_scope: "user",
      reservation_expiry_hours: 48,
    },
    visibility: { kind: "public" },
    placement: { kind: "standalone" },
    response_schema_json: '{"kind":"freeform"}',
    payload: { kind: "none", json: "" },
    task_type: "general",
    attachments: [],
  });
  const createTask = request(
    requestExport,
    "POST",
    "/api/tasks",
    taskBody,
    "create task",
    authorization,
  );
  assertStatus(createTask, 201, "create task");
  const taskID = requiredString(responseBody(createTask, "create task"), "id");

  const measuredRoutes: Array<
    { method: string; path: string; body: string; label: string }
  > = [
    { method: "GET", path: "/api/users", body: "", label: "GET /api/users" },
    {
      method: "GET",
      path: "/api/organizations",
      body: "",
      label: "GET /api/organizations",
    },
    {
      method: "GET",
      path: "/api/tasks?scope=public",
      body: "",
      label: "GET /api/tasks",
    },
    {
      method: "GET",
      path: `/api/tasks/${taskID}`,
      body: "",
      label: "GET /api/tasks/{task_id}",
    },
    {
      method: "GET",
      path: "/api/credits/balance",
      body: "",
      label: "GET /api/credits/balance",
    },
  ];

  const samples: LatencySample[] = [];
  for (const route of measuredRoutes) {
    for (let iteration = 0; iteration < requestsPerRoute; iteration++) {
      const start = performance.now();
      const response = request(
        requestExport,
        route.method,
        route.path,
        route.body,
        route.label,
        authorization,
      );
      const elapsed = performance.now() - start;
      assertStatus(response, 200, route.label);
      samples.push({ route: route.label, millis: elapsed });
    }
  }

  const memoryAfterRequests = Deno.memoryUsage();

  console.log(`Sharecrop WASM runtime measurement for ${wasmPath}`);
  console.log("");
  console.log("Artifact size:");
  console.log(`  ${formatMebibytes(artifactSize)} (${artifactSize} bytes)`);
  console.log("");
  console.log("Startup:");
  console.log(
    `  instantiate + first status call: ${formatMillis(instantiateMillis)}`,
  );
  console.log(`  host configuration: ${formatMillis(configureMillis)}`);
  console.log("");
  console.log(
    "Process memory (Deno.memoryUsage, this host process, not the WASM linear memory):",
  );
  console.log(
    `  rss before load:        ${formatMebibytes(memoryBeforeLoad.rss)}`,
  );
  console.log(
    `  rss after configure:    ${formatMebibytes(memoryAfterConfigure.rss)}`,
  );
  console.log(
    `  rss after ${measuredRoutes.length * requestsPerRoute} requests: ${
      formatMebibytes(memoryAfterRequests.rss)
    }`,
  );
  console.log(
    `  heapUsed after configure: ${
      formatMebibytes(memoryAfterConfigure.heapUsed)
    }`,
  );
  console.log(
    `  heapUsed after requests:  ${
      formatMebibytes(memoryAfterRequests.heapUsed)
    }`,
  );
  console.log("");
  console.log(
    `Request latency (${requestsPerRoute} requests per route, configured host, in-memory storage):`,
  );
  for (const route of measuredRoutes) {
    const stats = summarize(route.label, samples);
    console.log(
      `  ${stats.route}: min=${formatMillis(stats.minMillis)} mean=${
        formatMillis(stats.meanMillis)
      } p50=${formatMillis(stats.p50Millis)} p95=${
        formatMillis(stats.p95Millis)
      } max=${formatMillis(stats.maxMillis)}`,
    );
  }

  runPromise.catch((errorValue: unknown) => {
    console.error(errorValue);
    Deno.exit(1);
  });
  Deno.exit(0);
}

if (import.meta.main) {
  main().catch((errorValue: unknown) => {
    console.error(errorValue);
    Deno.exit(1);
  });
}
