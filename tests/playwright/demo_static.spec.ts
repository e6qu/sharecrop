import { expect, test } from "@playwright/test";

const demoUrl = `file://${Deno.cwd()}/site/demo/index.html`;

test("static demo supports theme, user, local state, and reset flows", async ({ page }) => {
  await page.goto(demoUrl);

  await expect(page.getByRole("heading", {
    name: "Command the work board, fund rewards, and settle results.",
  })).toBeVisible();

  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("button", { name: "Dark" }).click();
  await page.getByRole("button", { name: /Blocky/ }).click();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "dark");
  await expect(page.locator("body")).toHaveAttribute("data-theme", "blocky");

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("jules");
  await page.getByRole("button", { name: "Command", exact: true }).click();
  await expect(page.getByRole("heading", {
    name: "Pick a mission, claim the slot, and deliver the payload.",
  })).toBeVisible();

  await page.getByRole("button", { name: "Post Mission", exact: true }).click();
  await page.getByLabel("Task title").fill("");
  await page.getByLabel("Task title").pressSequentially(
    "Demo persistence task",
  );
  await page.getByLabel("Objective").fill("");
  await page.getByLabel("Objective").pressSequentially(
    "A local demo task created by typing normally.",
  );
  await page.getByRole("button", { name: "Create draft task" }).click();
  await expect(page.getByRole("button", { name: /Demo persistence task/ }))
    .toBeVisible();

  await page.reload();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "dark");
  await expect(page.locator("body")).toHaveAttribute("data-theme", "blocky");
  await page.getByRole("button", { name: "Post Mission", exact: true }).click();
  await expect(page.getByRole("button", { name: /Demo persistence task/ }))
    .toBeVisible();

  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("button", { name: "Clear state" }).click();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "light");
  await expect(page.locator("body")).toHaveAttribute("data-theme", "blocky");
  await expect(page.getByRole("button", { name: /Demo persistence task/ }))
    .toHaveCount(0);
});

test("static demo supports mission state transitions", async ({ page }) => {
  await page.goto(demoUrl);

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("jules");

  await page.getByRole("button", { name: "Mission Board", exact: true })
    .click();
  await page.getByRole("button", { name: /Normalize sensor map tiles/ })
    .click();
  await page.getByRole("button", { name: "Reserve mission" }).click();
  await expect(
    page.getByText("Jules Park reserved Normalize sensor map tiles.").first(),
  )
    .toBeVisible();

  await page.getByLabel("Submission payload").fill(
    '{"region":"north","quality":92}',
  );
  await page.getByRole("button", { name: "Submit payload" }).click();
  await expect(
    page.getByText(
      "Jules Park submitted a payload for Normalize sensor map tiles.",
    ).first(),
  )
    .toBeVisible();

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("mara");
  await page.getByRole("button", { name: "Review Queue", exact: true }).click();
  await page.getByLabel("Selected task").selectOption("map-sensor-cleanup");
  await page.getByRole("button", { name: "Accept" }).click();
  await expect(
    page.getByText(
      /Mara Chen accepted Jules Park on Normalize sensor map tiles/,
    ).first(),
  )
    .toBeVisible();
});

test("static demo keeps review decisions persona-scoped", async ({ page }) => {
  await page.goto(demoUrl);

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("jules");
  await page.getByRole("button", { name: "Review Queue", exact: true }).click();
  await expect(
    page.getByText(
      "No reservations or submitted payloads need this persona's review.",
    ),
  )
    .toBeVisible();
  await expect(page.getByRole("button", { name: "Accept" })).toHaveCount(0);

  await page.locator(".account-button").click();
  await page.getByLabel("Select persona").selectOption("mara");
  await page.getByRole("button", { name: "Review Queue", exact: true }).click();
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
  await page.getByRole("button", { name: "Mission Board", exact: true })
    .click();
  await page.getByRole("button", { name: /Label orchard photos/ }).click();
  await expect(page.getByRole("heading", { name: "Revise payload" }))
    .toBeVisible();
});
