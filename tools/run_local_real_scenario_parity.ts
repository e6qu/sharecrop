import {
  assertScenario,
  type JsonRecord,
  noScenarioBody,
  runSharedScenarioParity,
} from "../tests/scenario_parity/scenario.ts";

interface Options {
  origin: string;
  databaseURL: string;
}

interface AdminSession {
  subjectID: string;
  accessToken: string;
  refreshToken: string;
}

function parseArgs(args: string[]): Options {
  const normalized = args[0] === "--" ? args.slice(1) : args;
  let origin = "";
  let databaseURL = Deno.env.get("DATABASE_URL") || "";
  for (let index = 0; index < normalized.length; index += 1) {
    const arg = normalized[index];
    if (arg === "--origin") {
      origin = normalized[index + 1] || "";
      index += 1;
    } else if (arg === "--database-url") {
      databaseURL = normalized[index + 1] || "";
      index += 1;
    } else {
      throw new Error(`unknown argument: ${arg}`);
    }
  }
  if (origin.trim() === "") {
    throw new Error("missing required --origin");
  }
  if (databaseURL.trim() === "") {
    throw new Error("missing required DATABASE_URL or --database-url");
  }
  return { origin: origin.replace(/\/+$/, ""), databaseURL };
}

function parseResponseBody(text: string, label: string): JsonRecord {
  if (text === "") return {};
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`${label} returned invalid JSON: ${message}`);
  }
  assertScenario(
    parsed != undefined && !Array.isArray(parsed) &&
      typeof parsed !== "string" && typeof parsed !== "number" &&
      typeof parsed !== "boolean",
    `${label} returned a non-record response`,
  );
  return parsed as JsonRecord;
}

function requireString(value: JsonRecord, key: string): string {
  const found = value[key];
  if (typeof found !== "string" || found.trim() === "") {
    throw new Error(`${key} must be a non-empty string`);
  }
  return found;
}

function extractRefreshToken(headers: Headers): string {
  const setCookie = headers.get("set-cookie") || "";
  const match = setCookie.match(/(?:^|,\s*)sharecrop_refresh_token=([^;]+)/);
  return match?.[1] || "";
}

async function probeHealth(origin: string): Promise<void> {
  const health = await fetch(`${origin}/healthz`);
  if (health.status !== 200) {
    const body = await health.text();
    throw new Error(
      `GET /healthz returned ${health.status}: ${body.slice(0, 400)}`,
    );
  }
}

async function registerAdmin(origin: string): Promise<AdminSession> {
  const email = `local-real-parity-admin-${Date.now()}@example.com`;
  const response = await fetch(`${origin}/api/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email,
      password: "correct horse battery staple",
    }),
  });
  const text = await response.text();
  if (response.status !== 201) {
    throw new Error(
      `POST /api/auth/register returned ${response.status}: ${
        text.slice(0, 400)
      }`,
    );
  }
  const refreshToken = extractRefreshToken(response.headers);
  if (refreshToken === "") {
    throw new Error("registration response did not set refresh cookie");
  }
  const body = parseResponseBody(text, "POST /api/auth/register");
  const subjectID = requireString(body, "subject_id");
  const accessToken = requireString(body, "access_token");
  if (
    !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
      .test(subjectID)
  ) {
    throw new Error("registered subject_id is not a UUID");
  }
  return { subjectID, accessToken, refreshToken };
}

async function grantPlatformAdmin(
  databaseURL: string,
  userID: string,
): Promise<void> {
  const sql =
    `insert into platform_admins (user_id, source, state, granted_by_user_id) values ('${userID}'::uuid, 'granted', 'active', '${userID}'::uuid) on conflict (user_id) do update set source = 'granted', state = 'active', granted_by_user_id = '${userID}'::uuid, updated_at = now();`;
  const command = new Deno.Command("psql", {
    args: [databaseURL, "-v", "ON_ERROR_STOP=1", "-c", sql],
    stdout: "piped",
    stderr: "piped",
  });
  const output = await command.output();
  if (!output.success) {
    const stderr = new TextDecoder().decode(output.stderr).trim();
    const stdout = new TextDecoder().decode(output.stdout).trim();
    throw new Error(
      `platform admin grant failed with psql exit ${output.code}: ${
        stderr || stdout
      }`,
    );
  }
}

function clientForSession(origin: string, session: AdminSession) {
  let refreshToken = session.refreshToken;
  const makeClient = (accessToken: string) => ({
    async request(
      method: string,
      path: string,
      body: typeof noScenarioBody | JsonRecord,
    ) {
      const response = await fetch(`${origin}${path}`, {
        method,
        headers: {
          "Authorization": `Bearer ${accessToken}`,
          "Content-Type": "application/json",
          "Cookie": `sharecrop_refresh_token=${refreshToken}`,
        },
        body: body === noScenarioBody ? undefined : JSON.stringify(body),
      });
      const nextRefreshToken = extractRefreshToken(response.headers);
      if (nextRefreshToken !== "") {
        refreshToken = nextRefreshToken;
      }
      const text = await response.text();
      return {
        status: response.status,
        json: parseResponseBody(text, `${method} ${path}`),
      };
    },
    withAccessToken(nextToken: string) {
      return makeClient(nextToken);
    },
  });
  return makeClient(session.accessToken);
}

const options = parseArgs(Deno.args);
await probeHealth(options.origin);
const admin = await registerAdmin(options.origin);
await grantPlatformAdmin(options.databaseURL, admin.subjectID);
await runSharedScenarioParity(clientForSession(options.origin, admin));

console.log("local real API shared scenario parity suite passed");
