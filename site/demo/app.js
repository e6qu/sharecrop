const storageKey = "sharecrop-demo-state-v5";

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
    difficulty: "A",
    visibility: "public",
    policy: policy.reservation,
    reward: rewardBundle(45, ["Ripe Lens"]),
    lifecycle: lifecycle.open,
    availability: availability.submitted,
    objective: "You are given 20 orchard photos by URL. Return one or more condition labels for each photo (ripe, unripe, or damaged) in the labels array. Accepted when every photo has at least one label from that set.",
    schema: '{"kind":"object","fields":{"labels":{"kind":"array","items":{"kind":"string"}}}}',
    reservations: [{ id: "res-orchard-jules", by: "jules", state: "active", expires: "48h" }],
    submissions: [{
      id: "sub-orchard-jules",
      by: "jules",
      state: "submitted",
      response: '{"labels":["ripe","needs review"],"confidence":[91,74]}',
      reviewNote: "",
      partialPayout: "",
      tip: "",
      ban: false,
    }],
    timeline: ["Mara posted the orchard image run.", "Jules reserved the mission.", "Jules submitted a labeled payload."],
  }),
  task({
    id: "invoice-cleanup",
    title: "Extract invoice totals",
    requester: "mara",
    assignee: "",
    area: "Ledger Bay",
    difficulty: "B",
    visibility: "organization",
    policy: policy.approval,
    reward: rewardCredits(30),
    lifecycle: lifecycle.open,
    availability: availability.awaitingApproval,
    objective: "Open the linked batch of 8 vendor invoices and add up their grand totals. Submit the combined amount as a decimal string in total, for example 1240.50. Accepted when it matches the verified sum within 0.01.",
    schema: '{"kind":"object","fields":{"total":{"kind":"decimal_string"}}}',
    reservations: [{ id: "res-invoice-ren", by: "ren", state: "requested", expires: "24h" }],
    timeline: ["Ren requested clearance to work on invoice extraction."],
  }),
  task({
    id: "badge-copy",
    title: "Draft badge copy",
    requester: "ren",
    assignee: "",
    area: "Foundry",
    difficulty: "C",
    visibility: "public",
    policy: policy.open,
    reward: rewardNone(),
    lifecycle: lifecycle.open,
    availability: availability.available,
    objective: "Propose 5 short name ideas (max 3 words each) for a new contributor achievement badge. Submit them as plain text, one per line. Accepted when at least 5 distinct, on-brand names are provided.",
    schema: '{"kind":"freeform"}',
    timeline: ["Ren opened the brief for public submissions."],
  }),
  task({
    id: "map-sensor-cleanup",
    title: "Standardize map-tile region names",
    requester: "mara",
    assignee: "",
    area: "Cartography",
    difficulty: "S",
    visibility: "public",
    policy: policy.reservation,
    reward: rewardCredits(80),
    lifecycle: lifecycle.open,
    availability: availability.available,
    objective: "You are given a CSV of 200 map tiles whose region column is spelled inconsistently. Pick the single canonical region name for the file and rate its overall data quality from 0 to 100. Submit region and quality. Accepted when the region matches the canonical list in the brief.",
    schema: '{"kind":"object","fields":{"region":{"kind":"string"},"quality":{"kind":"integer"}}}',
    timeline: ["Mission opened with credit escrow held."],
  }),
  task({
    id: "collectible-audit",
    title: "Verify collectible transfers",
    requester: "ren",
    assignee: "tala",
    area: "Vault",
    difficulty: "A",
    visibility: "organization",
    policy: policy.reservation,
    reward: rewardCollectible(["Vault Seal"]),
    lifecycle: lifecycle.open,
    availability: availability.changesRequested,
    objective: "Review the linked ledger of 50 collectible transfers and flag any that look fraudulent, such as the same item moved twice or a transfer to a banned account. Submit the transfer ids to investigate in suspicious_ids, each with a one-line reason in the thread.",
    schema: '{"kind":"object","fields":{"suspicious_ids":{"kind":"array","items":{"kind":"string"}}}}',
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
    difficulty: "B",
    visibility: "public",
    policy: policy.approval,
    reward: rewardBundle(25, ["Storm Pin"]),
    lifecycle: lifecycle.open,
    availability: availability.submitted,
    objective: "Using a scoped agent credential over MCP, fetch the current temperature in Celsius for the three cities named in the brief and submit them, in order, as decimal strings in readings. Accepted when three plausible readings are present.",
    schema: '{"kind":"object","fields":{"readings":{"kind":"array","items":{"kind":"decimal_string"}}}}',
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
    difficulty: "C",
    visibility: "public",
    policy: policy.open,
    reward: rewardCredits(12),
    lifecycle: lifecycle.open,
    availability: availability.available,
    objective: "You are given five marketplace task descriptions. Choose exactly two category tags for each from the provided tag list and submit them in the tags array, grouped per task. Accepted when every task has two tags from the list.",
    schema: '{"kind":"object","fields":{"tags":{"kind":"array","items":{"kind":"string"}}}}',
    timeline: ["Low-risk mission opened for open submissions."],
  }),
  task({
    id: "review-denied-batch",
    title: "Rejected crop batch",
    requester: "mara",
    assignee: "jules",
    area: "Field Ops",
    difficulty: "B",
    visibility: "public",
    policy: policy.open,
    reward: rewardCredits(18),
    lifecycle: lifecycle.open,
    availability: availability.rejected,
    objective: "Read the linked crop-inspection report that failed QA and write a 2 to 3 sentence summary of why it failed and what to fix. Submit as plain text. Accepted when the summary names the failing metric.",
    schema: '{"kind":"freeform"}',
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
    difficulty: "C",
    visibility: "organization",
    policy: policy.reservation,
    reward: rewardCredits(20),
    lifecycle: lifecycle.closed,
    availability: availability.accepted,
    objective: "Add each of the 42 accepted receipts in the linked folder to the archive index, one row per receipt with date, vendor, and amount. Submit a short note confirming the count archived. Accepted when the index row count matches.",
    schema: '{"kind":"freeform"}',
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
    difficulty: "A",
    visibility: "public",
    policy: policy.reservation,
    reward: rewardBundle(60, ["Drone Patch"]),
    lifecycle: lifecycle.draft,
    availability: availability.available,
    objective: "You are given 30 short drone clips by URL. Group each clip by field name and weather condition (for example field-3-clear) and submit the grouped clip identifiers in clips. Accepted when every clip is grouped.",
    schema: '{"kind":"object","fields":{"clips":{"kind":"array","items":{"kind":"string"}}}}',
    timeline: ["Draft mission waiting for funding and opening."],
  }),
  task({
    id: "funded-api-docs",
    title: "Polish API examples",
    requester: "ren",
    assignee: "",
    area: "Docs Deck",
    difficulty: "C",
    visibility: "public",
    policy: policy.approval,
    reward: rewardCredits(22),
    lifecycle: lifecycle.funded,
    availability: availability.available,
    objective: "Rewrite the three REST curl examples and one MCP example on the linked docs page so they run correctly against the current API. Submit the corrected snippets as plain text. Accepted when each example is copy-paste runnable.",
    schema: '{"kind":"freeform"}',
    timeline: ["Reward funded; requester has not opened the mission."],
  }),
  task({
    id: "expired-reservation-demo",
    title: "Expired soil sample reservation",
    requester: "mara",
    assignee: "",
    area: "Lab",
    difficulty: "B",
    visibility: "public",
    policy: policy.reservation,
    reward: rewardCredits(26),
    lifecycle: lifecycle.open,
    availability: availability.available,
    objective: "You are given a soil-sample CSV. List the sample ids that are missing a moisture reading in the missing array. Accepted when every row with a blank moisture value is included and no others.",
    schema: '{"kind":"object","fields":{"missing":{"kind":"array","items":{"kind":"string"}}}}',
    reservations: [{ id: "res-soil-jules", by: "jules", state: "expired", expires: "0h" }],
    timeline: ["A previous reservation expired and released the mission."],
  }),
];

