const storageKey = "sharecrop-demo-state-v8";

const productTagline = "Sharecrop is a reverse MCP: post a task with a response schema, and a person — or their agent connecting over MCP/REST — performs it and returns a structured result you review and pay for.";

const policy = {
  open: "open",
  reservation: "reservation",
  approval: "approval",
};

const lifecycle = {
  draft: "draft",
  funded: "funded",
  open: "open",
  closed: "closed",
  canceled: "canceled",
};

const availability = {
  available: "available",
  reserved: "reserved",
  awaitingApproval: "awaiting_approval",
  submitted: "submitted",
  changesRequested: "changes_requested",
  rejected: "rejected",
  accepted: "accepted",
  closed: "closed",
};

const pages = [
  { id: "overview", label: "Dashboard" },
  { id: "discover", label: "Tasks" },
  { id: "task", label: "Task Detail", detailOnly: true },
  { id: "requester", label: "Post Task" },
  { id: "review", label: "Reviews" },
  { id: "integrations", label: "Agent/API" },
  { id: "settings", label: "Settings" },
  { id: "user", label: "Profile", detailOnly: true },
];

const users = [
  { id: "mara", name: "Mara Chen", role: "Requester", balance: 180 },
  { id: "jules", name: "Jules Park", role: "Implementor", balance: 64 },
  { id: "ren", name: "Ren Ito", role: "Organization reviewer", balance: 112 },
  { id: "sol", name: "Sol Rivera", role: "Agent operator", balance: 100 },
  { id: "tala", name: "Tala Stone", role: "Implementor", balance: 83 },
];

const providers = ["Google", "Apple", "Microsoft", "Facebook", "X.com"];
const maxLocalTasks = 6;
const maxActivity = 12;

const seedTasks = [
  task({
    id: "orchard-labels",
    title: "Label orchard photos",
    requester: "mara",
    assignee: "jules",
    area: "Field Ops",
    visibility: "public",
    policy: policy.reservation,
    reward: rewardBundle(45, ["Ripe Lens"]),
    lifecycle: lifecycle.open,
    availability: availability.submitted,
    objective: "Classify each of the 6 fruit observations below as ripe, unripe, or damaged. Return one label per observation, in order, in the labels array. Accepted when all 6 have a label from that set.",
    schema: '{"kind":"object","fields":{"labels":{"kind":"array","items":{"kind":"string","enum":["ripe","unripe","damaged"]},"required":true}}}',
    inputs: [{
      kind: "list",
      label: "6 fruit observations to classify, in order:",
      items: [
        "Deep red skin, firm to the touch, no blemishes",
        "Mostly green, very hard, no give",
        "Fully colored with a soft brown bruise on one side",
        "Evenly colored, slight give when pressed",
        "Pale and small, skin still tight and hard",
        "Red but the skin is split with a mold spot",
      ],
    }],
    reservations: [{ id: "res-orchard-jules", by: "jules", state: "active", expires: "48h" }],
    submissions: [{
      id: "sub-orchard-jules",
      by: "jules",
      state: "submitted",
      response: '{"labels":["ripe","unripe","damaged","ripe","unripe","damaged"]}',
      reviewNote: "",
      partialPayout: "",
      tip: "",
      ban: false,
    }],
    timeline: ["Mara posted the labeling task.", "Jules reserved the task.", "Jules submitted a labeled response."],
  }),
  task({
    id: "invoice-cleanup",
    title: "Extract invoice totals",
    requester: "mara",
    assignee: "",
    area: "Ledger Bay",
    visibility: "organization",
    policy: policy.approval,
    reward: rewardCredits(30),
    lifecycle: lifecycle.open,
    availability: availability.awaitingApproval,
    objective: "Add up the grand totals of the 5 invoices below. Submit the combined amount as a decimal string in total, for example 1240.50. Accepted when it matches the verified sum within 0.01.",
    schema: '{"kind":"object","fields":{"total":{"kind":"decimal_string"}}}',
    inputs: [{
      kind: "records",
      label: "5 invoices — sum the grand totals:",
      columns: ["Invoice", "Vendor", "Grand total"],
      rows: [
        ["INV-1001", "Birch Supply", "312.40"],
        ["INV-1002", "Cedar Freight", "88.10"],
        ["INV-1003", "Delta Print", "146.00"],
        ["INV-1004", "Ferro Metals", "502.75"],
        ["INV-1005", "Grove Cafe", "41.25"],
      ],
    }],
    reservations: [{ id: "res-invoice-ren", by: "ren", state: "requested", expires: "24h" }],
    timeline: ["Ren requested clearance to work on invoice extraction."],
  }),
  task({
    id: "badge-copy",
    title: "Draft badge copy",
    requester: "ren",
    assignee: "",
    area: "Foundry",
    visibility: "public",
    policy: policy.open,
    reward: rewardNone(),
    lifecycle: lifecycle.open,
    availability: availability.available,
    objective: "Propose 5 short name ideas (max 3 words each) for the achievement described below. Submit them as plain text, one per line. Accepted when at least 5 distinct, on-brand names are provided.",
    schema: '{"kind":"freeform"}',
    inputs: [{
      kind: "text",
      label: "Achievement to name:",
      body: "A contributor has had 10 submissions accepted across public tasks. Tone: encouraging and short. No numbers in the name.",
    }],
    timeline: ["Ren opened the brief for public submissions."],
  }),
  task({
    id: "map-sensor-cleanup",
    title: "Standardize map-tile region names",
    requester: "mara",
    assignee: "",
    area: "Cartography",
    visibility: "public",
    policy: policy.reservation,
    reward: rewardCredits(80),
    lifecycle: lifecycle.open,
    availability: availability.available,
    objective: "The region values below are all meant to be the same place, spelled inconsistently. Pick the single canonical name (from the canonical options) and rate the data quality from 0 to 100. Submit region and quality.",
    schema: '{"kind":"object","fields":{"region":{"kind":"string"},"quality":{"kind":"integer"}}}',
    inputs: [{
      kind: "list",
      label: "Region values found in the file (canonical options: North Vale, South Vale, East Vale):",
      items: ["Nørth Vale", "north vale", "N. Vale", "Northvale", "north  vale", "North Vale"],
    }],
    timeline: ["Task opened with the reward held in escrow."],
  }),
  task({
    id: "collectible-audit",
    title: "Verify collectible transfers",
    requester: "ren",
    assignee: "tala",
    area: "Vault",
    visibility: "organization",
    policy: policy.reservation,
    reward: rewardCollectible(["Vault Seal"]),
    lifecycle: lifecycle.open,
    availability: availability.changesRequested,
    objective: "Review the transfer ledger below and flag any rows that look fraudulent — the same item moved twice, or a transfer to a banned account. Submit the transfer ids to investigate in suspicious_ids.",
    schema: '{"kind":"object","fields":{"suspicious_ids":{"kind":"array","items":{"kind":"string"}}}}',
    inputs: [{
      kind: "records",
      label: "Transfer ledger (accounts starting with banned- are blocked):",
      columns: ["Transfer", "Item", "From", "To"],
      rows: [
        ["cx-17", "Vault Seal #2", "mara", "jules"],
        ["cx-18", "Field Badge #9", "tala", "ren"],
        ["cx-19", "Vault Seal #2", "jules", "sol"],
        ["cx-20", "Storm Pin #1", "ren", "banned-0042"],
        ["cx-21", "Drone Patch #4", "sol", "tala"],
      ],
    }],
    reservations: [{ id: "res-audit-tala", by: "tala", state: "active", expires: "12h" }],
    submissions: [{
      id: "sub-audit-tala",
      by: "tala",
      state: "changes_requested",
      response: '{"suspicious_ids":["cx-19"]}',
      reviewNote: "Include the second transfer batch before final review.",
      partialPayout: "",
      tip: "",
      ban: false,
    }],
    timeline: ["Tala submitted an audit pass.", "Ren requested a second batch check."],
  }),
  task({
    id: "agent-weather-json",
    title: "Agent weather JSON",
    requester: "mara",
    assignee: "sol",
    area: "Uplink",
    visibility: "public",
    policy: policy.approval,
    reward: rewardBundle(25, ["Storm Pin"]),
    lifecycle: lifecycle.open,
    availability: availability.submitted,
    objective: "Reverse-MCP task: point a scoped agent at a weather tool and fetch the current temperature in Celsius for the three cities below, in order. Submit them as decimal strings in readings. Accepted when three plausible readings are present.",
    schema: '{"kind":"object","fields":{"readings":{"kind":"array","items":{"kind":"decimal_string"}}}}',
    inputs: [{
      kind: "list",
      label: "Fetch current temperature (°C) for these cities, in order:",
      items: ["Lisbon, PT", "Nairobi, KE", "Osaka, JP"],
    }],
    reservations: [{ id: "res-weather-sol", by: "sol", state: "active", expires: "8h" }],
    submissions: [{
      id: "sub-weather-sol",
      by: "sol",
      state: "submitted",
      response: '{"readings":["21.4","20.9","22.1"],"source":"agent"}',
      reviewNote: "",
      partialPayout: "",
      tip: "",
      ban: false,
    }],
    timeline: ["Sol created a scoped agent token.", "The agent submitted readings over MCP."],
  }),
  task({
    id: "market-tags",
    title: "Tag public market tasks",
    requester: "mara",
    assignee: "",
    area: "Market",
    visibility: "public",
    policy: policy.open,
    reward: rewardCredits(12),
    lifecycle: lifecycle.open,
    availability: availability.available,
    objective: "Choose exactly two category tags for each of the five task descriptions below, using only the allowed tags. Submit them in the tags array, grouped per task in order.",
    schema: '{"kind":"object","fields":{"tags":{"kind":"array","items":{"kind":"string"}}}}',
    inputs: [
      { kind: "text", label: "Allowed tags:", body: "data, images, writing, code, audio, research" },
      {
        kind: "list",
        label: "Tasks to tag (two tags each, in order):",
        items: [
          "Transcribe a 3-minute recorded interview",
          "Clean a CSV of product prices",
          "Write release notes from a changelog",
          "Label a set of street-sign photos",
          "Summarize three research abstracts",
        ],
      },
    ],
    timeline: ["Low-risk task opened for open submissions."],
  }),
  task({
    id: "review-denied-batch",
    title: "Rejected crop batch",
    requester: "mara",
    assignee: "jules",
    area: "Field Ops",
    visibility: "public",
    policy: policy.open,
    reward: rewardCredits(18),
    lifecycle: lifecycle.open,
    availability: availability.rejected,
    objective: "Read the inspection report below and write a 2 to 3 sentence summary of why the batch failed and what to fix. Submit as plain text. Accepted when the summary names the failing metric.",
    schema: '{"kind":"freeform"}',
    inputs: [{
      kind: "text",
      label: "Inspection report:",
      body: "Batch B-204 — moisture 18.2% (limit 14.0%), foreign material 0.1% (within limit), color grade A (within limit). Disposition: rejected.",
    }],
    submissions: [{
      id: "sub-denied-jules",
      by: "jules",
      state: "rejected",
      response: '{"summary":"batch failed moisture range"}',
      reviewNote: "Moisture values were missing from the result.",
      partialPayout: "6",
      tip: "0",
      ban: false,
    }],
    timeline: ["Jules submitted a summary.", "Mara rejected the result and paid 6 credits."],
  }),
  task({
    id: "closed-orchard-archive",
    title: "Archive orchard receipts",
    requester: "mara",
    assignee: "tala",
    area: "Archive",
    visibility: "organization",
    policy: policy.reservation,
    reward: rewardCredits(20),
    lifecycle: lifecycle.closed,
    availability: availability.accepted,
    objective: "Add each of the 3 accepted receipts below to the archive index, one row per receipt with date, vendor, and amount. Submit a short note confirming the count archived. Accepted when the index row count matches.",
    schema: '{"kind":"freeform"}',
    inputs: [{
      kind: "records",
      label: "Receipts to archive:",
      columns: ["Date", "Vendor", "Amount"],
      rows: [
        ["2026-05-02", "Birch Supply", "312.40"],
        ["2026-05-03", "Cedar Freight", "88.10"],
        ["2026-05-05", "Grove Cafe", "41.25"],
      ],
    }],
    submissions: [{
      id: "sub-archive-tala",
      by: "tala",
      state: "accepted",
      response: '{"archived":true,"count":42}',
      reviewNote: "Accepted.",
      partialPayout: "20",
      tip: "3",
      ban: false,
    }],
    timeline: ["Tala delivered the archive index.", "Mara accepted and tipped 3 credits."],
  }),
  task({
    id: "draft-drone-footage",
    title: "Sort drone footage",
    requester: "mara",
    assignee: "",
    area: "Hangar",
    visibility: "public",
    policy: policy.reservation,
    reward: rewardBundle(60, ["Drone Patch"]),
    lifecycle: lifecycle.draft,
    availability: availability.available,
    objective: "Group the drone clips below by field and weather (for example field-3-clear) and submit the grouped clip identifiers in clips. Accepted when every clip is grouped.",
    schema: '{"kind":"object","fields":{"clips":{"kind":"array","items":{"kind":"string"}}}}',
    inputs: [{
      kind: "records",
      label: "Drone clips with metadata:",
      columns: ["Clip", "Field", "Weather"],
      rows: [
        ["clip-1", "field-3", "clear"],
        ["clip-2", "field-1", "rain"],
        ["clip-3", "field-3", "clear"],
        ["clip-4", "field-1", "rain"],
        ["clip-5", "field-2", "fog"],
      ],
    }],
    timeline: ["Draft task waiting for funding and opening."],
  }),
  task({
    id: "funded-api-docs",
    title: "Polish API examples",
    requester: "ren",
    assignee: "",
    area: "Docs Deck",
    visibility: "public",
    policy: policy.approval,
    reward: rewardCredits(22),
    lifecycle: lifecycle.funded,
    availability: availability.available,
    objective: "The API snippets below are broken. Rewrite them so they run correctly against the current API (base URL https://sharecrop.example, JSON bodies, bearer auth). Submit the corrected snippets as plain text.",
    schema: '{"kind":"freeform"}',
    inputs: [{
      kind: "code",
      label: "Broken snippets to fix:",
      body: "curl -X GET /api/tasks\ncurl -X POST /tasks/{id}/funding -d amount=20\nmcp: sharecrop.get_task(id)",
    }],
    timeline: ["Reward funded; requester has not opened the task."],
  }),
  task({
    id: "expired-reservation-demo",
    title: "Expired soil sample reservation",
    requester: "mara",
    assignee: "",
    area: "Lab",
    visibility: "public",
    policy: policy.reservation,
    reward: rewardCredits(26),
    lifecycle: lifecycle.open,
    availability: availability.available,
    objective: "From the soil-sample rows below, list the sample ids that are missing a moisture reading in the missing array. Accepted when every row with a blank moisture value is included and no others.",
    schema: '{"kind":"object","fields":{"missing":{"kind":"array","items":{"kind":"string"}}}}',
    inputs: [{
      kind: "records",
      label: "Soil samples (blank moisture = missing):",
      columns: ["Sample", "Moisture %"],
      rows: [
        ["s-01", "22.4"],
        ["s-02", ""],
        ["s-03", "19.0"],
        ["s-04", ""],
        ["s-05", "20.1"],
      ],
    }],
    reservations: [{ id: "res-soil-jules", by: "jules", state: "expired", expires: "0h" }],
    timeline: ["A previous reservation expired and released the task."],
  }),
];

