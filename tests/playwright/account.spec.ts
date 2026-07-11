import { expect, test } from "@playwright/test";
import { password, uniqueEmail } from "./helpers.ts";
import { accountLifecycleScenario } from "./scenarios.ts";

test("guest entry and account lifecycle controls work in the browser", async ({ page }) => {
  const email = uniqueEmail("ui-account");
  const { changedPassword, resetPassword } = accountLifecycleScenario;

  await page.goto("/");
  // Guest sessions only work against the demo backend: the real API rejects
  // the guest subject on every data route, so the real app hides the button
  // instead of offering a dead end.
  await expect(page.getByTestId("login")).toBeVisible();
  await expect(page.getByTestId("guest-login")).toHaveCount(0);

  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("register").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-profile").click();
  await page.getByTestId("account-email-verification").click();
  await page.getByTestId("request-email-verification").click();
  await expect(page.getByTestId("account-message")).toContainText(
    "Verification token created.",
  );
  await page.getByTestId("confirm-email-verification").click();
  await expect(page.getByTestId("account-message")).toContainText(
    "Account updated.",
  );

  await page.getByTestId("account-password").click();
  await page.getByTestId("current-password").fill(password);
  await page.getByTestId("new-password").fill(changedPassword);
  await page.getByTestId("change-password").click();
  await expect(page.getByTestId("account-message")).toContainText(
    "Account updated.",
  );

  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("logout").click();
  await page.getByTestId("reset-email").fill(email);
  await page.getByTestId("request-password-reset").click();
  // Successes render in the auth-notice slot (green), not auth-error.
  await expect(page.getByTestId("auth-notice")).toContainText(
    "Password reset token created",
  );
  await page.getByTestId("reset-password").fill(resetPassword);
  await page.getByTestId("confirm-password-reset").click();
  await expect(page.getByTestId("auth-notice")).toContainText(
    "Password reset. Log in with the new password.",
  );

  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(resetPassword);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");
});
