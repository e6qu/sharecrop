import { expect, test } from "@playwright/test";
import { type AuthBody, password, uniqueEmail } from "./helpers.ts";

interface CollectibleBody {
  id: string;
}

interface TeamBody {
  id: string;
}

test("task creation uses directory selectors and funds selected collectible rewards", async ({ page, request }) => {
  const ownerEmail = uniqueEmail("aaa-ui-selector-owner");
  const targetEmail = uniqueEmail("aaa-ui-selector-target");

  const ownerResponse = await request.post("/api/auth/register", {
    data: { email: ownerEmail, password },
  });
  expect(ownerResponse.ok()).toBeTruthy();
  const owner = (await ownerResponse.json()) as AuthBody;

  const targetResponse = await request.post("/api/auth/register", {
    data: { email: targetEmail, password },
  });
  expect(targetResponse.ok()).toBeTruthy();
  const target = (await targetResponse.json()) as AuthBody;

  const teamResponse = await request.post("/api/teams", {
    headers: { Authorization: `Bearer ${owner.access_token}` },
    data: { name: `Selectors ${crypto.randomUUID()}` },
  });
  expect(teamResponse.ok()).toBeTruthy();
  const team = (await teamResponse.json()) as TeamBody;

  const collectibleName = `Selector medal ${crypto.randomUUID()}`;
  const collectibleResponse = await request.post("/api/collectibles", {
    headers: { Authorization: `Bearer ${owner.access_token}` },
    data: {
      name: collectibleName,
      kind: "badge",
      transfer_policy: "non_transferable_except_payout",
    },
  });
  expect(collectibleResponse.ok()).toBeTruthy();
  const collectible = (await collectibleResponse.json()) as CollectibleBody;

  await page.goto("/");
  await page.getByTestId("email").fill(ownerEmail);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await page.getByTestId("nav-create-task").click();
  const title = `Selector reward task ${crypto.randomUUID()}`;
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill(
    "Created through selector controls.",
  );

  await page.getByTestId("create-reward-kind-collectible").click();
  await page.getByTestId(`create-reward-collectible-${collectible.id}`).check();

  await page.getByTestId("create-visibility-user").click();
  await page.getByTestId("create-scope-user").selectOption({
    label: targetEmail,
  });
  await expect(page.getByTestId("create-scope-user")).toHaveValue(
    target.subject_id,
  );

  await page.getByTestId("create-visibility-team").click();
  await page.getByTestId("create-scope-team").selectOption(team.id);
  await expect(page.getByTestId("create-scope-team")).toHaveValue(team.id);

  await page.getByTestId("create-visibility-user").click();
  await page.getByTestId("create-scope-user").selectOption({
    label: targetEmail,
  });
  await page.getByTestId("create-task").click();
  await expect(page.getByTestId("create-message")).toContainText(
    "Created task",
  );

  await page.getByTestId("nav-tasks").click();
  await expect(page.getByTestId("task-row").filter({ hasText: title }))
    .toContainText(
      "1 collectible",
    );
});