function holdsEscrow(taskItem) {
  return (taskItem.lifecycle === lifecycle.funded || taskItem.lifecycle === lifecycle.open) &&
    (taskItem.reward.credits || 0) > 0 &&
    ![availability.accepted, availability.rejected, availability.closed].includes(taskItem.availability);
}

const seedEscrow = Object.fromEntries(
  seedTasks.filter(holdsEscrow).map((taskItem) => [taskItem.id, taskItem.reward.credits]),
);

const seedState = {
  mode: "light",
  theme: "showcase",
  page: "overview",
  loginOpen: false,
  userId: "mara",
  socialProvider: "",
  includeReserved: false,
  boardFilter: "all",
  selectedTaskId: "orchard-labels",
  selectedUserId: "mara",
  draftTitle: "Label orchard photos",
  draftDescription: "Classify each of the 6 fruit observations as ripe, unripe, or damaged. Return one label per observation, in order, in the labels array.",
  draftResponseKind: "structured",
  draftFields: [{ name: "labels", type: "string_list", required: true, enum: "ripe, unripe, damaged" }],
  draftRewardKind: "bundle",
  draftCredits: "45",
  draftCollectible: "Ripe Lens",
  draftPolicy: policy.reservation,
  draftVisibility: "public",
  draftReservationHours: "48",
  responseDrafts: {},
  reviewDrafts: {},
  localTaskSeq: 1,
  balances: Object.fromEntries(users.map((user) => [user.id, user.balance])),
  escrow: seedEscrow,
  inventories: {
    mara: ["Ripe Lens", "Drone Patch", "Storm Pin"],
    jules: ["Archive Token"],
    ren: ["Vault Seal"],
    sol: ["Agent Sigil"],
    tala: ["Field Badge"],
  },
  activityLog: [
    "Mara accepted Archive Orchard Receipts and tipped Tala 3 credits.",
    "Sol's agent submitted weather readings through MCP.",
    "Ren requested changes on Collectible Transfer Audit.",
    "Jules reserved Label Orchard Photos.",
  ],
  tasks: seedTasks,
};

let state = loadState();
let saveHandle = 0;

document.addEventListener("click", handleClick);
document.addEventListener("change", handleCommit);
document.addEventListener("input", handleDraftInput);
window.addEventListener("popstate", () => {
  applyHash();
  render();
});
window.addEventListener("hashchange", () => {
  applyHash();
  render();
});

applyHash();
render();

// hashFromState and applyHash keep the URL hash and the current page in sync so
// task and user pages are linkable, refreshable, and shareable on GitHub Pages.
function taskHref(taskId) {
  return `#/tasks/${encodeURIComponent(taskId)}`;
}

function userHref(userId) {
  return `#/users/${encodeURIComponent(userId)}`;
}

function hashFromState() {
  if (state.page === "task") return `#/tasks/${encodeURIComponent(state.selectedTaskId)}`;
  if (state.page === "user") return `#/users/${encodeURIComponent(state.selectedUserId)}`;
  if (state.page === "overview") return "#/";
  return `#/${state.page}`;
}

