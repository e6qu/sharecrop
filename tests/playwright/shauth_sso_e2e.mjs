// SPDX-License-Identifier: AGPL-3.0-or-later

import assert from "node:assert/strict";
import { chromium, request as playwrightRequest } from "playwright";

const issuer = requiredEnvironment("SHARECROP_SHAUTH_E2E_ISSUER");
const application = requiredEnvironment("SHARECROP_SHAUTH_E2E_APPLICATION");
const password = requiredEnvironment("SHAUTH_BOOTSTRAP_ADMIN_PASSWORD");
const screenshotDirectory = Deno.env.get(
  "SHARECROP_SHAUTH_E2E_SCREENSHOT_DIR",
);

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

  await page.goto(`${application}/auth/validation`);
  await page.getByRole("heading", { name: "Authenticated session" }).waitFor();
  assert.equal(
    (await page.getByTestId("validation-username").textContent())?.trim(),
    "admin",
  );
  assert.equal(
    (await page.getByTestId("validation-role").textContent())?.trim(),
    "admin",
  );
  assert.equal(
    (await page.getByTestId("validation-release").textContent())?.trim(),
    "0123456789ab",
  );
  await page.goto(`${application}/`);

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
  const recovery = page.getByRole("link", { name: "Sign in with Shauth" });
  await recovery.waitFor();
  assert.equal(await recovery.getAttribute("href"), "/api/auth/shauth");
  const lightBackground = await signedOutThemeBackground(page, "light");
  if (screenshotDirectory) {
    await Deno.mkdir(screenshotDirectory, { recursive: true });
    await page.screenshot({
      path: `${screenshotDirectory}/sharecrop-signed-out-light.png`,
      fullPage: true,
    });
  }
  const darkBackground = await signedOutThemeBackground(page, "dark");
  assert.notEqual(
    darkBackground,
    lightBackground,
    "signed-out light and dark themes rendered the same background",
  );
  if (screenshotDirectory) {
    await page.screenshot({
      path: `${screenshotDirectory}/sharecrop-signed-out-dark.png`,
      fullPage: true,
    });
  }
  await page.emulateMedia({ colorScheme: "light" });
  assert.equal(
    frontChannelLogoutSeen,
    true,
    "Shauth did not complete Sharecrop Front-Channel Logout",
  );

  // The application bridge never accepts a destination from the request. A
  // consumed or missing provider completion correlation ends at Shauth's safe
  // signed-out page rather than following attacker-controlled input.
  await page.goto(
    `${application}/auth/shauth/logout/complete?next=${
      encodeURIComponent("https://attacker.example/stolen")
    }&post_logout_redirect_uri=${
      encodeURIComponent("https://attacker.example/stolen")
    }&completion_token=replayed`,
  );
  await page.waitForURL(`${issuer}/signed-out`);
  assert.equal(page.url().startsWith("https://attacker.example"), false);
  await page.goto(`${application}/api/auth/signed-out`);
  await page.getByRole("heading", { name: "You are signed out" }).waitFor();

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

  // The application-owned page is a stable local destination: reload does not
  // restart OIDC, while its explicit same-origin control performs exact
  // recovery through Shauth when the user chooses to sign in again.
  await page.reload();
  await waitForExactURL(
    page,
    `${application}/api/auth/signed-out`,
    navigationTrace,
    browserErrors,
  );
  await page.getByRole("link", { name: "Sign in with Shauth" }).click();
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

  const providerLogoutRefresh = await context.request.post(
    `${application}/api/auth/refresh`,
  );
  assert.equal(providerLogoutRefresh.status(), 200);
  const providerLogoutSession = await providerLogoutRefresh.json();
  assert.ok(providerLogoutSession.access_token);
  const retainedRefreshCookie = (await context.cookies(application)).find(
    (cookie) => cookie.name === "sharecrop_refresh_token",
  );
  assert.ok(
    retainedRefreshCookie?.value,
    "Sharecrop refresh cookie is missing",
  );

  // Provider launch remains silent while the Shauth session is active.
  await page.goto(`${issuer}/apps`);
  await page.waitForURL(`${issuer}/apps`);

  // Provider-initiated logout is global: Shauth ends its own browser session,
  // notifies Sharecrop, and Sharecrop rejects both retained API and refresh
  // credentials. A direct launch then reaches Shauth's login page rather than
  // failing open into the application shell.
  await page.getByRole("link", { name: "Sign out", exact: true }).click();
  await page.waitForURL(`${issuer}/logout`);
  await page.getByRole("button", { name: "Sign out of all apps" }).click();
  await waitForExactURL(
    page,
    `${issuer}/signed-out`,
    navigationTrace,
    browserErrors,
  );

  const providerStaleAccess = await context.request.get(
    `${application}/api/credits/balance`,
    {
      headers: {
        authorization: `Bearer ${providerLogoutSession.access_token}`,
      },
    },
  );
  assert.equal(
    providerStaleAccess.status(),
    401,
    "provider logout retained Sharecrop API access",
  );

  const retainedSessionRequest = await playwrightRequest.newContext({
    extraHTTPHeaders: {
      cookie: `sharecrop_refresh_token=${retainedRefreshCookie.value}`,
    },
  });
  try {
    const retainedRefresh = await retainedSessionRequest.post(
      `${application}/api/auth/refresh`,
    );
    assert.equal(
      retainedRefresh.status(),
      401,
      "provider logout retained a Sharecrop refresh session",
    );
  } finally {
    await retainedSessionRequest.dispose();
  }

  await page.goto(`${application}/`);
  await page.waitForURL((url) =>
    url.origin === issuer && url.pathname === "/login"
  );
  assert.equal(page.url().includes("consent_challenge="), false);

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

async function signedOutThemeBackground(page, colorScheme) {
  await page.emulateMedia({ colorScheme });
  return await page.locator("body").evaluate((body) =>
    getComputedStyle(body).backgroundColor
  );
}