const seedState = {
  mode: "light",
  theme: "blocky",
  page: "overview",
  loginOpen: false,
  userId: "mara",
  socialProvider: "",
  includeReserved: false,
  boardFilter: "all",
  selectedTaskId: "orchard-labels",
  selectedUserId: "mara",
  draftTitle: "Label orchard photos",
  draftDescription: "You are given 20 orchard photos by URL. Return one or more condition labels for each photo (ripe, unripe, or damaged) in the labels array.",
  draftRewardKind: "bundle",
  draftCredits: "45",
  draftCollectible: "Ripe Lens",
  draftPolicy: policy.reservation,
  draftVisibility: "public",
  draftReservationHours: "48",
  responseText: '{"labels":["ripe","needs review"]}',
  reviewDrafts: {},
  localTaskSeq: 1,
  balances: Object.fromEntries(users.map((user) => [user.id, user.balance])),
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
  return taskItem.requester === user.id || user.role === "Organization reviewer";
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

function createDraftTask() {
  const id = `demo-${state.localTaskSeq}`;
  const newTask = task({
    id,
    title: state.draftTitle || "Untitled mission",
    requester: state.userId,
    assignee: "",
    area: "Custom Board",
    difficulty: "C",
    visibility: state.draftVisibility,
    policy: state.draftPolicy,
    reward: draftReward(),
    lifecycle: lifecycle.draft,
    availability: availability.available,
    reservationHours: state.draftReservationHours,
    schema: '{"kind":"freeform"}',
    objective: state.draftDescription || "Describe the requested result.",
    timeline: [`${selectedUser().name} drafted the mission.`],
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
            <span>Persona</span>
            <strong>${escapeHtml(user.name)}</strong>
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
      <span class="eyebrow">Choose persona</span>
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
    `<li><button class="link-button" data-open-task="${escapeAttribute(item.id)}">${escapeHtml(item.title)}</button> <span class="muted">${escapeHtml(availabilityLabel(item.availability))}</span></li>`;
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

function overviewPage() {
  const user = selectedUser();
  const taskItem = selectedTask();
  return `
    <section class="command-grid">
      <div class="hero-panel command-hero">
        <div>
          <span class="eyebrow">${escapeHtml(user.role)}</span>
          <h1>${escapeHtml(headlineFor(user.role))}</h1>
          <p>${escapeHtml(copyFor(user.role))}</p>
          <div class="row-actions">
            ${primaryFlowButton(user.role)}
            <button class="button secondary" data-page="settings">Tune demo</button>
          </div>
        </div>
        <div class="summary-grid">
          ${metricCard("Open tasks", String(state.tasks.filter((item) => item.lifecycle === lifecycle.open).length))}
          ${metricCard("Review signals", String(reviewSignals()))}
          ${metricCard("Credits", `${balanceOf(user.id)}`)}
          ${metricCard("Collectibles", String(inventoryOf(user.id).length))}
        </div>
      </div>
      ${missionBriefing(taskItem)}
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
        <label class="wide-field" for="draft-description">Objective<textarea id="draft-description" data-field="draftDescription">${escapeHtml(state.draftDescription)}</textarea></label>
        <label for="draft-reward-kind">Reward kind
          <select id="draft-reward-kind" data-field="draftRewardKind">
            ${option("none", "No reward", state.draftRewardKind)}
            ${option("credits", "Credits", state.draftRewardKind)}
            ${option("collectible", "Collectible", state.draftRewardKind)}
            ${option("bundle", "Credits + collectible", state.draftRewardKind)}
          </select>
        </label>
        <label for="draft-credits">Credits<input id="draft-credits" data-field="draftCredits" value="${escapeAttribute(state.draftCredits)}"></label>
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
        <label for="draft-reservation-hours">Reservation expiry<input id="draft-reservation-hours" data-field="draftReservationHours" value="${escapeAttribute(state.draftReservationHours)}"></label>
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
          <h1>Review reservations and submitted payloads</h1>
          <p>Approve access, request changes, reject with fair partial payout, or accept and settle the reward crate.</p>
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
      </div>` : `<p class="empty-state">No reservations or submitted payloads need this persona's review.</p>`}
    </section>
    ${activityFeed()}
  `;
}

function integrationsPage() {
  const taskItem = selectedTask();
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Agent/API console</span>
          <h1>Provide a result for ${escapeHtml(taskItem.title)}</h1>
          <p>Each task carries direct REST and MCP instructions so people, scripts, and local agents can all submit results.</p>
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
      "url": "https://sharecrop.example/mcp",
      "headers": { "Authorization": "Bearer &lt;AGENT_TOKEN&gt;" }
    }
  }
}</pre>
          <button class="button primary" data-action="agentRun">Run as Sol agent</button>
          <p class="hint">Demo scopes: tasks_read, submissions_write, submissions_review. The action adds a Sol-labeled MCP submission to this task.</p>
        </article>
        <article class="sub-panel uplink-panel">
          <span class="eyebrow">Worker REST</span>
          <h2>Submit result</h2>
          <pre>curl -X POST https://sharecrop.example/api/tasks/${escapeHtml(taskItem.id)}/submissions \\
  -H "Authorization: Bearer &lt;ACCESS_TOKEN&gt;" \\
  -H "Content-Type: application/json" \\
  -d '{"response_json":"{}"}'</pre>
        </article>
        <article class="sub-panel uplink-panel">
          <span class="eyebrow">Worker MCP</span>
          <h2>Session workflow</h2>
          <pre>initialize -> Mcp-Session-Id
sharecrop.reserve_task
sharecrop.get_task_schema
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
          <button class="link-button task-title-link" data-open-task="${escapeAttribute(taskItem.id)}">${escapeHtml(taskItem.title)}</button>
          <span class="rank-badge inline-rank">${escapeHtml(difficultyLabel(taskItem.difficulty))}</span>
        </div>
        <p>${escapeHtml(taskItem.objective)}</p>
        <div class="mission-meta">
          <span>${escapeHtml(areaLabel(taskItem.area))}</span>
          <span>${escapeHtml(policyLabel(taskItem.policy))}</span>
          <span>${escapeHtml(taskItem.assignee ? `Assigned: ${userName(taskItem.assignee)}` : `Requester: ${userName(taskItem.requester)}`)}</span>
        </div>
      </div>
      <div class="task-widget">
        <span class="action-chip">${escapeHtml(availabilityLabel(taskItem.availability))}</span>
        <span class="action-chip">${escapeHtml(cardActionLabel(action))}</span>
        <div class="reward-row">${rewardChips(taskItem.reward)}</div>
      </div>
      <div class="task-actions">
        ${taskListAction(action, taskItem)}
        <button class="button secondary" data-open-task="${escapeAttribute(taskItem.id)}">Open task page</button>
      </div>
    </article>
  `;
}

function taskListAction(action, taskItem) {
  if (action.kind === "none" || action.kind === "submit" || action.kind === "agentRun") return "";
  if (action.kind === "openReviewQueue") return `<button class="button primary" data-page="review">Review queue</button>`;
  return `<button class="button primary" data-task-action="${escapeAttribute(action.kind)}" data-task-id="${escapeAttribute(taskItem.id)}">${escapeHtml(cardActionLabel(action))}</button>`;
}

function missionBoard(tasks, context, selectedId = selectedTask()?.id) {
  const lanes = boardLanes(tasks);
  return `
    <div class="mission-board" data-context="${escapeAttribute(context)}">
      ${lanes.map((lane) => `
        <section class="mission-lane ${lane.tasks.length === 0 ? "empty-lane" : ""}" data-lane="${escapeAttribute(lane.id)}">
          <div class="lane-header">
            <strong>${escapeHtml(lane.label)}</strong>
            <span>${lane.tasks.length}</span>
          </div>
          ${lane.tasks.length === 0 ? `<p class="empty-state">No missions in this lane.</p>` : lane.tasks.map((item) => missionCard(item, selectedId)).join("")}
        </section>
      `).join("")}
    </div>
  `;
}

function boardLanes(tasks) {
  const lanes = [
    { id: "available", label: "Available", tasks: [] },
    { id: "reserved", label: "Reserved", tasks: [] },
    { id: "awaiting", label: "Awaiting approval", tasks: [] },
    { id: "submitted", label: "Submitted", tasks: [] },
    { id: "settled", label: "Settled", tasks: [] },
  ];
  for (const taskItem of tasks) {
    lanes.find((lane) => lane.id === laneForTask(taskItem)).tasks.push(taskItem);
  }
  return lanes;
}

function laneForTask(taskItem) {
  if ([availability.accepted, availability.rejected, availability.closed].includes(taskItem.availability) || taskItem.lifecycle === lifecycle.closed) return "settled";
  if ([availability.submitted, availability.changesRequested].includes(taskItem.availability)) return "submitted";
  if (taskItem.availability === availability.awaitingApproval) return "awaiting";
  if (taskItem.availability === availability.reserved) return "reserved";
  return "available";
}

function missionCard(taskItem, selectedId) {
  const selected = taskItem.id === selectedId ? "selected" : "";
  const next = nextAction(taskItem);
  return `
    <button class="mission-card ${selected}" data-task="${escapeAttribute(taskItem.id)}" aria-pressed="${selected ? "true" : "false"}">
      <span class="rank-badge">${escapeHtml(difficultyLabel(taskItem.difficulty))}</span>
      <strong>${escapeHtml(taskItem.title)}</strong>
      <small>${escapeHtml(taskItem.objective)}</small>
      <div class="mission-meta">
        <span>${escapeHtml(areaLabel(taskItem.area))}</span>
        <span>${escapeHtml(policyLabel(taskItem.policy))}</span>
        <span>${escapeHtml(taskItem.assignee ? `Assigned: ${userName(taskItem.assignee)}` : `Requester: ${userName(taskItem.requester)}`)}</span>
      </div>
      <span class="action-chip">${escapeHtml(cardActionLabel(next))}</span>
      <div class="reward-row">${rewardChips(taskItem.reward)}</div>
    </button>
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
        <div class="badge-row">
          <span>${escapeHtml(lifecycleLabel(taskItem.lifecycle))}</span>
          <span>${escapeHtml(visibilityLabel(taskItem.visibility))}</span>
          <span>${escapeHtml(policyLabel(taskItem.policy))}</span>
          <span>${escapeHtml(availabilityLabel(taskItem.availability))}</span>
        </div>
        <ul class="objective-list">
          <li>Requester: ${userLink(taskItem.requester)}</li>
          <li>Assignee: ${userLink(taskItem.assignee)}</li>
          <li>Reward: ${escapeHtml(rewardLabel(taskItem.reward))}</li>
        </ul>
        <div class="schema-block">
          <span>Expected response schema</span>
          <pre>${escapeHtml(taskItem.schema)}</pre>
        </div>
      </div>
      <div class="sub-panel action-console">
        <h2>${escapeHtml(action.title)}</h2>
        ${actionPayload(action)}
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
  if (submission?.state === "submitted") {
    return { kind: "none", title: "Payload received", explanation: "The requester has a submitted result to review." };
  }
  if (submission?.state === "changes_requested") {
    return { kind: "submit", title: "Revise payload", explanation: "Changes were requested. Submit an updated payload for the same task." };
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
  return { kind: "submit", title: "Submit payload", explanation: "You are eligible to submit a result for this task." };
}

function actionButtons(action) {
  if (action.kind === "fund") return `<button class="button primary" data-action="fund">Fund reward</button><button class="button secondary" data-action="cancelMission">Cancel</button>`;
  if (action.kind === "openMission") return `<button class="button primary" data-action="openMission">Open task</button><button class="button secondary" data-action="cancelMission">Cancel</button>`;
  if (action.kind === "reserve") return `<button class="button primary" data-action="reserve">Reserve task</button>`;
  if (action.kind === "requestApproval") return `<button class="button primary" data-action="requestApproval">Request approval</button>`;
  if (action.kind === "submit") return `<button class="button primary" data-action="submit">Submit payload</button>`;
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

function actionPayload(action) {
  if (action.kind === "submit" || action.kind === "agentRun") {
    return `<label for="response-text">Submission payload<textarea id="response-text" data-field="responseText">${escapeHtml(state.responseText)}</textarea></label>`;
  }
  return "";
}

function cardActionLabel(action) {
  if (action.kind === "fund") return "Fund";
  if (action.kind === "openMission") return "Open";
  if (action.kind === "reserve") return "Reserve";
  if (action.kind === "requestApproval") return "Request approval";
  if (action.kind === "submit") return "Submit";
  if (action.kind === "agentRun") return "Agent run";
  if (action.kind === "openReviewQueue") return "Review";
  return action.title;
}

function reservationQueue(taskItem) {
  if (taskItem.reservations.length === 0) return `<p class="empty-state">No reservations for this task.</p>`;
  return taskItem.reservations.map((reservation) => `
    <div class="queue-row">
      <div>
        <strong>${escapeHtml(userName(reservation.by))}</strong>
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
          <strong>${escapeHtml(userName(submission.by))}</strong>
          <span class="action-chip">${escapeHtml(submissionStateLabel(submission.state))}</span>
        </div>
        <code>${escapeHtml(submission.response)}</code>
        <div class="payout-strip">
          <span>Reward available</span>
          <strong>${escapeHtml(rewardLabel(taskItem.reward))}</strong>
        </div>
        <label>Review note<textarea data-field="reviewNote" data-submission="${escapeAttribute(submission.id)}">${escapeHtml(draft.reviewNote)}</textarea></label>
        <div class="mini-grid">
          <label>Partial payout<input data-field="partialPayout" data-submission="${escapeAttribute(submission.id)}" value="${escapeAttribute(draft.partialPayout)}"></label>
          <label>Tip<input data-field="tipAmount" data-submission="${escapeAttribute(submission.id)}" value="${escapeAttribute(draft.tipAmount)}"></label>
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
        <strong>${escapeHtml(userName(submission.by))}</strong>
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
  if (value === "changes_requested") return "Waiting for revised payload";
  if (value === "accepted") return "Accepted and settled";
  if (value === "rejected") return "Rejected";
  return "Decision recorded";
}

function outcomeCopy(value, paid) {
  if (value === "changes_requested") return "The implementor keeps this task and can submit an updated payload.";
  if (value === "accepted") return `The requester settled this submission with ${paid} credits plus any collectible reward.`;
  if (value === "rejected") return `The requester closed this submission with ${paid} credits paid.`;
  return "This submission no longer has active decision controls.";
}

function reviewDraft(submission) {
  return {
    reviewNote: submission.reviewNote || "Looks close. Add missing evidence if needed.",
    partialPayout: submission.partialPayout || "18",
    tipAmount: submission.tip || "4",
    banImplementor: submission.ban || false,
    ...(state.reviewDrafts[submission.id] ?? {}),
  };
}

function activityFeed() {
  return `
    <section class="panel activity-feed">
      <span class="eyebrow">Comms feed</span>
      ${state.activityLog.map((entry) => `<p>${escapeHtml(entry)}</p>`).join("")}
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

function difficultyLabel(value) {
  const labels = { S: "High effort", A: "Above average", B: "Moderate", C: "Light" };
  return labels[value] || value;
}

function boardTitleFor(role) {
  if (role === "Requester") return "Posted missions and market visibility";
  if (role === "Implementor") return "Available contracts and claimed work";
  if (role === "Organization reviewer") return "Organization dispatch board";
  return "Agent-ready missions and MCP runs";
}

function headlineFor(role) {
  if (role === "Requester") return "Track tasks, rewards, and reviews.";
  if (role === "Implementor") return "Pick a mission, claim the slot, and deliver the payload.";
  if (role === "Organization reviewer") return "Keep mission access, submissions, and payouts under control.";
  return "Connect local agents to missions through scoped uplinks.";
}

function copyFor(role) {
  if (role === "Requester") return "Post missions, attach reward crates, review submissions, and watch the board change as work moves.";
  if (role === "Implementor") return "The task list highlights what is available, reserved, awaiting approval, submitted, or settled.";
  if (role === "Organization reviewer") return "The review queue exposes approvals, changes, rejection, bans, partial payouts, and acceptance.";
  return "The Agent/API screen shows REST and MCP instructions and can simulate an agent submission.";
}

function reviewSignals() {
  return state.tasks.filter(hasReviewWork).length;
}

function hasReviewWork(taskItem) {
  return taskItem.reservations.some((reservation) => reservation.state === "requested") ||
    taskItem.submissions.some((submission) => submission.state === "submitted");
}

function balanceOf(userId) {
  return state.balances[userId] ?? 0;
}

function inventoryOf(userId) {
  return state.inventories[userId] ?? [];
}

function userName(userId) {
  return users.find((user) => user.id === userId)?.name ?? userId;
}

function userLink(userId) {
  if (!userId) return "unassigned";
  return `<button class="link-button" data-user-page="${escapeAttribute(userId)}">${escapeHtml(userName(userId))}</button>`;
}

function handleClick(event) {
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

  handleAction(target.dataset.action, target.dataset.user, target.dataset.submission);
}

function handleCommit(event) {
  const target = event.target;
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
  updateTask(taskItem.id, () => ({ lifecycle: lifecycle.funded }), `${selectedUser().name} funded ${taskItem.title}.`);
}

function openMission(taskItem) {
  updateTask(taskItem.id, () => ({ lifecycle: lifecycle.open, availability: availability.available }), `${selectedUser().name} opened ${taskItem.title} to the board.`);
}

function cancelMission(taskItem) {
  updateTask(taskItem.id, () => ({ lifecycle: lifecycle.canceled, availability: availability.closed }), `${selectedUser().name} canceled ${taskItem.title}.`);
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
  updateTask(taskItem.id, (current) => ({
    availability: availability.submitted,
    submissions: [
      ...current.submissions.filter((submission) => submission.by !== state.userId || submission.state !== "submitted"),
      { id: `sub-${taskItem.id}-${state.userId}`, by: state.userId, state: "submitted", response: state.responseText, reviewNote: "", partialPayout: "", tip: "", ban: false },
    ].slice(-4),
  }), `${selectedUser().name} submitted a payload for ${taskItem.title}.`);
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
      { id: `sub-${taskItem.id}-sol`, by: "sol", state: "submitted", response: '{"submitted_by":"agent","status":"ready"}', reviewNote: "", partialPayout: "", tip: "", ban: false },
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

  state = {
    ...state,
    balances: {
      ...state.balances,
      [userId]: balanceOf(userId) + payout + tip,
      [state.userId]: Math.max(0, balanceOf(state.userId) - tip),
    },
  };

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