function applyHash() {
  const parts = window.location.hash.replace(/^#\/?/, "").split("/").filter(Boolean);
  if (parts.length === 0) {
    state = { ...state, page: "overview" };
    return;
  }
  if (parts[0] === "tasks" && parts[1]) {
    state = { ...state, page: "task", selectedTaskId: decodeURIComponent(parts[1]) };
    return;
  }
  if (parts[0] === "users" && parts[1]) {
    state = { ...state, page: "user", selectedUserId: decodeURIComponent(parts[1]) };
    return;
  }
  const known = pages.find((page) => page.id === parts[0] && !page.detailOnly);
  if (known) {
    state = { ...state, page: known.id };
  }
}

function task(config) {
  return {
    submissions: [],
    reservations: [],
    timeline: [],
    ...config,
  };
}

function rewardNone() {
  return { kind: "none", credits: 0, collectibles: [] };
}

function rewardCredits(credits) {
  return { kind: "credits", credits, collectibles: [] };
}

function rewardCollectible(collectibles) {
  return { kind: "collectible", credits: 0, collectibles };
}

function rewardBundle(credits, collectibles) {
  return { kind: "bundle", credits, collectibles };
}

function loadState() {
  const stored = readStoredState();
  if (stored === null) return structuredClone(seedState);

  try {
    const parsed = JSON.parse(stored);
    const nextState = {
      ...structuredClone(seedState),
      ...parsed,
      tasks: mergeTasks(parsed.tasks ?? []),
      balances: { ...structuredClone(seedState.balances), ...(parsed.balances ?? {}) },
      inventories: { ...structuredClone(seedState.inventories), ...(parsed.inventories ?? {}) },
      reviewDrafts: plainObject(parsed.reviewDrafts) ? parsed.reviewDrafts : {},
      activityLog: arrayOrEmpty(parsed.activityLog, seedState.activityLog).slice(0, maxActivity),
    };
    return normalizeState(nextState);
  } catch (_error) {
    return structuredClone(seedState);
  }
}

function mergeTasks(storedTasks) {
  const byId = new Map(seedTasks.map((item) => [item.id, item]));
  for (const storedTask of arrayOrEmpty(storedTasks).slice(0, seedTasks.length + maxLocalTasks)) {
    if (storedTask && typeof storedTask.id === "string") {
      byId.set(storedTask.id, normalizeTask({ ...byId.get(storedTask.id), ...storedTask }));
    }
  }
  return [...byId.values()];
}

function normalizeState(nextState) {
  const knownUser = users.some((user) => user.id === nextState.userId) ? nextState.userId : seedState.userId;
  const knownPage = pages.some((page) => page.id === nextState.page) ? nextState.page : seedState.page;
  const tasks = mergeTasks(nextState.tasks);
  const selectedTaskId = tasks.some((item) => item.id === nextState.selectedTaskId) ? nextState.selectedTaskId : tasks[0]?.id;
  return {
    ...nextState,
    userId: knownUser,
    page: knownPage,
    selectedTaskId,
    tasks,
  };
}

function normalizeTask(taskItem) {
  const reward = plainObject(taskItem.reward) ? taskItem.reward : rewardNone();
  return task({
    ...taskItem,
    reward: {
      kind: typeof reward.kind === "string" ? reward.kind : "none",
      credits: Number.isFinite(Number(reward.credits)) ? Number(reward.credits) : 0,
      collectibles: arrayOrEmpty(reward.collectibles).filter((item) => typeof item === "string"),
    },
    reservations: arrayOrEmpty(taskItem.reservations).filter((item) => plainObject(item)),
    submissions: arrayOrEmpty(taskItem.submissions).filter((item) => plainObject(item)),
    timeline: arrayOrEmpty(taskItem.timeline).filter((item) => typeof item === "string").slice(0, 5),
  });
}

function readStoredState() {
  try {
    return localStorage.getItem(storageKey);
  } catch (_error) {
    return null;
  }
}

function writeStoredState(value) {
  try {
    localStorage.setItem(storageKey, value);
    return true;
  } catch (_error) {
    return false;
  }
}

function removeStoredState() {
  try {
    localStorage.removeItem(storageKey);
  } catch (_error) {
    // The demo can still run without persistent browser storage.
  }
}

function arrayOrEmpty(value, replacement = []) {
  return Array.isArray(value) ? value : replacement;
}

function plainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function saveSoon() {
  clearTimeout(saveHandle);
  saveHandle = setTimeout(saveToStorage, 120);
}

function saveNow() {
  clearTimeout(saveHandle);
  saveToStorage();
}

function saveToStorage() {
  if (!writeStoredState(JSON.stringify(state))) {
    state = {
      ...structuredClone(seedState),
      page: "settings",
      storageWarning: "The browser rejected demo storage. State was reset.",
    };
    removeStoredState();
    render();
  }
}

function setState(patch, options = { render: true }) {
  const previousHash = hashFromState();
  state = { ...state, ...patch };
  saveNow();
  const nextHash = hashFromState();
  if (nextHash !== previousHash) {
    window.history.pushState(null, "", nextHash);
  }
  if (options.render) render();
}

function resetState() {
  removeStoredState();
  state = structuredClone(seedState);
  render();
}

function selectedUser() {
  return users.find((user) => user.id === state.userId) ?? users[0];
}

function selectedTask() {
  return state.tasks.find((item) => item.id === state.selectedTaskId) ?? firstVisibleTask() ?? state.tasks[0];
}

function firstVisibleTask() {
  return visibleTasks()[0];
}

function visibleTasks() {
  const user = selectedUser();
  return state.tasks.filter((taskItem) => canSeeTask(taskItem, user));
}

function canSeeTask(taskItem, user) {
  if (taskItem.lifecycle === lifecycle.draft || taskItem.lifecycle === lifecycle.funded) {
    return taskItem.requester === user.id || user.role === "Organization reviewer";
  }
  if (!state.includeReserved && taskItem.availability === availability.reserved) {
    return taskItem.requester === user.id || taskItem.assignee === user.id || user.role === "Organization reviewer";
  }
  if (taskItem.visibility === "organization") {
    return user.role !== "Implementor" || taskItem.assignee === user.id || taskItem.requester === user.id;
  }
  return true;
}

function tasksForRequester() {
  return state.tasks.filter((taskItem) => taskItem.requester === state.userId || selectedUser().role === "Organization reviewer");
}

function reviewableTasks() {
  const user = selectedUser();
  return state.tasks.filter((taskItem) =>
    canReviewTask(taskItem, user) && (
      taskItem.submissions.length > 0 ||
      taskItem.reservations.some((reservation) => reservation.state === "requested" || reservation.state === "active")
    )
  );
}

function canReviewTask(taskItem, user = selectedUser()) {
  if (taskItem.requester === user.id) return true;
  // An organization reviewer can settle organization-visibility work, but never
  // public tasks they did not request.
  return user.role === "Organization reviewer" && taskItem.visibility === "organization";
}

function updateTask(taskId, change, activity) {
  state = {
    ...state,
    tasks: state.tasks.map((item) => item.id === taskId ? appendTaskEvent({ ...item, ...change(item) }, activity) : item),
    activityLog: activity ? [activity, ...state.activityLog].slice(0, maxActivity) : state.activityLog,
  };
  saveNow();
  render();
}

function appendTaskEvent(taskItem, activity) {
  if (!activity) return taskItem;
  return { ...taskItem, timeline: [activity, ...taskItem.timeline].slice(0, 5) };
}

function enumValuesOf(field) {
  return (field.enum || "").split(",").map((value) => value.trim()).filter(Boolean);
}

function schemaForField(field) {
  const values = enumValuesOf(field);
  let spec;
  if (field.type === "integer") spec = { kind: "integer" };
  else if (field.type === "decimal") spec = { kind: "decimal_string" };
  else if (field.type === "string_list") spec = { kind: "array", items: values.length ? { kind: "string", enum: values } : { kind: "string" } };
  else spec = values.length ? { kind: "string", enum: values } : { kind: "string" };
  if (field.required) spec.required = true;
  return spec;
}

function draftSchema() {
  if (state.draftResponseKind !== "structured") return '{"kind":"freeform"}';
  const fields = state.draftFields.filter((field) => field.name.trim() !== "");
  if (fields.length === 0) return '{"kind":"freeform"}';
  const built = { kind: "object", fields: {} };
  fields.forEach((field) => {
    built.fields[field.name.trim()] = schemaForField(field);
  });
  return JSON.stringify(built);
}

function friendlyType(spec) {
  if (!spec || !spec.kind) return "value";
  if (spec.kind === "string") return spec.enum ? `one of ${spec.enum.join(", ")}` : "text";
  if (spec.kind === "integer") return "whole number";
  if (spec.kind === "decimal_string") return "decimal (a number sent as text)";
  if (spec.kind === "array") return `list of ${friendlyType(spec.items)}`;
  if (spec.kind === "object") return "object";
  return spec.kind;
}

function schemaSummary(schemaJson) {
  let parsed;
  try {
    parsed = JSON.parse(schemaJson);
  } catch {
    return "";
  }
  if (!parsed || parsed.kind === "freeform") return "Free-form text — no required structure.";
  if (parsed.kind === "object" && parsed.fields) {
    const fields = Object.entries(parsed.fields).map(([name, spec]) => `${name}: ${friendlyType(spec)}${spec.required ? " (required)" : ""}`);
    return fields.length ? fields.join(" · ") : "Free-form text — no required structure.";
  }
  return "";
}

function validateResponse(schemaJson, text) {
  let schema;
  try {
    schema = JSON.parse(schemaJson);
  } catch {
    return { ok: true, errors: [] };
  }
  if (!schema || schema.kind === "freeform" || !schema.fields) return { ok: true, errors: [] };
  let value;
  try {
    value = JSON.parse(text);
  } catch {
    return { ok: false, errors: ["Response is not valid JSON."] };
  }
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return { ok: false, errors: ["Response must be a JSON object."] };
  }
  const errors = [];
  for (const [name, spec] of Object.entries(schema.fields)) {
    if (!Object.prototype.hasOwnProperty.call(value, name)) {
      if (spec.required) errors.push(`Missing required field "${name}".`);
      continue;
    }
    const fieldValue = value[name];
    if (spec.kind === "integer" && !Number.isInteger(fieldValue)) {
      errors.push(`"${name}" must be a whole number.`);
    } else if (spec.kind === "decimal_string" && typeof fieldValue !== "string") {
      errors.push(`"${name}" must be a decimal sent as text.`);
    } else if (spec.kind === "string") {
      if (typeof fieldValue !== "string") errors.push(`"${name}" must be text.`);
      else if (spec.enum && !spec.enum.includes(fieldValue)) errors.push(`"${name}" must be one of ${spec.enum.join(", ")}.`);
    } else if (spec.kind === "array") {
      if (!Array.isArray(fieldValue)) errors.push(`"${name}" must be a list.`);
      else if (spec.items?.enum) {
        const invalid = fieldValue.filter((item) => !spec.items.enum.includes(item));
        if (invalid.length) errors.push(`"${name}" has values outside ${spec.items.enum.join(", ")}.`);
      }
    }
  }
  return { ok: errors.length === 0, errors };
}

function schemaDesigner() {
  const kindSelect = `
    <label for="draft-response-kind">Response from worker
      <select id="draft-response-kind" data-field="draftResponseKind">
        ${option("freeform", "Free-form text", state.draftResponseKind)}
        ${option("structured", "Structured fields", state.draftResponseKind)}
      </select>
    </label>`;
  if (state.draftResponseKind !== "structured") {
    return `
      <div class="schema-designer wide-field">
        ${kindSelect}
        <p class="schema-hint">Workers reply in free-form text. No response schema is enforced.</p>
      </div>`;
  }
  const allowsEnum = (type) => type === "string" || type === "string_list" || type === undefined;
  const rows = state.draftFields
    .map((field, index) => `
      <div class="schema-field-row">
        <input class="schema-field-name" data-schema-name="${index}" value="${escapeAttribute(field.name)}" placeholder="field name" aria-label="Field name">
        <select class="schema-field-type" data-schema-type="${index}" aria-label="Field type">
          ${option("string", "Text", field.type)}
          ${option("integer", "Whole number", field.type)}
          ${option("decimal", "Decimal", field.type)}
          ${option("string_list", "List of text", field.type)}
        </select>
        <label class="schema-field-required"><input type="checkbox" data-schema-required="${index}" ${field.required ? "checked" : ""}> required</label>
        <button class="button ghost" type="button" data-action="remove-field" data-index="${index}" aria-label="Remove field">Remove</button>
        ${allowsEnum(field.type) ? `<input class="schema-field-enum" data-schema-enum="${index}" value="${escapeAttribute(field.enum || "")}" placeholder="allowed values (comma-separated, optional)" aria-label="Allowed values">` : ""}
      </div>`)
    .join("");
  const names = state.draftFields.map((field) => field.name.trim()).filter(Boolean);
  const duplicates = names.filter((name, index) => names.indexOf(name) !== index);
  const hasEmpty = state.draftFields.some((field) => field.name.trim() === "");
  const warnings = [
    duplicates.length ? `Duplicate field name "${duplicates[0]}" — the last one wins.` : "",
    hasEmpty ? "Fields with no name are omitted from the schema." : "",
  ].filter(Boolean);
  const warning = warnings.length ? `<p class="schema-warning">${escapeHtml(warnings.join(" "))}</p>` : "";
  return `
    <div class="schema-designer wide-field">
      ${kindSelect}
      <p class="schema-hint">Design the structured result you want back. Mark fields required and, for text fields, list the allowed values workers must choose from.</p>
      <div class="schema-fields">${rows}</div>
      ${warning}
      <button class="button secondary" type="button" data-action="add-field">Add field</button>
      <div class="schema-block">
        <span>What workers must return</span>
        <p class="schema-friendly">${escapeHtml(schemaSummary(draftSchema()))}</p>
        <pre>${escapeHtml(draftSchema())}</pre>
      </div>
    </div>`;
}

function addDraftField() {
  setState({ draftFields: [...state.draftFields, { name: "", type: "string", required: false, enum: "" }] });
}

function removeDraftField(index) {
  setState({ draftFields: state.draftFields.filter((_, position) => position !== index) });
}

function updateDraftField(index, key, value) {
  const fields = state.draftFields.map((field, position) => (position === index ? { ...field, [key]: value } : field));
  setState({ draftFields: fields });
}

function createDraftTask() {
  const id = `demo-${state.localTaskSeq}`;
  const newTask = task({
    id,
    title: state.draftTitle || "Untitled task",
    requester: state.userId,
    assignee: "",
    area: "Custom Board",
    visibility: state.draftVisibility,
    policy: state.draftPolicy,
    reward: draftReward(),
    lifecycle: lifecycle.draft,
    availability: availability.available,
    reservationHours: state.draftReservationHours,
    schema: draftSchema(),
    objective: state.draftDescription || "Describe the requested result.",
    timeline: [`${selectedUser().name} drafted the task.`],
  });
  state = {
    ...state,
    localTaskSeq: state.localTaskSeq + 1,
    selectedTaskId: id,
    page: "requester",
    tasks: [
      newTask,
      ...state.tasks.filter((item) => !item.id.startsWith("demo-")),
      ...state.tasks.filter((item) => item.id.startsWith("demo-")).slice(0, maxLocalTasks - 1),
    ],
    activityLog: [`${selectedUser().name} drafted ${newTask.title}.`, ...state.activityLog].slice(0, maxActivity),
  };
  saveNow();
  render();
}

