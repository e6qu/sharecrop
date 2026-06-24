import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  use: {
    baseURL: "http://127.0.0.1:18080",
  },
  webServer: [
    {
      command:
        "SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=migrations go run ./cmd/sharecrop serve",
      cwd: "../..",
      url: "http://127.0.0.1:18080/healthz",
      reuseExistingServer: true,
      timeout: 30_000,
    },
    {
      // Static origin for the demo bundle (the real Elm client + fake backend).
      // Browser.application needs a real HTTP origin, so file:// will not do.
      command:
        "deno run --allow-net --allow-read jsr:@std/http@1/file-server -p 18081 site/demo",
      cwd: "../..",
      url: "http://127.0.0.1:18081/index.html",
      reuseExistingServer: true,
      timeout: 30_000,
    },
  ],
});
