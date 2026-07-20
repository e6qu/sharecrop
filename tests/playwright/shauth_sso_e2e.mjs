// SPDX-License-Identifier: AGPL-3.0-or-later

import assert from "node:assert/strict";
import { chromium } from "playwright";

const issuer = requiredEnvironment("SHARECROP_SHAUTH_E2E_ISSUER");
const application = requiredEnvironment("SHARECROP_SHAUTH_E2E_APPLICATION");
const password = requiredEnvironment("SHAUTH_BOOTSTRAP_ADMIN_PASSWORD");

const browser = await chromium.launch({ headless: true });
try {
  const context = await browser.newContext();
  const page = await context.newPage();
  const browserErrors = [];
  const navigationTrace = [];
  let frontChannelLogoutSeen = false;
  page.on("console", (message) => {
    if (message.type() === "error") browserErrors.push(message.text());
  });
  page.on("pageerror", (error) => browserErrors.push(error.message));
  page.on("requestfailed", (request) => {
    browserErrors.push(
      `${sanitizeURL(request.url())}: ${
        request.failure()?.errorText ?? "request failed"
      }`,
    );
  });
  page.on("request", (request) => {
    if (request.isNavigationRequest()) {
      navigationTrace.push(
        `request ${request.method()} ${sanitizeURL(request.url())}`,
      );
    }
  });
  page.on("response", (response) => {
    const responseURL = new URL(response.url());
    if (
      responseURL.origin === application &&
      responseURL.pathname === "/api/auth/shauth/frontchannel-logout"
    ) {
      frontChannelLogoutSeen = response.ok();
    }
    if (response.request().isNavigationRequest()) {
      navigationTrace.push(
        `response ${response.status()} ${sanitizeURL(response.url())}`,
      );
    }
  });

  // Portal launch: authenticate once at Shauth, then open Sharecrop from the
  // provider-owned app catalog without another login or consent form.
  await page.goto(`${issuer}/apps`);
  await page.waitForURL((url) =>
    url.origin === issuer && url.pathname === "/login"
  );
  await page.locator("#username").fill("admin");
  await page.locator("#password").fill(password);
  await page.getByRole("button", { name: "Sign in with password" }).click();
  await page.waitForURL(`${issuer}/apps`);
  await page.getByRole("link", { name: "Open Sharecrop" }).click();
  await waitForApplication(page, navigationTrace, browserErrors);
  await page.waitForLoadState("networkidle");
  await page.waitForURL(`${application}/`);
  const firstRefresh = await context.request.post(
    `${application}/api/auth/refresh`,
  );
  assert.equal(
    firstRefresh.status(),
    200,
    "Sharecrop could not rotate its SSO session",
  );
  const firstSession = await firstRefresh.json();
  assert.ok(
    firstSession.access_token,
    "Sharecrop refresh omitted its access token",
  );
  assert.equal(await page.getByTestId("login").count(), 0);
  assert.equal(await page.getByTestId("register").count(), 0);
  assert.equal(await page.getByTestId("reset-email").count(), 0);

  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-profile").click();
  await page.getByText("Account settings").waitFor();
  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("logout").click();
  await waitForExactURL(
    page,
    `${application}/api/auth/signed-out`,
    navigationTrace,
    browserErrors,
  );
  await page.getByRole("heading", { name: "You are signed out" }).waitFor();
  assert.equal(
    frontChannelLogoutSeen,
    true,
    "Shauth did not complete Sharecrop Front-Channel Logout",
  );

  const staleAccess = await context.request.get(
    `${application}/api/credits/balance`,
    {
      headers: { authorization: `Bearer ${firstSession.access_token}` },
    },
  );
  assert.equal(
    staleAccess.status(),
    401,
    "a signed-out browser retained API access",
  );

  await page.goto(`${issuer}/apps`);
  await page.waitForURL((url) =>
    url.origin === issuer && url.pathname === "/login"
  );

  // Direct launch: Sharecrop must redirect the signed-out browser to Shauth,
  // and successful provider authentication must return to Sharecrop itself.
  await page.goto(`${application}/`);
  await page.waitForURL((url) =>
    url.origin === issuer && url.pathname === "/login"
  );
  assert.equal(page.url().includes("consent_challenge="), false);
  await page.locator("#username").fill("admin");
  await page.locator("#password").fill(password);
  await page.getByRole("button", { name: "Sign in with password" }).click();
  await waitForApplication(page, navigationTrace, browserErrors);
  await page.waitForLoadState("networkidle");
  await page.waitForURL(`${application}/`);
  assert.equal(await page.getByTestId("shauth-login").count(), 0);
  await page.goto(`${issuer}/apps`);
  await page.waitForURL(`${issuer}/apps`);

  assert.deepEqual(browserErrors, [], navigationTrace.join("\n"));
} finally {
  await browser.close();
}

function requiredEnvironment(name) {
  const value = process.env[name];
  assert.ok(value, `${name} is required`);
  return value.replace(/\/$/, "");
}

function sanitizeURL(value) {
  const parsed = new URL(value);
  return `${parsed.origin}${parsed.pathname}`;
}

async function waitForExactURL(page, expected, trace, errors) {
  const deadline = Date.now() + 30_000;
  while (page.url() !== expected && Date.now() < deadline) {
    await page.waitForTimeout(100);
  }
  assert.equal(
    page.url(),
    expected,
    [...trace, ...errors.map((error) => `browser error ${error}`)].join("\n"),
  );
}

async function waitForApplication(page, trace, errors) {
  try {
    await page.getByTestId("balance").waitFor();
  } catch (error) {
    const body = await page.locator("body").innerText().catch(() =>
      "<body unavailable>"
    );
    console.error(
      [
        `final URL ${page.url()}`,
        ...trace,
        ...errors.map((message) => `browser error ${message}`),
        `body ${body}`,
      ].join("\n"),
    );
    throw error;
  }
}