function draftReward() {
  const credits = Number.parseInt(state.draftCredits, 10) || 0;
  const collectibles = state.draftCollectible.trim() === "" ? [] : [state.draftCollectible.trim()];
  if (state.draftRewardKind === "none") return rewardNone();
  if (state.draftRewardKind === "credits") return rewardCredits(credits);
  if (state.draftRewardKind === "collectible") return rewardCollectible(collectibles);
  return rewardBundle(credits, collectibles);
}

function render() {
  document.body.dataset.mode = state.mode;
  document.body.dataset.theme = state.theme;
  document.getElementById("app").innerHTML = `
    <main class="app-shell">
      ${topbar()}
      ${pageNavigation()}
      <section class="page-shell">${pageView()}</section>
    </main>
  `;
}

function topbar() {
  const user = selectedUser();
  return `
    <nav class="topbar">
      <a class="brand" href="../">Sharecrop</a>
      <div class="topbar-actions">
        <button class="button secondary" data-action="reset" data-testid="topbar-reset">Reset demo</button>
        <a class="button secondary" href="../docs/">Docs</a>
        <div class="account-menu">
          <button class="account-button" data-action="toggleLogin" aria-expanded="${state.loginOpen ? "true" : "false"}">
            <span>Viewing as — switch role ▾</span>
            <strong>${escapeHtml(user.name)} · ${escapeHtml(user.role)}</strong>
          </button>
          ${state.loginOpen ? loginPanel() : ""}
        </div>
      </div>
    </nav>
  `;
}

function loginPanel() {
  return `
    <section class="login-popover">
      <span class="eyebrow">Switch role</span>
      <p class="role-switch-hint">This demo lets you role-play both sides. Switch between a requester, a worker, an agent operator, and an org reviewer to see how the same task looks to each.</p>
      <label for="demo-user">Select persona</label>
      <select id="demo-user" data-field="userId">
        ${users.map((item) => `<option value="${escapeAttribute(item.id)}" ${item.id === state.userId ? "selected" : ""}>${escapeHtml(item.name)} - ${escapeHtml(item.role)}</option>`).join("")}
      </select>
      <div class="persona-list">
        ${users.map((item) => personaBadge(item)).join("")}
      </div>
      <div class="provider-grid">
        ${providers.map((provider) => `<button class="button secondary compact-button" data-provider="${escapeAttribute(provider)}">${escapeHtml(provider)}</button>`).join("")}
      </div>
      <p class="hint">${state.socialProvider ? `${escapeHtml(state.socialProvider)} sign-in is a placeholder.` : "Provider buttons are non-functional demo controls."}</p>
    </section>
  `;
}

function personaBadge(user) {
  return `
    <button class="persona-badge ${user.id === state.userId ? "selected" : ""}" data-action="choosePersona" data-user="${escapeAttribute(user.id)}" aria-pressed="${user.id === state.userId ? "true" : "false"}">
      <strong>${escapeHtml(user.name)}</strong>
      <span>${escapeHtml(user.role)}</span>
    </button>
  `;
}

function pageNavigation() {
  const topLevelPages = pages.filter((page) => !page.detailOnly);
  return `
    <nav class="page-tabs" aria-label="Demo pages">
      ${topLevelPages.map((page) => `<button class="tab ${state.page === page.id ? "active" : ""}" data-page="${escapeAttribute(page.id)}" ${state.page === page.id ? 'aria-current="page"' : ""}>${escapeHtml(page.label)}</button>`).join("")}
    </nav>
  `;
}

function pageView() {
  if (state.page === "discover") return discoverPage();
  if (state.page === "task") return taskDetailPage();
  if (state.page === "user") return userPage();
  if (state.page === "requester") return requesterPage();
  if (state.page === "review") return reviewPage();
  if (state.page === "integrations") return integrationsPage();
  if (state.page === "settings") return settingsPage();
  return overviewPage();
}

function userPage() {
  const user = users.find((item) => item.id === state.selectedUserId) ?? users[0];
  const requested = state.tasks.filter((item) => item.requester === user.id);
  const assigned = state.tasks.filter((item) => item.assignee === user.id);
  const taskLine = (item) =>
    `<li><a class="link-button" data-open-task="${escapeAttribute(item.id)}" href="${taskHref(item.id)}">${escapeHtml(item.title)}</a> <span class="muted">${escapeHtml(availabilityLabel(item.availability))}</span></li>`;
  return `
    <section class="panel briefing-panel">
      <div>
        <span class="eyebrow">Profile</span>
        <h2>${escapeHtml(user.name)}</h2>
        <div class="badge-row">
          <span>${escapeHtml(user.role)}</span>
          <span>${escapeHtml(user.balance)} credits</span>
        </div>
        <h3>Requested tasks</h3>
        ${requested.length ? `<ul class="objective-list">${requested.map(taskLine).join("")}</ul>` : `<p class="muted">No requested tasks.</p>`}
        <h3>Assigned tasks</h3>
        ${assigned.length ? `<ul class="objective-list">${assigned.map(taskLine).join("")}</ul>` : `<p class="muted">No assigned tasks.</p>`}
        <button class="button secondary" data-page="discover">Back to tasks</button>
      </div>
    </section>
  `;
}

function dashboardSpotlight(taskItem) {
  if (!taskItem) return "";
  return `
    <section class="panel spotlight-panel">
      <span class="eyebrow">Continue where you left off</span>
      <h2>${escapeHtml(taskItem.title)}</h2>
      <p class="spotlight-objective">${escapeHtml(taskItem.objective)}</p>
      <div class="badge-row">
        ${toneSpan(lifecycleLabel(taskItem.lifecycle), lifecycleTone(taskItem.lifecycle))}
        ${toneSpan(availabilityLabel(taskItem.availability), availabilityTone(taskItem.availability))}
        <span>${escapeHtml(rewardLabel(taskItem.reward))}</span>
      </div>
      <a class="button primary" data-open-task="${escapeAttribute(taskItem.id)}" href="${taskHref(taskItem.id)}">Open task</a>
    </section>`;
}

function overviewPage() {
  const user = selectedUser();
  const taskItem = selectedTask();
  return `
    <section class="command-grid">
      <div class="hero-panel command-hero">
        <div>
          <span class="eyebrow">${escapeHtml(user.role)}</span>
          <h1>${escapeHtml(headlineFor(user.role))}</h1>
          <p class="product-line">${escapeHtml(productTagline)}</p>
          <p>${escapeHtml(copyFor(user.role))}</p>
          <div class="row-actions">
            ${primaryFlowButton(user.role)}
            <button class="button secondary" data-page="settings">Tune demo</button>
          </div>
        </div>
        <div class="summary-grid">
          ${metricCard("Open tasks", String(state.tasks.filter((item) => item.lifecycle === lifecycle.open).length))}
          ${metricCard("Credits available", `${balanceOf(user.id)}`)}
          ${metricCard("Held in escrow", `${escrowHeldBy(user.id)}`)}
          ${metricCard("Collectibles", String(inventoryOf(user.id).length))}
        </div>
      </div>
      ${dashboardSpotlight(taskItem)}
    </section>
    <section class="panel">
      <span class="eyebrow">Task paths</span>
      <div class="flow-grid">
        ${flowCard("discover", "Tasks", "Scan visible work, compare status, and open a task page.")}
        ${flowCard("requester", "Post Task", "Draft, fund, open, cancel, and inspect requester-owned tasks.")}
        ${flowCard("review", "Reviews", "Approve reservations and settle submissions with notes and payouts.")}
        ${flowCard("integrations", "Agent/API", "Read the REST and MCP instructions for the selected task.")}
      </div>
    </section>
    ${activityFeed()}
  `;
}

function discoverPage() {
  const tasks = filteredBoardTasks();
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Task list</span>
          <h1>${escapeHtml(boardTitleFor(selectedUser().role))}</h1>
          <p>Scan visible work by status, reward, and next action. Open a task page for the full briefing, submission form, and API/MCP instructions.</p>
        </div>
        <div class="board-controls">
          <label class="check-row">
            <input type="checkbox" data-field="includeReserved" ${state.includeReserved ? "checked" : ""}>
            Include reserved
          </label>
          <select data-field="boardFilter" aria-label="Task filter">
            ${filterOption("all", "All sectors")}
            ${filterOption("public", "Public")}
            ${filterOption("organization", "Organization")}
            ${filterOption("mine", "Mine")}
          </select>
        </div>
      </div>
      ${taskList(tasks, "discover")}
    </section>
    ${activityFeed()}
  `;
}

function taskDetailPage() {
  const taskItem = selectedTask();
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Task page</span>
          <h1>${escapeHtml(taskItem.title)}</h1>
          <p>Use this page for the briefing, task-specific actions, response schema, and handoff instructions.</p>
        </div>
        <div class="row-actions">
          <button class="button secondary" data-page="discover">Back to tasks</button>
          <button class="button secondary" data-page="integrations">API / MCP</button>
        </div>
      </div>
    </section>
    ${missionBriefing(taskItem)}
    ${activityFeed()}
  `;
}

