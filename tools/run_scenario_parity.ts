import {
  assertScenario,
  type JsonRecord,
  noScenarioBody,
  runSharedScenarioParity,
} from "../tests/scenario_parity/scenario.ts";

interface Options {
  origin: string;
  token: string;
}

function parseArgs(args: string[]): Options {
  const normalized = args[0] === "--" ? args.slice(1) : args;
  let origin = "";
  let token = "";
  for (let index = 0; index < normalized.length; index += 1) {
    const arg = normalized[index];
    if (arg === "--origin") {
      origin = normalized[index + 1] || "";
      index += 1;
    } else if (arg === "--token") {
      token = normalized[index + 1] || "";
      index += 1;
    } else {
      throw new Error(`unknown argument: ${arg}`);
    }
  }
  if (origin.trim() === "") {
    throw new Error("missing required --origin");
  }
  if (token.trim() === "") {
    throw new Error("missing required --token");
  }
  return { origin: origin.replace(/\/+$/, ""), token };
}

function parseResponseBody(text: string, label: string): JsonRecord {
  if (text === "") return {};
  const parsed = JSON.parse(text);
  assertScenario(
    parsed != undefined && !Array.isArray(parsed) &&
      typeof parsed !== "string" && typeof parsed !== "number" &&
      typeof parsed !== "boolean",
    `${label} returned a non-record response`,
  );
  return parsed as JsonRecord;
}

const options = parseArgs(Deno.args);

const clientForToken = (accessToken: string) => ({
  async request(
    method: string,
    path: string,
    body: typeof noScenarioBody | JsonRecord,
  ) {
    const response = await fetch(`${options.origin}${path}`, {
      method,
      headers: {
        "Authorization": `Bearer ${accessToken}`,
        "Content-Type": "application/json",
      },
      body: body === noScenarioBody ? undefined : JSON.stringify(body),
    });
    const text = await response.text();
    return {
      status: response.status,
      json: parseResponseBody(text, `${method} ${path}`),
    };
  },
  withAccessToken(nextToken: string) {
    return clientForToken(nextToken);
  },
});

await runSharedScenarioParity(clientForToken(options.token));

console.log("shared scenario parity suite passed");
