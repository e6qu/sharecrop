import { defineConfig } from "npm:@playwright/test";

export default defineConfig({
  testDir: ".",
  use: {
    baseURL: "http://127.0.0.1:18080",
  },
  webServer: {
    command: "go run ./cmd/sharecrop serve",
    cwd: "../..",
    url: "http://127.0.0.1:18080/healthz",
    reuseExistingServer: true,
    timeout: 30_000,
  },
});
