import { chromium } from "npm:playwright@1.61.0";

const output = Deno.args[0] ?? "/tmp/sharecrop-pr1-shell.png";

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 720 } });
await page.goto("http://127.0.0.1:18080/");
await page.getByRole("heading", { name: "Sharecrop" }).waitFor();
await page.screenshot({ path: output, fullPage: true });
await browser.close();