function requesterPage() {
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Post task</span>
          <h1>Create, fund, and open a task</h1>
          <p>Create a local task, choose the reward, set visibility and participation policy, then open it onto the board.</p>
        </div>
      </div>
      <form class="create-form" data-form="create-task">
        <label for="draft-title">Task title<input id="draft-title" data-field="draftTitle" value="${escapeAttribute(state.draftTitle)}"></label>
        <label class="wide-field" for="draft-description">Instructions (free-form)<textarea id="draft-description" data-field="draftDescription">${escapeHtml(state.draftDescription)}</textarea></label>
        ${schemaDesigner()}
        <label for="draft-reward-kind">Reward kind
          <select id="draft-reward-kind" data-field="draftRewardKind">
            ${option("none", "No reward", state.draftRewardKind)}
            ${option("credits", "Credits", state.draftRewardKind)}
            ${option("collectible", "Collectible", state.draftRewardKind)}
            ${option("bundle", "Credits + collectible", state.draftRewardKind)}
          </select>
        </label>
        <label for="draft-credits">Credits<input id="draft-credits" data-field="draftCredits" value="${escapeAttribute(state.draftCredits)}"><span class="schema-hint">Credits are this demo's currency (think 1 credit ≈ $1), held from your balance until you accept a result.</span></label>
        <label for="draft-collectible">Collectible<input id="draft-collectible" data-field="draftCollectible" value="${escapeAttribute(state.draftCollectible)}"></label>
        <label for="draft-visibility">Visibility
          <select id="draft-visibility" data-field="draftVisibility">
            ${option("public", "Public marketplace", state.draftVisibility)}
            ${option("organization", "Organization", state.draftVisibility)}
          </select>
        </label>
        <label for="draft-policy">Participation
          <select id="draft-policy" data-field="draftPolicy">
            ${option(policy.open, "Open submissions", state.draftPolicy)}
            ${option(policy.reservation, "Reservation required", state.draftPolicy)}
            ${option(policy.approval, "Requester approval required", state.draftPolicy)}
          </select>
        </label>
        ${state.draftPolicy === policy.open ? "" : `<label for="draft-reservation-hours">Reservation expiry (hours)<input id="draft-reservation-hours" data-field="draftReservationHours" value="${escapeAttribute(state.draftReservationHours)}"></label>`}
        <button class="button primary" data-action="create" type="button">Create draft task</button>
      </form>
    </section>
    <section class="panel">
      <span class="eyebrow">Requester task list</span>
      ${taskList(tasksForRequester(), "requester")}
    </section>
  `;
}

function reviewPage() {
  const tasks = reviewableTasks();
  const taskItem = activeTaskFor(tasks);
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Dispatch review</span>
          <h1>Review reservations and submitted responses</h1>
          <p>Approve access, request changes, reject with a fair partial payout, or accept and pay out the reward.</p>
        </div>
        ${taskItem ? taskSelect(taskItem, tasks) : ""}
      </div>
      ${taskItem ? `<div class="review-layout">
        <section class="sub-panel control-room">
          <h2>Reservation queue</h2>
          ${reservationQueue(taskItem)}
        </section>
        <section class="sub-panel control-room">
          <h2>Submission decisions</h2>
          ${submissionList(taskItem)}
        </section>
      </div>` : `<p class="empty-state">No reservations or submitted responses need this persona's review.</p>`}
    </section>
    ${activityFeed()}
  `;
}

function integrationsPage() {
  const taskItem = selectedTask();
  const host = "https://sharecrop.example";
  const compactSample = (() => {
    try {
      return JSON.stringify(JSON.parse(schemaTemplate(taskItem.schema) || "{}"));
    } catch {
      return "{}";
    }
  })();
  const restBody = `{"response_json": ${JSON.stringify(compactSample)}}`;
  const claimStep = taskItem.policy === policy.reservation
    ? "sharecrop.reserve_task\n"
    : taskItem.policy === policy.approval
    ? "sharecrop.request_approval\n"
    : "";
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Agent/API console</span>
          <h1>Provide a result for ${escapeHtml(taskItem.title)}</h1>
          <p>Sharecrop is a reverse MCP: a person or their agent connects, reads the task's schema, and returns a structured result. Every task carries ready-to-run REST and MCP steps.</p>
        </div>
        ${taskSelect(taskItem, state.tasks)}
      </div>
      <div class="code-grid">
        <article class="sub-panel uplink-panel">
          <span class="eyebrow">Agent credential</span>
          <h2>Scoped client setup</h2>
          <pre>{
  "mcpServers": {
    "sharecrop": {
      "url": "${host}/mcp",
      "headers": { "Authorization": "Bearer &lt;AGENT_TOKEN&gt;" }
    }
  }
}</pre>
          <button class="button primary" data-action="agentRun">Run as Sol agent</button>
          <p class="hint">Demo scopes: tasks_read, submissions_write. The button adds a schema-shaped, Sol-labeled MCP submission to this task.</p>
        </article>
        <article class="sub-panel uplink-panel">
          <span class="eyebrow">Worker REST</span>
          <h2>Submit result</h2>
          <pre>curl -X POST ${host}/api/tasks/${escapeHtml(taskItem.id)}/submissions \\
  -H "Authorization: Bearer &lt;AGENT_TOKEN&gt;" \\
  -H "Content-Type: application/json" \\
  -d '${escapeHtml(restBody)}'</pre>
        </article>
        <article class="sub-panel uplink-panel">
          <span class="eyebrow">Worker MCP</span>
          <h2>Session workflow</h2>
          <pre>initialize -> Mcp-Session-Id
${escapeHtml(claimStep)}sharecrop.get_task_schema
sharecrop.submit_response</pre>
        </article>
        <article class="sub-panel uplink-panel">
          <span class="eyebrow">Requester MCP</span>
          <h2>Review tools</h2>
          <pre>sharecrop.list_task_reservations
sharecrop.approve_task_reservation
sharecrop.request_changes
sharecrop.accept_submission
sharecrop.reject_submission</pre>
        </article>
      </div>
    </section>
  `;
}

