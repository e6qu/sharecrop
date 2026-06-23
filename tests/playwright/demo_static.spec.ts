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

  await page.getByRole("button", { name: /Persona/ }).click();
  await page.getByLabel("Select persona").selectOption("jules");
  await page.getByRole("button", { name: "Command", exact: true }).click();
  await expect(page.getByRole("heading", {
    name: "Pick a mission, claim the slot, and deliver the payload.",
  })).toBeVisible();

  await page.getByRole("button", { name: "Post Mission", exact: true }).click();
  await page.getByLabel("Mission title").fill("");
  await page.getByLabel("Mission title").pressSequentially(
    "Demo persistence task",
  );
  await page.getByLabel("Objective").fill("");
  await page.getByLabel("Objective").pressSequentially(
    "A local demo task created by typing normally.",
  );
  await page.getByRole("button", { name: "Draft mission" }).click();
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

  await page.getByRole("button", { name: /Persona/ }).click();
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

  await page.getByRole("button", { name: /Persona/ }).click();
  await page.getByLabel("Select persona").selectOption("mara");
  await page.getByRole("button", { name: "Review Queue", exact: true }).click();
  await page.locator("select.task-select").selectOption("map-sensor-cleanup");
  await page.getByRole("button", { name: "Accept" }).click();
  await expect(
    page.getByText(
      /Mara Chen accepted Jules Park on Normalize sensor map tiles/,
    ).first(),
  )
    .toBeVisible();
});
