import { defineConfig } from "@playwright/test";
import process from "node:process";

const demoPort = process.env.SHARECROP_PLAYWRIGHT_DEMO_PORT ?? "29181";
const demoOrigin = `http://127.0.0.1:${demoPort}`;

export default defineConfig({
  testDir: ".",
  use: {
    baseURL: demoOrigin,
  },
  webServer: {
    command:
      `deno run --allow-net --allow-read jsr:@std/http@1/file-server -p ${demoPort} site/demo`,
    cwd: "../..",
    url: `${demoOrigin}/index.html`,
    reuseExistingServer: true,
    timeout: 30_000,
  },
});