function settingsPage() {
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Demo settings</span>
          <h1>Theme, local state, and test controls</h1>
          <p>Use the controls here to change presentation, reset the local task state, or keep demo-only tools out of the main workflows.</p>
        </div>
      </div>
      <div class="settings-grid">
        <section class="sub-panel">
          <h2>Theme</h2>
          <div class="segmented" aria-label="Mode">
            ${modeButton("light", "Light")}
            ${modeButton("dark", "Dark")}
          </div>
          <div class="theme-grid">
            ${themeButton("corporate", "Corporate")}
            ${themeButton("rustic", "Rustic")}
            ${themeButton("blocky", "Blocky")}
            ${themeButton("showcase", "Showcase")}
          </div>
        </section>
        <section class="sub-panel">
          <h2>Local state</h2>
          <p>The demo stores persona, task progress, activity, balances, and local draft tasks only in this browser.</p>
          ${state.storageWarning ? `<p class="danger-text">${escapeHtml(state.storageWarning)}</p>` : ""}
          <button class="button primary" data-action="reset">Clear state</button>
        </section>
      </div>
    </section>
  `;
}

function taskList(tasks, context) {
  if (tasks.length === 0) return `<p class="empty-state">No tasks match these filters.</p>`;
  return `
    <div class="task-list" data-context="${escapeAttribute(context)}">
      ${tasks.map((item) => taskListRow(item)).join("")}
    </div>
  `;
}

function taskListRow(taskItem) {
  const action = nextAction(taskItem);
  return `
    <article class="task-list-row">
      <div class="task-list-main">
        <div class="task-title-row">
          <a class="link-button task-title-link stretched-link" data-open-task="${escapeAttribute(taskItem.id)}" href="${taskHref(taskItem.id)}">${escapeHtml(taskItem.title)}</a>
        </div>
        <p>${escapeHtml(taskItem.objective)}</p>
        <div class="mission-meta">
          <span>${escapeHtml(areaLabel(taskItem.area))}</span>
          <span>${escapeHtml(policyLabel(taskItem.policy))}</span>
          <span>${taskItem.assignee ? `Assigned: ${userLink(taskItem.assignee)}` : `Requester: ${userLink(taskItem.requester)}`}</span>
        </div>
      </div>
      <div class="task-widget">
        ${toneSpan(availabilityLabel(taskItem.availability), availabilityTone(taskItem.availability))}
        <div class="reward-row">${rewardChips(taskItem.reward)}</div>
      </div>
      <div class="task-actions">
        ${taskListAction(action, taskItem)}
      </div>
    </article>
  `;
}

function taskListAction(action, taskItem) {
  if (action.kind === "openReviewQueue") return `<button class="button primary compact-button" data-task-action="openReviewQueue" data-task-id="${escapeAttribute(taskItem.id)}">Review queue</button>`;
  if (action.kind === "reserve") return `<button class="button primary compact-button" data-task-action="reserve" data-task-id="${escapeAttribute(taskItem.id)}">Reserve</button>`;
  if (action.kind === "requestApproval") return `<button class="button primary compact-button" data-task-action="requestApproval" data-task-id="${escapeAttribute(taskItem.id)}">Request approval</button>`;
  if (action.kind === "fund") return `<button class="button primary compact-button" data-task-action="fund" data-task-id="${escapeAttribute(taskItem.id)}">Fund</button>`;
  if (action.kind === "openMission") return `<button class="button primary compact-button" data-task-action="openMission" data-task-id="${escapeAttribute(taskItem.id)}">Open task</button>`;
  if (action.kind === "submit") return `<a class="button primary compact-button" data-open-task="${escapeAttribute(taskItem.id)}" href="${taskHref(taskItem.id)}">Open to submit</a>`;
  if (action.kind === "agentRun") return `<a class="button primary compact-button" data-open-task="${escapeAttribute(taskItem.id)}" href="${taskHref(taskItem.id)}">Open to run agent</a>`;
  const reserved = taskItem.availability === availability.reserved ||
    taskItem.reservations.some((reservation) => reservation.state === "active" || reservation.state === "requested");
  if (reserved) {
    const who = taskItem.assignee ? ` by ${userName(taskItem.assignee)}` : "";
    return `<span class="reserved-pill" title="Already reserved${escapeAttribute(who)}">Reserved</span>`;
  }
  return "";
}

function availabilityTone(value) {
  if (value === availability.accepted) return "success";
  if (value === availability.submitted || value === availability.reserved || value === availability.awaitingApproval || value === availability.changesRequested) return "warning";
  if (value === availability.rejected) return "danger";
  return "";
}

function lifecycleTone(value) {
  if (value === lifecycle.open) return "success";
  if (value === lifecycle.draft || value === lifecycle.funded) return "warning";
  if (value === lifecycle.canceled) return "danger";
  return "";
}

function toneSpan(label, tone) {
  return `<span class="${tone ? `badge--${tone}` : ""}">${escapeHtml(label)}</span>`;
}

function renderInputBlock(block) {
  const label = block.label ? `<p class="input-label">${escapeHtml(block.label)}</p>` : "";
  if (block.kind === "list") {
    return `${label}<ul class="input-list">${block.items.map((item) => `<li>${escapeHtml(item)}</li>`).join("")}</ul>`;
  }
  if (block.kind === "records") {
    const head = `<tr>${block.columns.map((column) => `<th>${escapeHtml(column)}</th>`).join("")}</tr>`;
    const body = block.rows
      .map((row) => `<tr>${row.map((cell) => `<td>${cell === "" ? "—" : escapeHtml(cell)}</td>`).join("")}</tr>`)
      .join("");
    return `${label}<div class="input-table-wrap"><table class="input-table"><thead>${head}</thead><tbody>${body}</tbody></table></div>`;
  }
  if (block.kind === "code") {
    return `${label}<pre class="input-code">${escapeHtml(block.body)}</pre>`;
  }
  return `${label}<p class="input-text">${escapeHtml(block.body)}</p>`;
}

function renderTaskInputs(inputs) {
  if (!Array.isArray(inputs) || inputs.length === 0) return "";
  return `
    <div class="input-block">
      <span class="eyebrow">Input / materials</span>
      ${inputs.map(renderInputBlock).join("")}
    </div>
  `;
}

function missionBriefing(taskItem) {
  if (!taskItem) return "";
  const action = nextAction(taskItem);
  return `
    <section class="panel briefing-panel">
      <div>
        <span class="eyebrow">Task briefing</span>
        <h2>${escapeHtml(taskItem.title)}</h2>
        <p>${escapeHtml(taskItem.objective)}</p>
        ${renderTaskInputs(taskItem.inputs)}
        <div class="badge-row">
          ${toneSpan(lifecycleLabel(taskItem.lifecycle), lifecycleTone(taskItem.lifecycle))}
          <span>${escapeHtml(visibilityLabel(taskItem.visibility))}</span>
          <span>${escapeHtml(policyLabel(taskItem.policy))}</span>
          ${toneSpan(availabilityLabel(taskItem.availability), availabilityTone(taskItem.availability))}
        </div>
        <ul class="objective-list">
          <li>Requester: ${userLink(taskItem.requester)}</li>
          <li>Assignee: ${userLink(taskItem.assignee)}</li>
          <li>Reward: ${escapeHtml(rewardLabel(taskItem.reward))}</li>
        </ul>
        <div class="schema-block">
          <span>What to return</span>
          <p class="schema-friendly">${escapeHtml(schemaSummary(taskItem.schema))}</p>
          <pre>${escapeHtml(taskItem.schema)}</pre>
        </div>
      </div>
      <div class="sub-panel action-console">
        <h2>${escapeHtml(action.title)}</h2>
        ${workerReviewNotice(taskItem)}
        ${actionPayload(action, taskItem)}
        ${actionControls(action)}
      </div>
      <div class="sub-panel activity-feed">
        <h2>Task log</h2>
        ${taskItem.timeline.slice(0, 5).map((item) => `<p>${escapeHtml(item)}</p>`).join("")}
      </div>
    </section>
  `;
}

function nextAction(taskItem) {
  const user = selectedUser();
  const reservation = taskItem.reservations.find((item) => item.by === state.userId);
  const submission = taskItem.submissions.find((item) => item.by === state.userId);

  if (taskItem.requester === state.userId && taskItem.lifecycle === lifecycle.draft) {
    return { kind: "fund", title: "Prepare reward", explanation: "Fund or attach the reward before opening the task." };
  }
  if (taskItem.requester === state.userId && taskItem.lifecycle === lifecycle.funded) {
    return { kind: "openMission", title: "Open task", explanation: "Publish the funded task to the list." };
  }
  if (taskItem.requester === state.userId && taskItem.lifecycle === lifecycle.open) {
    if (hasReviewWork(taskItem)) {
      return { kind: "openReviewQueue", title: "Requester watch", explanation: "Open Reviews to decide reservations and submissions." };
    }
    return { kind: "none", title: "Watching", explanation: "This task is open. New reservations or submissions will appear in Reviews." };
  }
  if (taskItem.lifecycle !== lifecycle.open || taskItem.availability === availability.closed || taskItem.availability === availability.accepted) {
    return { kind: "none", title: "No action", explanation: "This task is not open for new work." };
  }
  if (taskItem.banned?.includes(state.userId)) {
    return { kind: "none", title: "Blocked", explanation: "This implementor is banned from this task." };
  }
  const reservedByOther = (taskItem.assignee && taskItem.assignee !== state.userId) ||
    taskItem.reservations.some((item) => item.state === "active" && item.by !== state.userId);
  if (reservedByOther) {
    return { kind: "none", title: "Reserved", explanation: `This task is reserved by ${userName(taskItem.assignee || "another worker")}.` };
  }
  if (submission?.state === "submitted") {
    return { kind: "none", title: "Response received", explanation: "The requester has a submitted result to review." };
  }
  if (submission?.state === "changes_requested") {
    return { kind: "submit", title: "Revise response", explanation: "Changes were requested. Submit an updated response for the same task." };
  }
  if (taskItem.policy === policy.approval && reservation?.state === "requested") {
    return { kind: "none", title: "Awaiting clearance", explanation: "Requester approval is pending." };
  }
  if (taskItem.policy === policy.approval && taskItem.assignee !== state.userId) {
    return { kind: "requestApproval", title: "Request clearance", explanation: "This task requires requester approval before work starts." };
  }
  if (taskItem.policy === policy.reservation && taskItem.assignee !== state.userId) {
    return { kind: "reserve", title: "Claim reservation", explanation: "Reserve the task to make it exclusive for your run." };
  }
  if (user.role === "Agent operator") {
    return { kind: "agentRun", title: "Run agent", explanation: "Simulate a scoped local agent submitting over MCP." };
  }
  return { kind: "submit", title: "Submit response", explanation: "You are eligible to submit a result for this task." };
}

function actionButtons(action) {
  if (action.kind === "fund") return `<button class="button primary" data-action="fund">Fund reward</button><button class="button secondary" data-action="cancelMission">Cancel</button>`;
  if (action.kind === "openMission") return `<button class="button primary" data-action="openMission">Open task</button><button class="button secondary" data-action="cancelMission">Cancel</button>`;
  if (action.kind === "reserve") return `<button class="button primary" data-action="reserve">Reserve task</button>`;
  if (action.kind === "requestApproval") return `<button class="button primary" data-action="requestApproval">Request approval</button>`;
  if (action.kind === "submit") return `<button class="button primary" data-action="submit">Submit response</button>`;
  if (action.kind === "agentRun") return `<button class="button primary" data-action="agentRun">Simulate agent run</button>`;
  if (action.kind === "openReviewQueue") return `<button class="button primary" data-page="review">Review queue</button>`;
  return "";
}

function actionControls(action) {
  const buttons = actionButtons(action);
  if (buttons !== "") {
    return `<div class="row-actions">${buttons}</div><p class="hint">${escapeHtml(action.explanation)}</p>`;
  }
  return `<div class="status-panel"><strong>${escapeHtml(action.title)}</strong><span>${escapeHtml(action.explanation)}</span></div>`;
}

function sampleForSpec(spec) {
  if (!spec) return "";
  if (spec.kind === "array") return [];
  if (spec.kind === "integer") return 0;
  return "";
}

function schemaTemplate(schemaJson) {
  let parsed;
  try {
    parsed = JSON.parse(schemaJson);
  } catch {
    return "";
  }
  if (!parsed || parsed.kind === "freeform" || !parsed.fields) return "";
  const skeleton = {};
  for (const [name, spec] of Object.entries(parsed.fields)) skeleton[name] = sampleForSpec(spec);
  return JSON.stringify(skeleton, null, 2);
}

function responseDraftFor(taskItem) {
  if (!taskItem) return "";
  const saved = state.responseDrafts?.[taskItem.id];
  return saved !== undefined ? saved : schemaTemplate(taskItem.schema);
}

function workerReviewNotice(taskItem) {
  const submission = taskItem.submissions.find((item) => item.by === state.userId);
  if (!submission || (submission.state !== "changes_requested" && submission.state !== "rejected")) return "";
  const note = submission.reviewNote
    ? `<p class="review-note-text">${escapeHtml(submission.reviewNote)}</p>`
    : `<p class="review-note-text">No note was left.</p>`;
  return `
    <div class="worker-review-notice">
      <span class="eyebrow">${submission.state === "rejected" ? "Rejected by requester" : "Changes requested"}</span>
      ${note}
      <p class="input-label">Your previous response:</p>
      <pre class="input-code">${escapeHtml(submission.response)}</pre>
    </div>`;
}

function actionPayload(action, taskItem) {
  if (action.kind === "submit" || action.kind === "agentRun") {
    const summary = schemaSummary(taskItem.schema);
    const hint = summary ? `<span class="schema-hint">Your response should match — ${escapeHtml(summary)}</span>` : "";
    const note = state.submitNote && state.submitNote.taskId === taskItem.id
      ? (state.submitNote.ok
        ? `<p class="validate-ok">Response matches the schema.</p>`
        : `<p class="validate-error">${escapeHtml(state.submitNote.errors.join(" "))}</p>`)
      : "";
    return `<label for="response-text">Your response${hint}<textarea id="response-text" data-response-task="${escapeAttribute(taskItem.id)}">${escapeHtml(responseDraftFor(taskItem))}</textarea></label>${note}`;
  }
  return "";
}

function reservationQueue(taskItem) {
  if (taskItem.reservations.length === 0) return `<p class="empty-state">No reservations for this task.</p>`;
  return taskItem.reservations.map((reservation) => `
    <div class="queue-row">
      <div>
        <strong>${userLink(reservation.by)}</strong>
        <span>${escapeHtml(reservation.state)} / expires ${escapeHtml(reservation.expires ?? "48h")}</span>
      </div>
      <div class="row-actions">
        ${reservation.state === "requested" ? `<button class="button secondary" data-action="declineReservation" data-user="${escapeAttribute(reservation.by)}">Decline</button><button class="button primary" data-action="approve" data-user="${escapeAttribute(reservation.by)}">Approve</button>` : ""}
        ${reservation.state === "active" ? `<button class="button secondary" data-action="releaseReservation" data-user="${escapeAttribute(reservation.by)}">Release</button>` : ""}
      </div>
    </div>
  `).join("");
}

function submissionList(taskItem) {
  if (taskItem.submissions.length === 0) return `<p class="empty-state">No submissions for this task.</p>`;
  return taskItem.submissions.map((submission) => {
    const draft = reviewDraft(submission);
    if (submission.state !== "submitted") return submissionOutcome(submission);
    return `
      <div class="submission-row decision-console">
        <div>
          <strong>${userLink(submission.by)}</strong>
          <span class="action-chip">${escapeHtml(submissionStateLabel(submission.state))}</span>
        </div>
        <p class="decision-criteria"><strong>Should return</strong> — ${escapeHtml(schemaSummary(taskItem.schema))}</p>
        <code>${escapeHtml(submission.response)}</code>
        <div class="payout-strip">
          <span>Funded reward</span>
          <strong>${escapeHtml(rewardLabel(taskItem.reward))}</strong>
        </div>
        <label>Review note<textarea data-field="reviewNote" data-submission="${escapeAttribute(submission.id)}" placeholder="Optional note to the worker">${escapeHtml(draft.reviewNote)}</textarea></label>
        <div class="mini-grid">
          <label>Settle (credits)<input data-field="partialPayout" data-submission="${escapeAttribute(submission.id)}" value="${escapeAttribute(draft.partialPayout)}" placeholder="${escapeAttribute(String(taskItem.reward.credits || 0))} — full reward"></label>
          <label>Tip<input data-field="tipAmount" data-submission="${escapeAttribute(submission.id)}" value="${escapeAttribute(draft.tipAmount)}" placeholder="0"></label>
        </div>
        <label class="check-row"><input type="checkbox" data-field="banImplementor" data-submission="${escapeAttribute(submission.id)}" ${draft.banImplementor ? "checked" : ""}> Ban implementor from this task</label>
        <div class="row-actions">
          <button class="button secondary" data-action="requestChanges" data-user="${escapeAttribute(submission.by)}" data-submission="${escapeAttribute(submission.id)}">Request changes</button>
          <button class="button secondary" data-action="reject" data-user="${escapeAttribute(submission.by)}" data-submission="${escapeAttribute(submission.id)}">Reject</button>
          <button class="button primary" data-action="accept" data-user="${escapeAttribute(submission.by)}" data-submission="${escapeAttribute(submission.id)}">Accept</button>
        </div>
      </div>
    `;
  }).join("");
}

function submissionOutcome(submission) {
  const paid = Number(submission.payoutCredits ?? submission.partialPayout ?? 0) + Number(submission.tip ?? 0);
  return `
    <div class="submission-row outcome-row">
      <div>
        <strong>${userLink(submission.by)}</strong>
        <span class="action-chip">${escapeHtml(submissionStateLabel(submission.state))}</span>
      </div>
      <code>${escapeHtml(submission.response)}</code>
      <div class="status-panel">
        <strong>${escapeHtml(outcomeTitle(submission.state))}</strong>
        <span>${escapeHtml(submission.reviewNote || outcomeCopy(submission.state, paid))}</span>
      </div>
    </div>
  `;
}

function submissionStateLabel(value) {
  return value.replaceAll("_", " ");
}

function outcomeTitle(value) {
  if (value === "changes_requested") return "Waiting for revised response";
  if (value === "accepted") return "Accepted and settled";
  if (value === "rejected") return "Rejected";
  return "Decision recorded";
}

function outcomeCopy(value, paid) {
  if (value === "changes_requested") return "The worker keeps this task and can submit an updated response.";
  if (value === "accepted") return `The requester settled this submission with ${paid} credits plus any collectible reward.`;
  if (value === "rejected") return `The requester closed this submission with ${paid} credits paid.`;
  return "This submission no longer has active decision controls.";
}

function reviewDraft(submission) {
  return {
    reviewNote: submission.reviewNote || "",
    partialPayout: submission.partialPayout || "",
    tipAmount: submission.tip || "",
    banImplementor: submission.ban || false,
    ...(state.reviewDrafts[submission.id] ?? {}),
  };
}

function linkifyActivity(text) {
  let html = escapeHtml(text);
  for (const user of users) {
    html = html.replaceAll(escapeHtml(user.name), userLink(user.id));
  }
  return html;
}

function activityFeed() {
  return `
    <section class="panel activity-feed">
      <span class="eyebrow">Comms feed</span>
      ${state.activityLog.map((entry) => `<p>${linkifyActivity(entry)}</p>`).join("")}
    </section>
  `;
}

function taskSelect(taskItem, tasks) {
  return `
    <select class="task-select" data-field="selectedTaskId" aria-label="Selected task">
      ${tasks.map((item) => `<option value="${escapeAttribute(item.id)}" ${item.id === taskItem.id ? "selected" : ""}>${escapeHtml(item.title)}</option>`).join("")}
    </select>
  `;
}

function modeButton(value, label) {
  return `<button class="button ${state.mode === value ? "primary" : "secondary"}" data-mode="${escapeAttribute(value)}" aria-pressed="${state.mode === value ? "true" : "false"}">${escapeHtml(label)}</button>`;
}

function themeButton(value, label) {
  return `<button class="theme-chip ${state.theme === value ? "selected" : ""}" data-theme="${escapeAttribute(value)}" aria-pressed="${state.theme === value ? "true" : "false"}">${escapeHtml(label)}</button>`;
}

function metricCard(label, value) {
  return `<div class="metric-card"><span>${escapeHtml(label)}</span><strong>${escapeHtml(value)}</strong></div>`;
}

function flowCard(page, title, copy) {
  return `
    <button class="flow-card" data-page="${escapeAttribute(page)}">
      <strong>${escapeHtml(title)}</strong>
      <span>${escapeHtml(copy)}</span>
    </button>
  `;
}

function primaryFlowButton(role) {
  if (role === "Requester") return `<button class="button primary" data-page="requester">Post task</button>`;
  if (role === "Implementor") return `<button class="button primary" data-page="discover">Open tasks</button>`;
  if (role === "Organization reviewer") return `<button class="button primary" data-page="review">Open reviews</button>`;
  return `<button class="button primary" data-page="integrations">Open Agent/API</button>`;
}

function filteredBoardTasks() {
  const tasks = visibleTasks();
  if (state.boardFilter === "public") return tasks.filter((item) => item.visibility === "public");
  if (state.boardFilter === "organization") return tasks.filter((item) => item.visibility === "organization");
  if (state.boardFilter === "mine") return tasks.filter((item) => item.requester === state.userId || item.assignee === state.userId);
  return tasks;
}

function activeTaskFor(tasks) {
  return tasks.find((item) => item.id === state.selectedTaskId) ?? tasks[0] ?? null;
}

function filterOption(value, label) {
  return option(value, label, state.boardFilter);
}

function option(value, label, selected) {
  return `<option value="${escapeAttribute(value)}" ${value === selected ? "selected" : ""}>${escapeHtml(label)}</option>`;
}

function rewardChips(reward) {
  if (reward.kind === "none") return `<span class="reward-chip muted-chip">No reward</span>`;
  const chips = [];
  if (reward.credits > 0) chips.push(`<span class="reward-chip">${reward.credits} credits</span>`);
  for (const collectible of reward.collectibles) chips.push(`<span class="reward-chip rare-chip">${escapeHtml(collectible)}</span>`);
  return chips.join("");
}

function rewardLabel(reward) {
  if (reward.kind === "none") return "no reward";
  const parts = [];
  if (reward.credits > 0) parts.push(`${reward.credits} credits`);
  if (reward.collectibles.length > 0) parts.push(reward.collectibles.join(", "));
  return parts.join(" + ");
}

function visibilityLabel(value) {
  return value === "organization" ? "Organization" : "Public marketplace";
}

function policyLabel(value) {
  if (value === policy.open) return "Open submissions";
  if (value === policy.approval) return "Approval required";
  return "Reservation required";
}

function lifecycleLabel(value) {
  if (value === lifecycle.draft) return "Draft";
  if (value === lifecycle.funded) return "Funded";
  if (value === lifecycle.closed) return "Closed";
  if (value === lifecycle.canceled) return "Canceled";
  return "Open";
}

function availabilityLabel(value) {
  return value.replaceAll("_", " ");
}

function areaLabel(value) {
  const labels = {
    "Field Ops": "Image labeling",
    "Ledger Bay": "Bookkeeping",
    "Foundry": "Content",
    "Cartography": "Geo data",
    "Vault": "Trust & safety",
    "Uplink": "Weather data",
    "Market": "Marketplace",
    "Archive": "Records",
    "Hangar": "Video",
    "Docs Deck": "Documentation",
    "Lab": "Lab data",
  };
  return labels[value] || value;
}

function boardTitleFor(role) {
  if (role === "Requester") return "Your posted tasks";
  if (role === "Implementor") return "Available tasks and your claimed work";
  if (role === "Organization reviewer") return "Organization review board";
  return "Agent-ready tasks";
}

function headlineFor(role) {
  if (role === "Requester") return "Post a task, set the reward, review results.";
  if (role === "Implementor") return "Find a task, do the work, get paid.";
  if (role === "Organization reviewer") return "Approve, review, and settle your organization's work.";
  return "Point your agent at a task and return a structured result.";
}

function copyFor(role) {
  if (role === "Requester") return "Post a task with instructions and a response schema, fund the reward, then review and settle submissions.";
  if (role === "Implementor") return "Browse available tasks, see exactly what each one provides and expects, submit a structured result, and get paid on acceptance.";
  if (role === "Organization reviewer") return "Approve reservations, request changes, reject, or accept and pay out your organization's tasks.";
  return "Each task ships REST and MCP steps so your agent can read the schema and submit a matching result.";
}

function hasReviewWork(taskItem) {
  return taskItem.reservations.some((reservation) => reservation.state === "requested") ||
    taskItem.submissions.some((submission) => submission.state === "submitted");
}

function balanceOf(userId) {
  return state.balances[userId] ?? 0;
}

function escrowHeldBy(userId) {
  return state.tasks
    .filter((taskItem) => taskItem.requester === userId)
    .reduce((sum, taskItem) => sum + (state.escrow?.[taskItem.id] ?? 0), 0);
}

function withoutEscrow(escrow, taskId) {
  const next = { ...escrow };
  delete next[taskId];
  return next;
}

function pushActivity(message) {
  state = { ...state, activityLog: [message, ...state.activityLog].slice(0, maxActivity) };
  saveNow();
  render();
}

function inventoryOf(userId) {
  return state.inventories[userId] ?? [];
}

function userName(userId) {
  return users.find((user) => user.id === userId)?.name ?? userId;
}

function userLink(userId) {
  if (!userId) return "unassigned";
  return `<a class="link-button" data-user-page="${escapeAttribute(userId)}" href="${userHref(userId)}">${escapeHtml(userName(userId))}</a>`;
}

function handleClick(event) {
  // Real anchors (task/user links) are left to the browser so left-click,
  // modifier-click (open in new tab), and right-click all behave natively;
  // hash changes are picked up by the hashchange listener.
  if (event.target.closest("a[href]")) return;

  const target = event.target.closest("[data-action], [data-page], [data-mode], [data-theme], [data-open-task], [data-task-action], [data-task], [data-user-page], [data-provider]");
  if (target === null) return;

  if (target.dataset.page !== undefined) {
    setState({ page: target.dataset.page, selectedTaskId: taskIdForPage(target.dataset.page), loginOpen: false });
    return;
  }
  if (target.dataset.mode !== undefined) {
    setState({ mode: target.dataset.mode });
    return;
  }
  if (target.dataset.theme !== undefined) {
    setState({ theme: target.dataset.theme });
    return;
  }
  if (target.dataset.openTask !== undefined) {
    setState({ selectedTaskId: target.dataset.openTask, page: "task", loginOpen: false });
    return;
  }
  if (target.dataset.taskAction !== undefined && target.dataset.taskId !== undefined) {
    state = { ...state, selectedTaskId: target.dataset.taskId };
    handleAction(target.dataset.taskAction, target.dataset.user, target.dataset.submission);
    return;
  }
  if (target.dataset.task !== undefined) {
    setState({ selectedTaskId: target.dataset.task, page: "task", loginOpen: false });
    return;
  }
  if (target.dataset.userPage !== undefined) {
    setState({ selectedUserId: target.dataset.userPage, page: "user", loginOpen: false });
    return;
  }
  if (target.dataset.provider !== undefined) {
    setState({ socialProvider: target.dataset.provider, loginOpen: false });
    return;
  }

  if (target.dataset.action === "add-field") {
    addDraftField();
    return;
  }
  if (target.dataset.action === "remove-field") {
    removeDraftField(Number(target.dataset.index));
    return;
  }

  handleAction(target.dataset.action, target.dataset.user, target.dataset.submission);
}

function handleCommit(event) {
  const target = event.target;
  if (target.dataset.schemaName !== undefined) {
    updateDraftField(Number(target.dataset.schemaName), "name", target.value);
    return;
  }
  if (target.dataset.schemaType !== undefined) {
    updateDraftField(Number(target.dataset.schemaType), "type", target.value);
    return;
  }
  if (target.dataset.schemaRequired !== undefined) {
    updateDraftField(Number(target.dataset.schemaRequired), "required", target.checked);
    return;
  }
  if (target.dataset.schemaEnum !== undefined) {
    updateDraftField(Number(target.dataset.schemaEnum), "enum", target.value);
    return;
  }
  if (target.dataset.field === undefined) return;

  const key = target.dataset.field;
  const value = target.type === "checkbox" ? target.checked : target.value;
  if (key === "userId") {
    choosePersona(value);
    return;
  }
  if (target.dataset.submission !== undefined) {
    updateReviewDraft(target.dataset.submission, key, value, { render: false });
    return;
  }
  if (target.type === "checkbox" || target.tagName === "SELECT") {
    setState({ [key]: value });
    return;
  }
  state = { ...state, [key]: value };
  saveNow();
}

function handleDraftInput(event) {
  const target = event.target;
  if (target.dataset.responseTask !== undefined) {
    state = { ...state, responseDrafts: { ...state.responseDrafts, [target.dataset.responseTask]: target.value } };
    saveSoon();
    return;
  }
  if (target.dataset.field === undefined) return;

  const key = target.dataset.field;
  if (key === "userId" || target.type === "checkbox" || target.tagName === "SELECT") return;
  if (target.dataset.submission !== undefined) {
    updateReviewDraft(target.dataset.submission, key, target.value, { render: false });
    return;
  }
  state = { ...state, [key]: target.value };
  saveSoon();
}

function updateReviewDraft(submissionId, key, value, options = { render: true }) {
  const field = key === "tipAmount" ? "tipAmount" : key;
  state = {
    ...state,
    reviewDrafts: {
      ...state.reviewDrafts,
      [submissionId]: {
        ...(state.reviewDrafts[submissionId] ?? {}),
        [field]: value,
      },
    },
  };
  if (options.render) {
    saveNow();
  } else {
    saveSoon();
  }
  if (options.render) render();
}

function handleAction(action, userId, submissionId) {
  switch (action) {
    case "reset":
      resetState();
      return;
    case "toggleLogin":
      setState({ loginOpen: !state.loginOpen });
      return;
    case "choosePersona":
      choosePersona(userId);
      return;
    case "create":
      createDraftTask();
      return;
    case "openReviewQueue":
      setState({ page: "review", loginOpen: false });
      return;
  }

  const taskItem = taskForAction(action);
  if (!taskItem) return;
  switch (action) {
    case "fund":
      fundMission(taskItem);
      return;
    case "openMission":
      openMission(taskItem);
      return;
    case "cancelMission":
      cancelMission(taskItem);
      return;
    case "reserve":
      reserveMission(taskItem);
      return;
    case "requestApproval":
      requestApproval(taskItem);
      return;
    case "submit":
      submitMission(taskItem);
      return;
    case "agentRun":
      agentRun(taskItem);
      return;
    case "approve":
      if (!canReviewTask(taskItem)) return;
      approveReservation(taskItem, userId);
      return;
    case "declineReservation":
      if (!canReviewTask(taskItem)) return;
      declineReservation(taskItem, userId);
      return;
    case "releaseReservation":
      if (!canReviewTask(taskItem)) return;
      releaseReservation(taskItem, userId);
      return;
    case "requestChanges":
      if (!canReviewTask(taskItem)) return;
      decideSubmission(taskItem, userId, submissionId, "changes_requested");
      return;
    case "reject":
      if (!canReviewTask(taskItem)) return;
      decideSubmission(taskItem, userId, submissionId, "rejected");
      return;
    case "accept":
      if (!canReviewTask(taskItem)) return;
      decideSubmission(taskItem, userId, submissionId, "accepted");
      return;
  }
}

function taskForAction(action) {
  if (isReviewAction(action)) return activeTaskFor(reviewableTasks());
  return activeTaskForCurrentPage();
}

function isReviewAction(action) {
  return ["approve", "declineReservation", "releaseReservation", "requestChanges", "reject", "accept"].includes(action);
}

function taskIdForPage(page) {
  if (page === "task") return selectedTask()?.id;
  const tasks = page === "review" ? reviewableTasks() : page === "discover" ? filteredBoardTasks() : visibleTasks();
  return activeTaskFor(tasks)?.id ?? selectedTask()?.id;
}

function activeTaskForCurrentPage() {
  if (state.page === "task") return selectedTask();
  if (state.page === "discover") return activeTaskFor(filteredBoardTasks());
  if (state.page === "requester") return activeTaskFor(tasksForRequester());
  if (state.page === "review") return activeTaskFor(reviewableTasks());
  return selectedTask();
}

function choosePersona(userId) {
  const nextUser = users.find((user) => user.id === userId) ?? users[0];
  const nextPage = nextUser.role === "Requester" ? "requester" : nextUser.role === "Implementor" ? "discover" : nextUser.role === "Organization reviewer" ? "review" : "integrations";
  const nextTask = state.tasks.find((item) => canSeeTask(item, nextUser)) ?? state.tasks[0];
  setState({ userId: nextUser.id, page: nextPage, selectedTaskId: nextTask.id, loginOpen: false });
}

function fundMission(taskItem) {
  const cost = taskItem.reward.credits || 0;
  if (cost > balanceOf(state.userId)) {
    pushActivity(`${selectedUser().name} can't fund ${taskItem.title}: needs ${cost} credits but has ${balanceOf(state.userId)} available.`);
    return;
  }
  if (cost > 0) {
    state = {
      ...state,
      balances: { ...state.balances, [state.userId]: balanceOf(state.userId) - cost },
      escrow: { ...state.escrow, [taskItem.id]: cost },
    };
  }
  const note = cost > 0 ? ` — ${cost} credits held in escrow` : "";
  updateTask(taskItem.id, () => ({ lifecycle: lifecycle.funded }), `${selectedUser().name} funded ${taskItem.title}${note}.`);
}

