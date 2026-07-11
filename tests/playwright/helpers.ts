import { type Page } from "@playwright/test";

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
