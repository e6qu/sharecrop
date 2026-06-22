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
  await page.getByRole("button", { name: item.mode }).click();
  await page.getByRole("button", { name: new RegExp(item.themeLabel) }).click();
  await page.screenshot({
    path: `${outputDirectory}/${item.name}.png`,
    fullPage: true,
  });
  await page.close();
}
await browser.close();
