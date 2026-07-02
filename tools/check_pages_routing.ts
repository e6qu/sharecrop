interface CheckTarget {
  path: string;
  mustContain: string;
}

function parseOrigin(args: string[]): string {
  const normalized = args[0] === "--" ? args.slice(1) : args;
  if (normalized.length !== 2 || normalized[0] !== "--origin") {
    throw new Error(
      "usage: deno run --allow-net tools/check_pages_routing.ts --origin https://example.github.io/sharecrop",
    );
  }
  const origin = normalized[1].trim();
  if (origin === "") {
    throw new Error("missing required --origin value");
  }
  return origin.replace(/\/+$/, "");
}

async function checkTarget(origin: string, target: CheckTarget): Promise<void> {
  const url = `${origin}${target.path}`;
  const response = await fetch(url, { redirect: "follow" });
  if (response.status !== 200) {
    throw new Error(`${url} returned HTTP ${response.status}`);
  }
  const text = await response.text();
  if (!text.includes(target.mustContain)) {
    throw new Error(
      `${url} did not contain expected marker: ${target.mustContain}`,
    );
  }
}

const origin = parseOrigin(Deno.args);
const targets: CheckTarget[] = [
  { path: "/", mustContain: "Sharecrop" },
  { path: "/docs/", mustContain: "Sharecrop" },
  { path: "/docs/openapi.html", mustContain: "OpenAPI reference" },
  { path: "/docs/openapi.json", mustContain: '"openapi"' },
  { path: "/demo", mustContain: "Sharecrop Demo" },
  { path: "/demo/", mustContain: "Sharecrop Demo" },
  { path: "/demo/index.html", mustContain: "Sharecrop Demo" },
  { path: "/demo/app.css", mustContain: "tailwindcss" },
  { path: "/demo/arcade.css", mustContain: "Arcade" },
  { path: "/demo/wasm-host.js", mustContain: "SharecropWasmDemo" },
  { path: "/demo/wasm_exec.js", mustContain: "globalThis.Go" },
  { path: "/demo/elm.js", mustContain: "Elm" },
];

for (const target of targets) {
  await checkTarget(origin, target);
}

console.log(`GitHub Pages routing check passed for ${origin}`);
