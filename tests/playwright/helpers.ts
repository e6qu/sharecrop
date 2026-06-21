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

export function userTaskRequest(
  title: string,
  userId: string,
): Record<string, unknown> {
  return {
    owner: { kind: "user", user_id: userId, team_id: "", organization_id: "" },
    title,
    description: "A task created from a browser test.",
    visibility: {
      kind: "default",
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
    response_schema_json: '{"kind":"freeform"}',
    payload: { kind: "none", json: "" },
  };
}