function openMission(taskItem) {
  updateTask(taskItem.id, () => ({ lifecycle: lifecycle.open, availability: availability.available }), `${selectedUser().name} opened ${taskItem.title} to the board.`);
}

function cancelMission(taskItem) {
  const held = state.escrow?.[taskItem.id] ?? 0;
  if (held > 0) {
    state = {
      ...state,
      balances: { ...state.balances, [state.userId]: balanceOf(state.userId) + held },
      escrow: withoutEscrow(state.escrow, taskItem.id),
    };
  }
  const note = held > 0 ? ` — ${held} credits refunded from escrow` : "";
  updateTask(taskItem.id, () => ({ lifecycle: lifecycle.canceled, availability: availability.closed }), `${selectedUser().name} canceled ${taskItem.title}${note}.`);
}

function reserveMission(taskItem) {
  updateTask(taskItem.id, () => ({
    assignee: state.userId,
    availability: availability.reserved,
    reservations: [{ id: `res-${taskItem.id}-${state.userId}`, by: state.userId, state: "active", expires: `${taskItem.reservationHours ?? 48}h` }],
  }), `${selectedUser().name} reserved ${taskItem.title}.`);
}

function requestApproval(taskItem) {
  updateTask(taskItem.id, (current) => ({
    availability: availability.awaitingApproval,
    reservations: [
      ...current.reservations.filter((reservation) => reservation.by !== state.userId),
      { id: `res-${taskItem.id}-${state.userId}`, by: state.userId, state: "requested", expires: `${taskItem.reservationHours ?? 48}h` },
    ].slice(-4),
  }), `${selectedUser().name} requested approval for ${taskItem.title}.`);
}

