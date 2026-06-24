import { expect, test } from "@playwright/test";

const demoUrl = `file://${Deno.cwd()}/site/demo/index.html`;

test("static demo supports theme, user, local state, and reset flows", async ({ page }) => {
  await page.goto(demoUrl);

  await expect(page.getByRole("heading", {
    name: "Post a task, set the reward, review results.",
  })).toBeVisible();

  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("button", { name: "Dark" }).click();
  await page.getByRole("button", { name: /Blocky/ }).click();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "dark");
  await expect(page.locator("body")).toHaveAttribute("data-theme", "blocky");

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("jules");
  await page.getByRole("button", { name: "Dashboard", exact: true }).click();
  await expect(page.getByRole("heading", {
    name: "Find a task, do the work, get paid.",
  })).toBeVisible();

  await page.getByRole("button", { name: "Post Task", exact: true }).click();
  await page.getByLabel("Task title").fill("");
  await page.getByLabel("Task title").pressSequentially(
    "Demo persistence task",
  );
  await page.getByLabel("Instructions (free-form)").fill("");
  await page.getByLabel("Instructions (free-form)").pressSequentially(
    "A local demo task created by typing normally.",
  );
  await page.getByRole("button", { name: "Create draft task" }).click();
  await expect(page.getByRole("button", { name: /Demo persistence task/ }))
    .toBeVisible();

  await page.reload();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "dark");
  await expect(page.locator("body")).toHaveAttribute("data-theme", "blocky");
  await page.getByRole("button", { name: "Post Task", exact: true }).click();
  await expect(page.getByRole("button", { name: /Demo persistence task/ }))
    .toBeVisible();

  // The reset control is always visible in the top bar, not buried in Settings.
  await page.getByTestId("topbar-reset").click();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "light");
  await expect(page.locator("body")).toHaveAttribute("data-theme", "showcase");
  await expect(page.getByRole("button", { name: /Demo persistence task/ }))
    .toHaveCount(0);
});

test("static demo supports mission state transitions", async ({ page }) => {
  await page.goto(demoUrl);

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("jules");

  await page.getByRole("button", { name: "Tasks", exact: true })
    .click();
  await page.getByRole("button", { name: /Standardize map-tile region names/ })
    .click();
  await expect(
    page.getByRole("heading", {
      level: 1,
      name: "Standardize map-tile region names",
    }),
  )
    .toBeVisible();
  await page.getByRole("button", { name: "Reserve task" }).click();
  await expect(
    page.getByText("Jules Park reserved Standardize map-tile region names.")
      .first(),
  )
    .toBeVisible();

  await page.getByLabel("Your response").fill(
    '{"region":"North Vale","quality":92}',
  );
  await page.getByRole("button", { name: "Submit response" }).click();
  await expect(
    page.getByText(
      "Jules Park submitted a response for Standardize map-tile region names.",
    ).first(),
  )
    .toBeVisible();

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("mara");
  await page.getByRole("button", { name: "Reviews", exact: true }).click();
  await page.getByLabel("Selected task").selectOption("map-sensor-cleanup");
  await page.getByRole("button", { name: "Accept" }).click();
  await expect(
    page.getByText(
      /Mara Chen accepted Jules Park on Standardize map-tile region names/,
    ).first(),
  )
    .toBeVisible();
});

test("static demo links to task and user pages by URL", async ({ page }) => {
  // Deep-linking a task page loads it directly.
  await page.goto(`${demoUrl}#/tasks/invoice-cleanup`);
  await expect(
    page.getByRole("heading", { level: 2, name: "Extract invoice totals" }),
  ).toBeVisible();

  // The requester name links to that user's profile page and the URL reflects it.
  await page.locator('[data-user-page="mara"]').first().click();
  await expect(page).toHaveURL(/#\/users\/mara$/);
  await expect(
    page.getByRole("heading", { level: 2, name: "Mara Chen" }),
  ).toBeVisible();

  // Deep-linking a user page directly loads it.
  await page.goto(`${demoUrl}#/users/jules`);
  await expect(
    page.getByRole("heading", { level: 2, name: "Jules Park" }),
  ).toBeVisible();
});

test("static demo keeps review decisions persona-scoped", async ({ page }) => {
  await page.goto(demoUrl);

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("jules");
  await page.getByRole("button", { name: "Reviews", exact: true }).click();
  await expect(
    page.getByText(
      "No reservations or submitted responses need this persona's review.",
    ),
  )
    .toBeVisible();
  await expect(page.getByRole("button", { name: "Accept" })).toHaveCount(0);

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("mara");
  await page.getByRole("button", { name: "Reviews", exact: true }).click();
  await page.getByLabel("Selected task").selectOption("orchard-labels");
  await page.getByLabel("Tip").fill("99");
  await page.getByRole("button", { name: "Request changes" }).click();
  await expect(
    page.getByText(
      "Mara Chen requested changes from Jules Park on Label orchard photos.",
    ),
  ).toBeVisible();

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("jules");
  await page.getByRole("button", { name: "Tasks", exact: true })
    .click();
  await page.getByRole("button", { name: /Label orchard photos/ }).click();
  await expect(page.getByText("Task page")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Revise response" }))
    .toBeVisible();
});
