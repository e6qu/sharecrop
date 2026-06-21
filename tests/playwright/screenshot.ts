import { chromium } from "playwright";

const output = Deno.args[0];
if (output === undefined || output.length === 0) {
  console.error("screenshot output path is required");
  Deno.exit(1);
}

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 720 } });
await page.goto("http://127.0.0.1:18080/");
await page.getByRole("heading", { name: "Sharecrop" }).waitFor();
await page.screenshot({ path: output, fullPage: true });
await browser.close();
