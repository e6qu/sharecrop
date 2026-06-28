import { expect, test } from "@playwright/test";
import { password, uniqueEmail } from "./helpers.ts";
import { accountLifecycleScenario } from "./scenarios.ts";

test("guest entry and account lifecycle controls work in the browser", async ({ page }) => {
  const email = uniqueEmail("ui-account");
  const { changedPassword, resetPassword } = accountLifecycleScenario;

  await page.goto("/");
  await page.getByTestId("guest-login").click();
  await expect(page.getByTestId("overview")).toBeVisible();
  await page.getByTestId("logout").click();

  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("register").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await page.getByTestId("nav-profile").click();
  await page.getByTestId("request-email-verification").click();
  await expect(page.getByTestId("account-message")).toContainText(
    "Verification token created.",
  );
  await page.getByTestId("confirm-email-verification").click();
  await expect(page.getByTestId("account-message")).toContainText(
    "Account updated.",
  );

  await page.getByTestId("current-password").fill(password);
  await page.getByTestId("new-password").fill(changedPassword);
  await page.getByTestId("change-password").click();
  await expect(page.getByTestId("account-message")).toContainText(
    "Account updated.",
  );

  await page.getByTestId("logout").click();
  await page.getByTestId("reset-email").fill(email);
  await page.getByTestId("request-password-reset").click();
  await expect(page.getByTestId("auth-error")).toContainText(
    "Password reset token created.",
  );
  await page.getByTestId("reset-password").fill(resetPassword);
  await page.getByTestId("confirm-password-reset").click();
  await expect(page.getByTestId("auth-error")).toContainText(
    "Password reset. Log in with the new password.",
  );

  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(resetPassword);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");
});