function submitMission(taskItem) {
  const response = responseDraftFor(taskItem);
  const validation = validateResponse(taskItem.schema, response);
  if (!validation.ok) {
    setState({ submitNote: { taskId: taskItem.id, ok: false, errors: validation.errors } });
    return;
  }
  state = { ...state, submitNote: { taskId: taskItem.id, ok: true, errors: [] } };
  updateTask(taskItem.id, (current) => ({
    availability: availability.submitted,
    submissions: [
      ...current.submissions.filter((submission) => submission.by !== state.userId || submission.state !== "submitted"),
      { id: `sub-${taskItem.id}-${state.userId}`, by: state.userId, state: "submitted", response, reviewNote: "", partialPayout: "", tip: "", ban: false },
    ].slice(-4),
  }), `${selectedUser().name} submitted a response for ${taskItem.title}.`);
}

function agentRun(taskItem) {
  updateTask(taskItem.id, (current) => ({
    assignee: "sol",
    availability: availability.submitted,
    reservations: [
      ...current.reservations.filter((reservation) => reservation.by !== "sol"),
      { id: `res-${taskItem.id}-sol`, by: "sol", state: "active", expires: "8h" },
    ].slice(-4),
    submissions: [
      ...current.submissions.filter((submission) => submission.by !== "sol" || submission.state !== "submitted"),
      { id: `sub-${taskItem.id}-sol`, by: "sol", state: "submitted", response: schemaTemplate(taskItem.schema) || "{}", reviewNote: "", partialPayout: "", tip: "", ban: false },
    ].slice(-4),
  }), `Sol simulated an MCP agent run for ${taskItem.title}.`);
}

function approveReservation(taskItem, userId) {
  updateTask(taskItem.id, (current) => ({
    assignee: userId,
    availability: availability.reserved,
    reservations: current.reservations.map((reservation) => reservation.by === userId ? { ...reservation, state: "active" } : reservation),
  }), `${selectedUser().name} approved ${userName(userId)} for ${taskItem.title}.`);
}

function declineReservation(taskItem, userId) {
  updateTask(taskItem.id, (current) => ({
    availability: current.assignee === userId ? availability.available : current.availability,
    assignee: current.assignee === userId ? "" : current.assignee,
    reservations: current.reservations.map((reservation) => reservation.by === userId ? { ...reservation, state: "declined" } : reservation),
  }), `${selectedUser().name} declined ${userName(userId)} for ${taskItem.title}.`);
}

function releaseReservation(taskItem, userId) {
  updateTask(taskItem.id, (current) => ({
    availability: availability.available,
    assignee: "",
    reservations: current.reservations.map((reservation) => reservation.by === userId ? { ...reservation, state: "released" } : reservation),
  }), `${selectedUser().name} released ${userName(userId)} from ${taskItem.title}.`);
}

function decideSubmission(taskItem, userId, submissionId, decision) {
  const submission = taskItem.submissions.find((item) => item.id === submissionId || item.by === userId);
  if (!submission || submission.state !== "submitted") return;
  const draft = reviewDraft(submission);
  const partial = decision === "changes_requested" ? 0 : Number.parseInt(draft.partialPayout, 10) || 0;
  const tip = decision === "changes_requested" ? 0 : Number.parseInt(draft.tipAmount, 10) || 0;
  const payout = decision === "accepted" ? (partial || taskItem.reward.credits || 0) : partial;
  const nextAvailability = decision === "accepted" ? availability.accepted : decision === "rejected" ? availability.rejected : availability.changesRequested;
  const nextLifecycle = decision === "accepted" ? lifecycle.closed : taskItem.lifecycle;
  const activity = decision === "accepted"
    ? `${selectedUser().name} accepted ${userName(userId)} on ${taskItem.title} and paid ${payout + tip} credits.`
    : decision === "rejected"
    ? `${selectedUser().name} rejected ${userName(userId)} on ${taskItem.title} with ${payout + tip} credits paid.`
    : `${selectedUser().name} requested changes from ${userName(userId)} on ${taskItem.title}.`;

  const settles = decision === "accepted" || decision === "rejected";
  const held = state.escrow?.[taskItem.id] ?? 0;
  if (settles) {
    state = {
      ...state,
      balances: {
        ...state.balances,
        [userId]: balanceOf(userId) + payout + tip,
        // The requester already escrowed `held` at funding time; release it and
        // net the actual payout + tip (refunding any unused escrow).
        [state.userId]: Math.max(0, balanceOf(state.userId) + held - payout - tip),
      },
      escrow: withoutEscrow(state.escrow, taskItem.id),
    };
  }

  if (decision === "accepted" && taskItem.reward.collectibles.length > 0) {
    state = {
      ...state,
      inventories: {
        ...state.inventories,
        [userId]: [...inventoryOf(userId), ...taskItem.reward.collectibles],
      },
    };
  }

  updateTask(taskItem.id, (current) => ({
    lifecycle: nextLifecycle,
    availability: nextAvailability,
    banned: draft.banImplementor ? [...(current.banned ?? []), userId] : current.banned,
    submissions: current.submissions.map((item) =>
      item.id === submission.id
        ? {
          ...item,
          state: decision,
          reviewNote: draft.reviewNote,
          partialPayout: String(payout),
          tip: String(tip),
          ban: draft.banImplementor,
          payoutCredits: payout,
          payoutCollectibles: decision === "accepted" ? taskItem.reward.collectibles : [],
        }
        : item
    ),
  }), activity);
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function escapeAttribute(value) {
  return escapeHtml(value).replaceAll("'", "&#39;");
}
