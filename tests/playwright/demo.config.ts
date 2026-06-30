import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  use: {
    baseURL: "http://127.0.0.1:18081",
  },
  webServer: {
    command:
      "deno run --allow-net --allow-read jsr:@std/http@1/file-server -p 18081 site/demo",
    cwd: "../..",
    url: "http://127.0.0.1:18081/index.html",
    reuseExistingServer: true,
    timeout: 30_000,
  },
});
