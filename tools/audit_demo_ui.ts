import { chromium, type ConsoleMessage, type Page } from "playwright";

type AuditIssue = {
  page: string;
  kind: string;
  message: string;
};

type AuditTarget = {
  name: string;
  url: string;
};

type LayoutAudit = {
  bodyWidth: number;
  clientWidth: number;
  appText: string;
};

const screenshotDirectory = "/tmp/sharecrop-demo-audit";
const targets: AuditTarget[] = [
  { name: "deployed-demo", url: "https://e6qu.github.io/sharecrop/demo/" },
  { name: "local-demo", url: `file://${Deno.cwd()}/site/demo/index.html` },
];

await Deno.mkdir(screenshotDirectory, { recursive: true });

const browser = await chromium.launch();
const issues: AuditIssue[] = [];

for (const target of targets) {
  await auditTarget(target);
}

await browser.close();

if (issues.length > 0) {
  for (const issue of issues) {
    console.error(`${issue.page}: ${issue.kind}: ${issue.message}`);
  }
  Deno.exit(1);
}

async function auditTarget(target: AuditTarget): Promise<void> {
  const page = await browser.newPage({
    viewport: { width: 1440, height: 1100 },
  });
  page.on("console", (message: ConsoleMessage) => {
    if (message.type() === "error" || message.type() === "warning") {
      issues.push({
        page: target.name,
        kind: `console:${message.type()}`,
        message: message.text(),
      });
    }
  });
  page.on("pageerror", (error: Error) => {
    issues.push({
      page: target.name,
      kind: "pageerror",
      message: error.message,
    });
  });
  page.on("requestfailed", (request) => {
    issues.push({
      page: target.name,
      kind: "requestfailed",
      message: `${request.method()} ${request.url()}`,
    });
  });

  const response = await page.goto(target.url, { waitUntil: "networkidle" });
  if (response?.ok() !== true) {
    issues.push({
      page: target.name,
      kind: "navigation",
      message: `${response?.status() ?? "no response"} ${target.url}`,
    });
  }

  await page.screenshot({
    path: `${screenshotDirectory}/${target.name}-desktop.png`,
    fullPage: true,
  });

  await checkLayout(page, target.name);
  await page.setViewportSize({ width: 390, height: 1100 });
  await page.reload({ waitUntil: "networkidle" });
  await page.screenshot({
    path: `${screenshotDirectory}/${target.name}-mobile.png`,
    fullPage: true,
  });
  await checkLayout(page, `${target.name}-mobile`);
  await page.close();
}

async function checkLayout(page: Page, name: string): Promise<void> {
  const rawLayout = await page.evaluate(`(() => {
    const root = document.documentElement;
    return {
      bodyWidth: document.body.scrollWidth,
      clientWidth: root.clientWidth,
      appText: document.body.innerText.slice(0, 120),
    };
  })()`);
  const overflow = rawLayout as LayoutAudit;

  if (overflow.bodyWidth > overflow.clientWidth + 2) {
    issues.push({
      page: name,
      kind: "layout",
      message:
        `horizontal overflow ${overflow.bodyWidth} > ${overflow.clientWidth}`,
    });
  }

  if (!overflow.appText.includes("Sharecrop")) {
    issues.push({
      page: name,
      kind: "content",
      message: "missing Sharecrop text",
    });
  }
}
