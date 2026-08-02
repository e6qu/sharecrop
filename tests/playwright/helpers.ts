import { type APIRequestContext, type Page } from "@playwright/test";

export interface AuthBody {
  access_token: string;
  subject_id: string;
}

export interface TaskBody {
  id: string;
}

export const password = "correct horse battery staple";

export function uniqueEmail(prefix: string): string {
  return `${prefix}-${crypto.randomUUID()}@example.com`;
}

// verifyAccountByApi completes email verification for a registered account
// over the API. The 100-credit signup grant lands at first verification (not
// at registration), so browser tests that assert the granted balance call
// this right after registering. It needs the test server's api token delivery
// (SHARECROP_ACCOUNT_TOKEN_DELIVERY=api) so the token appears in the
// response.
export async function verifyAccountByApi(
  request: APIRequestContext,
  email: string,
): Promise<void> {
  const login = await request.post("/api/auth/login", {
    data: { email, password },
  });
  if (!login.ok()) {
    throw new Error(
      `verification login failed for ${email}: ${await login.text()}`,
    );
  }
  const auth = (await login.json()) as AuthBody;
  await verifyAccountWithToken(request, auth.access_token);
}

// verifyAccountWithToken completes email verification with an access token
// already in hand (e.g. from an API registration response).
export async function verifyAccountWithToken(
  request: APIRequestContext,
  accessToken: string,
): Promise<void> {
  const issued = await request.post("/api/account/email-verification", {
    headers: { Authorization: `Bearer ${accessToken}` },
    data: {},
  });
  if (!issued.ok()) {
    throw new Error(`verification request failed: ${await issued.text()}`);
  }
  const token =
    ((await issued.json()) as { token: string | undefined }).token ?? "";
  if (token === "") {
    throw new Error(
      "verification token missing: the test server must run SHARECROP_ACCOUNT_TOKEN_DELIVERY=api",
    );
  }
  const confirmed = await request.post("/api/auth/email-verification/confirm", {
    data: { token },
  });
  if (!confirmed.ok()) {
    throw new Error(`verification confirm failed: ${await confirmed.text()}`);
  }
}

export function taskRequest(
  title: string,
  userId: string,
  visibilityKind: string,
  rewardAmount = 0,
): Record<string, unknown> {
  return {
    owner: { kind: "user", user_id: userId, team_id: "", organization_id: "" },
    title,
    description: "A task created from a browser test.",
    visibility: {
      kind: visibilityKind,
      user_id: "",
      team_id: "",
      organization_id: "",
    },
    placement: {
      kind: "standalone",
      series_id: "",
      series_title: "",
      series_position: 0,
    },
    reward: rewardAmount > 0
      ? { kind: "credit", credit_amount: rewardAmount }
      : { kind: "none", credit_amount: 0 },
    response_schema_json: '{"kind":"freeform"}',
    payload: { kind: "none", json: "" },
  };
}

// fillDetailResponse writes a raw JSON response into the task-detail submit
// form. Tasks with a structured response schema now render a per-field form
// by default (with a "raw JSON" escape hatch); tasks with a freeform schema
// keep the raw JSON textarea. This helper switches into raw mode when needed
// so existing tests can keep asserting on hand-written JSON responses.
export async function fillDetailResponse(
  page: Page,
  json: string,
): Promise<void> {
  // Wait for the submit form to render (it loads after navigation) before
  // deciding which editor is showing - the Submit button is present in both
  // the schema form and the raw editor. Without this, a .count() check can
  // run before the form appears and take the wrong branch.
  await page.getByTestId("detail-submit").waitFor();
  if ((await page.getByTestId("detail-submit-input").count()) === 0) {
    await page.getByTestId("submit-raw-toggle").click();
  }
  await page.getByTestId("detail-submit-input").fill(json);
}
