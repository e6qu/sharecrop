import {
  assertScenario,
  type JsonRecord,
  noScenarioBody,
  runSharedScenarioParity,
} from "../scenario_parity/scenario.ts";

interface DemoRoute {
  method: string;
  pattern: string;
}

interface DemoResponse {
  status: number;
  body: string;
}

interface DemoBackend {
  routes: DemoRoute[];
  resolve(
    method: string,
    rawUrl: string,
    rawBody: string | undefined,
    headers: Record<string, string>,
  ): Promise<DemoResponse>;
}

async function loadDemoBackend(): Promise<DemoBackend> {
  const source = await Deno.readTextFile("site/demo/backend.js");
  function RealXMLHttpRequest() {}
  const windowObject = {
    location: { origin: "http://demo.test" },
    XMLHttpRequest: RealXMLHttpRequest,
  };
  const loader = new Function(
    "window",
    "console",
    `${source}\nreturn window.__sharecropDemoBackend;`,
  ) as (windowValue: typeof windowObject, consoleValue: Console) => DemoBackend;
  return loader(windowObject, console);
}

function parseBody(body: string): JsonRecord {
  if (body === "") return {};
  const parsed = JSON.parse(body);
  assertScenario(
    parsed != undefined && !Array.isArray(parsed) &&
      typeof parsed !== "string" && typeof parsed !== "number" &&
      typeof parsed !== "boolean",
    "demo backend returned a non-record response",
  );
  return parsed as JsonRecord;
}

Deno.test("shared scenario parity suite runs against backendless demo", async () => {
  const backend = await loadDemoBackend();
  const clientForToken = (accessToken: string) => ({
    async request(
      method: string,
      path: string,
      body: JsonRecord | typeof noScenarioBody,
    ) {
      const response = await backend.resolve(
        method,
        path,
        body === noScenarioBody ? undefined : JSON.stringify(body),
        accessToken === "" ? {} : { Authorization: `Bearer ${accessToken}` },
      );
      return { status: response.status, json: parseBody(response.body) };
    },
    withAccessToken(nextToken: string) {
      return clientForToken(nextToken);
    },
  });
  await runSharedScenarioParity(clientForToken(""));
});
