import { expect, test } from "@playwright/test";
import { contentType } from "jsr:@std/media-types@1";
import { extname } from "jsr:@std/path@1";

// The demo serves the REAL compiled Elm client (site/demo) against an in-browser
// fake backend (backend.js). Browser.application needs a real HTTP origin, so we
// serve site/demo from an in-process static server for the test.
const demoRoot = new URL("../../site/demo/", import.meta.url).pathname;
let server: Deno.HttpServer;
let demoOrigin = "";

test.beforeAll(async () => {
  const ac = new AbortController();
  server = Deno.serve(
    { port: 0, signal: ac.signal, onListen: () => {} },
    async (req) => {
      let path = new URL(req.url).pathname;
      if (path === "/") path = "/index.html";
      try {
        const bytes = await Deno.readFile(demoRoot + path.replace(/^\//, ""));
        return new Response(bytes, {
          headers: {
            "content-type": contentType(extname(path)) ||
              "application/octet-stream",
          },
        });
      } catch {
        return new Response("not found", { status: 404 });
      }
    },
  );
  // @ts-ignore signal kept on the server for teardown
  server._ac = ac;
  demoOrigin = `http://localhost:${(server.addr as Deno.NetAddr).port}`;
});

test.afterAll(() => {
  // @ts-ignore
  server._ac.abort();
});

test("demo boots the real Elm client against the fake backend with seeded tasks", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);

  // Boots straight into the seeded account (refresh auto-succeeds in the shim).
  await expect(page.getByText("1240 credits")).toBeVisible();

  // The real client's Discovery page lists the realistic seeded tasks.
  await page.getByRole("link", { name: "Discovery" }).click();
  await expect(
    page.getByText("Extract line items from 6 vendor invoices"),
  ).toBeVisible();
  await expect(page.getByText("Classify 20 support tickets by category"))
    .toBeVisible();

  // Opening a task shows the real detail view with its instructions and the
  // typed response schema, served by the fake backend.
  await page.getByTestId("discovery-view").first().click();
  await expect(
    page.getByText("Read the 6 attached invoice scans", { exact: false }),
  )
    .toBeVisible();
  await expect(page.getByText('"invoice_id"', { exact: false })).toBeVisible();
});
