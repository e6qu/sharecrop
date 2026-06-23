import { chromium } from "playwright";

type ScreenshotCase = {
  name: string;
  width: number;
  height: number;
  mode: string;
  theme: string;
  themeLabel: string;
};

const outputDirectory = "/tmp/sharecrop-screens";
const demoUrl = `file://${Deno.cwd()}/site/demo/index.html`;

const cases: ScreenshotCase[] = [
  {
    name: "desktop-corporate-light",
    width: 1440,
    height: 1100,
    mode: "Light",
    theme: "corporate",
    themeLabel: "Corporate",
  },
  {
    name: "desktop-discover-corporate-light",
    width: 1440,
    height: 1100,
    mode: "Light",
    theme: "corporate",
    themeLabel: "Corporate",
  },
  {
    name: "desktop-requester-corporate-light",
    width: 1440,
    height: 1100,
    mode: "Light",
    theme: "corporate",
    themeLabel: "Corporate",
  },
  {
    name: "desktop-review-corporate-light",
    width: 1440,
    height: 1100,
    mode: "Light",
    theme: "corporate",
    themeLabel: "Corporate",
  },
  {
    name: "desktop-integrations-corporate-light",
    width: 1440,
    height: 1100,
    mode: "Light",
    theme: "corporate",
    themeLabel: "Corporate",
  },
  {
    name: "desktop-settings-corporate-light",
    width: 1440,
    height: 1100,
    mode: "Light",
    theme: "corporate",
    themeLabel: "Corporate",
  },
  {
    name: "desktop-blocky-dark",
    width: 1440,
    height: 1100,
    mode: "Dark",
    theme: "blocky",
    themeLabel: "Blocky",
  },
  {
    name: "mobile-rustic-light",
    width: 390,
    height: 1200,
    mode: "Light",
    theme: "rustic",
    themeLabel: "Rustic",
  },
  {
    name: "mobile-showcase-dark",
    width: 390,
    height: 1200,
    mode: "Dark",
    theme: "showcase",
    themeLabel: "Showcase",
  },
];

await Deno.mkdir(outputDirectory, { recursive: true });

const browser = await chromium.launch();
for (const item of cases) {
  const page = await browser.newPage({
    viewport: { width: item.width, height: item.height },
  });
  await page.goto(demoUrl);
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("button", { name: item.mode, exact: true }).click();
  await page.getByRole("button", { name: new RegExp(item.themeLabel) }).click();
  await page.getByRole("button", { name: "Overview", exact: true }).click();
  if (item.name.includes("discover")) {
    await page.getByRole("button", { name: "Discover", exact: true }).click();
  }
  if (item.name.includes("requester")) {
    await page.getByRole("button", { name: "Create", exact: true }).click();
  }
  if (item.name.includes("review")) {
    await page.getByRole("button", { name: "Review", exact: true }).click();
  }
  if (item.name.includes("integrations")) {
    await page.getByRole("button", { name: "API & MCP", exact: true }).click();
  }
  if (item.name.includes("settings")) {
    await page.getByRole("button", { name: "Settings", exact: true }).click();
  }
  await page.screenshot({
    path: `${outputDirectory}/${item.name}.png`,
    fullPage: true,
  });
  await page.close();
}
await browser.close();
