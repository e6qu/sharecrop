import { defineConfig } from "@playwright/test";
import process from "node:process";

const apiPort = process.env.SHARECROP_PLAYWRIGHT_API_PORT ?? "29180";
const demoPort = process.env.SHARECROP_PLAYWRIGHT_DEMO_PORT ?? "29181";
const databaseURL = process.env.DATABASE_URL ??
  "postgres://sharecrop:sharecrop@127.0.0.1:25432/sharecrop?sslmode=disable";
const apiOrigin = `http://127.0.0.1:${apiPort}`;
const demoOrigin = `http://127.0.0.1:${demoPort}`;

export default defineConfig({
  testDir: ".",
  // Keep local and CI concurrency equal so the browser contract has one
  // deterministic execution mode. A failure is reported on its first run.
  workers: 2,
  retries: 0,
  expect: { timeout: 15_000 },
  use: {
    baseURL: apiOrigin,
  },
  webServer: [
    {
      // Account-token delivery defaults to log (fail closed); the browser
      // account/reset flows read the token from the response, so this test
      // server opts into api delivery like the demo does. The registration
      // budget is raised far above the production default (5 per IP) because
      // this suite registers a fresh account per test, all from 127.0.0.1.
      // Both capacity and refill are raised: buckets persist in the store, so
      // a bucket drained by an earlier suite against the same database would
      // otherwise still reject registrations regardless of capacity.
      command:
        `SHARECROP_HTTP_ADDR=:${apiPort} SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 SHARECROP_ACCOUNT_TOKEN_DELIVERY=api SHARECROP_REGISTRATION_RATE_CAPACITY=100000 SHARECROP_REGISTRATION_RATE_REFILL=100000 DATABASE_URL='${databaseURL}' SHARECROP_MIGRATIONS_DIR=migrations go run ./cmd/sharecrop serve`,
      cwd: "../..",
      url: `${apiOrigin}/healthz`,
      reuseExistingServer: true,
      timeout: 30_000,
    },
    {
      // Static origin for the demo bundle.
      // Browser.application needs a real HTTP origin, so file:// will not do.
      command:
        `deno run --allow-net --allow-read jsr:@std/http@1/file-server -p ${demoPort} site/demo`,
      cwd: "../..",
      url: `${demoOrigin}/index.html`,
      reuseExistingServer: true,
      timeout: 30_000,
    },
  ],
});
