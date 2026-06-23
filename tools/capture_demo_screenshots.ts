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
    name: "desktop-tasks-corporate-light",
    width: 1440,
    height: 1100,
    mode: "Light",
    theme: "corporate",
    themeLabel: "Corporate",
  },
  {
    name: "desktop-task-detail-corporate-light",
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
  {
    name: "mobile-task-detail-showcase-dark",
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
  await page.getByRole("button", { name: "Dashboard", exact: true }).click();
  if (item.name.includes("tasks")) {
    await page.getByRole("button", { name: "Tasks", exact: true }).click();
  }
  if (item.name.includes("task-detail")) {
    await page.getByRole("button", { name: "Tasks", exact: true }).click();
    await page.getByRole("button", { name: /Label orchard photos/ }).click();
  }
  if (item.name.includes("requester")) {
    await page.getByRole("button", { name: "Post Task", exact: true })
      .click();
  }
  if (item.name.includes("review")) {
    await page.getByRole("button", { name: "Reviews", exact: true }).click();
  }
  if (item.name.includes("integrations")) {
    await page.getByRole("button", { name: "Agent/API", exact: true }).click();
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
