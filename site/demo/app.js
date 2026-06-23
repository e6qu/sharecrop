const storageKey = "sharecrop-demo-state-v3";

const seedTasks = [
  {
    id: "orchard-labels",
    title: "Label orchard photos",
    requester: "mara",
    assignee: "jules",
    visibility: "Public marketplace",
    policy: "Reservation required",
    reward: "45 credits + 1 collectible",
    state: "open",
    availability: "reserved",
    schema: '{"kind":"object","fields":{"labels":{"kind":"array","items":{"kind":"string"}}}}',
    description: "Classify orchard imagery and return a compact JSON response.",
    submissions: [
      {
        by: "jules",
        state: "submitted",
        response: '{"labels":["ripe","needs review"],"confidence":[91,74]}',
      },
    ],
    reservations: [{ by: "jules", state: "active" }],
  },
  {
    id: "invoice-cleanup",
    title: "Extract invoice totals",
    requester: "mara",
    assignee: "",
    visibility: "Organization",
    policy: "Approval required",
    reward: "30 credits",
    state: "open",
    availability: "awaiting approval",
    schema: '{"kind":"object","fields":{"total":{"kind":"decimal_string"}}}',
    description: "Return invoice totals from a small vendor batch.",
    submissions: [],
    reservations: [{ by: "ren", state: "requested" }],
  },
  {
    id: "badge-copy",
    title: "Draft badge copy",
    requester: "ren",
    assignee: "",
    visibility: "Public marketplace",
    policy: "Open submissions",
    reward: "no reward",
    state: "open",
    availability: "available",
    schema: '{"kind":"freeform"}',
    description: "Suggest short labels for a platform collectible badge.",
    submissions: [],
    reservations: [],
  },
];

const seedState = {
  mode: "light",
  theme: "corporate",
  page: "overview",
  loginOpen: false,
  userId: "mara",
  socialProvider: "",
  includeReserved: false,
  selectedTaskId: "orchard-labels",
  draftTitle: "Label orchard photos",
  draftDescription: "Classify orchard imagery and return a compact JSON response.",
  draftReward: "45 credits + 1 collectible",
  draftPolicy: "reservation",
  draftVisibility: "public",
  draftReservationHours: "48",
  responseText: '{"labels":["ripe","needs review"]}',
  reviewNote: "Please include confidence values for each label.",
  partialPayout: "18",
  tipAmount: "4",
  banImplementor: false,
  tasks: seedTasks,
};

const pages = [
  { id: "overview", label: "Overview" },
  { id: "discover", label: "Discover" },
  { id: "requester", label: "Create" },
  { id: "review", label: "Review" },
  { id: "integrations", label: "API & MCP" },
  { id: "settings", label: "Settings" },
];

const users = [
  { id: "mara", name: "Mara Chen", role: "Requester", balance: 180 },
  { id: "jules", name: "Jules Park", role: "Implementor", balance: 64 },
  { id: "ren", name: "Ren Ito", role: "Organization reviewer", balance: 112 },
  { id: "sol", name: "Sol Rivera", role: "Agent operator", balance: 100 },
];

const providers = ["Google", "Apple", "Microsoft", "Facebook", "X.com"];
const maxDemoTasks = 8;
const maxDemoSubmissions = 4;
const maxDemoReservations = 4;
let state = loadState();
let saveHandle = 0;

document.addEventListener("click", handleClick);
document.addEventListener("change", handleCommit);
document.addEventListener("input", handleDraftInput);

render();

function loadState() {
  const stored = localStorage.getItem(storageKey);
  if (stored === null) {
    return structuredClone(seedState);
  }

  try {
    return { ...structuredClone(seedState), ...JSON.parse(stored) };
  } catch (_error) {
    return structuredClone(seedState);
  }
}

function saveSoon() {
  clearTimeout(saveHandle);
  saveHandle = setTimeout(() => {
    saveToStorage();
  }, 120);
}

function saveNow() {
  clearTimeout(saveHandle);
  saveToStorage();
}

function saveToStorage() {
  try {
    localStorage.setItem(storageKey, JSON.stringify(state));
  } catch (_error) {
    localStorage.removeItem(storageKey);
    state = { ...structuredClone(seedState), page: "settings" };
    render();
  }
}

function setState(patch, options = { render: true }) {
  state = { ...state, ...patch };
  saveNow();
  if (options.render) {
    render();
  }
}

function resetState() {
  localStorage.removeItem(storageKey);
  state = structuredClone(seedState);
  render();
}

function selectedUser() {
  return users.find((user) => user.id === state.userId) ?? users[0];
}

function selectedTask() {
  return state.tasks.find((task) => task.id === state.selectedTaskId) ?? state.tasks[0];
}

function visibleTasks() {
  return state.tasks.filter((task) => {
    if (state.includeReserved) {
      return true;
    }
    return task.availability !== "reserved" || task.assignee === state.userId || task.requester === state.userId;
  });
}

function updateTask(taskId, change) {
  state = {
    ...state,
    tasks: state.tasks.map((task) => task.id === taskId ? { ...task, ...change(task) } : task),
  };
  saveNow();
  render();
}

function createDraftTask() {
  const id = `demo-${Date.now()}`;
  state = {
    ...state,
    selectedTaskId: id,
    page: "requester",
    tasks: [
      {
        id,
        title: state.draftTitle,
        requester: state.userId,
        assignee: "",
        visibility: state.draftVisibility === "organization" ? "Organization" : "Public marketplace",
        policy: state.draftPolicy === "approval" ? "Approval required" : "Reservation required",
        reward: state.draftReward,
        state: "draft",
        availability: "available",
        reservationHours: state.draftReservationHours,
        schema: '{"kind":"freeform"}',
        description: state.draftDescription,
        submissions: [],
        reservations: [],
      },
      ...state.tasks.filter((task) => !task.id.startsWith("demo-")),
      ...state.tasks.filter((task) => task.id.startsWith("demo-")).slice(0, maxDemoTasks - 4),
    ].slice(0, maxDemoTasks),
  };
  saveNow();
  render();
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
        <a class="button secondary" href="../docs/">Docs</a>
        <div class="account-menu">
          <button class="account-button" data-action="toggleLogin" aria-expanded="${state.loginOpen ? "true" : "false"}">
            <span>Viewing as</span>
            <strong>${user.name} · ${user.role}</strong>
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
      <span class="eyebrow">Login demo</span>
      <label for="demo-user">Select user</label>
      <select id="demo-user" data-field="userId">
        ${users.map((item) => `<option value="${item.id}" ${item.id === state.userId ? "selected" : ""}>${item.name} - ${item.role}</option>`).join("")}
      </select>
      <div class="provider-grid">
        ${providers.map((provider) => `<button class="button secondary compact-button" data-provider="${provider}">${provider}</button>`).join("")}
      </div>
      <p class="hint">${state.socialProvider ? `${state.socialProvider} sign-in is a placeholder.` : "Provider buttons are non-functional demo controls."}</p>
    </section>
  `;
}

function pageNavigation() {
  return `
    <nav class="page-tabs" aria-label="Demo pages">
      ${pages.map((page) => `<button class="tab ${state.page === page.id ? "active" : ""}" data-page="${page.id}">${page.label}</button>`).join("")}
    </nav>
  `;
}

function pageView() {
  if (state.page === "discover") return discoverPage();
  if (state.page === "requester") return requesterPage();
  if (state.page === "review") return reviewPage();
  if (state.page === "integrations") return integrationsPage();
  if (state.page === "settings") return settingsPage();
  return overviewPage();
}

function overviewPage() {
  const user = selectedUser();
  return `
    <section class="hero-panel">
      <div>
        <span class="eyebrow">${user.role}</span>
        <h1>${headlineFor(user.role)}</h1>
        <p>${copyFor(user.role)}</p>
      </div>
      <div class="summary-grid">
        ${metricCard("Open tasks", String(state.tasks.filter((task) => task.state === "open").length))}
        ${metricCard("Reservations", String(state.tasks.flatMap((task) => task.reservations).length))}
        ${metricCard("Submissions", String(state.tasks.flatMap((task) => task.submissions).length))}
        ${metricCard("Balance", `${user.balance} credits`)}
      </div>
    </section>
    <section class="panel">
      <span class="eyebrow">Choose a flow</span>
      <div class="flow-grid">
        ${flowCard("discover", "Implementor discovery", "Find eligible tasks, inspect rewards and policies, and submit work.")}
        ${flowCard("requester", "Requester setup", "Create a task and see the requester-owned task list.")}
        ${flowCard("review", "Review queue", "Approve reservations and accept or reject submitted work.")}
        ${flowCard("integrations", "API and MCP", "Copy the instructions a worker or local agent needs to provide a result.")}
      </div>
    </section>
  `;
}

function discoverPage() {
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Implementor flow</span>
          <h1>Discover available work</h1>
          <p>Workers see the public task list, choose whether to include reserved tasks, and act on one selected task.</p>
        </div>
        <label class="check-row">
          <input type="checkbox" data-field="includeReserved" ${state.includeReserved ? "checked" : ""}>
          Include reserved
        </label>
      </div>
      ${taskTable(visibleTasks())}
    </section>
    ${taskActionPanel()}
  `;
}

function requesterPage() {
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Requester flow</span>
          <h1>Create and prepare a task</h1>
          <p>The real app funds and opens tasks through the API. This demo keeps a local draft so the creation flow is quick to inspect.</p>
        </div>
      </div>
      <form class="create-form" data-form="create-task">
        <label for="draft-title">Title<input id="draft-title" data-field="draftTitle" value="${escapeHtml(state.draftTitle)}"></label>
        <label class="wide-field" for="draft-description">Description<textarea id="draft-description" data-field="draftDescription">${escapeHtml(state.draftDescription)}</textarea></label>
        <label for="draft-reward">Reward<input id="draft-reward" data-field="draftReward" value="${escapeHtml(state.draftReward)}"></label>
        <label for="draft-visibility">Visibility
          <select id="draft-visibility" data-field="draftVisibility">
            <option value="public" ${state.draftVisibility === "public" ? "selected" : ""}>Public marketplace</option>
            <option value="organization" ${state.draftVisibility === "organization" ? "selected" : ""}>Organization</option>
          </select>
        </label>
        <label for="draft-policy">Participation
          <select id="draft-policy" data-field="draftPolicy">
            <option value="reservation" ${state.draftPolicy === "reservation" ? "selected" : ""}>Reservation required</option>
            <option value="approval" ${state.draftPolicy === "approval" ? "selected" : ""}>Requester approval required</option>
          </select>
        </label>
        <label for="draft-reservation-hours">Reservation expiry<input id="draft-reservation-hours" data-field="draftReservationHours" value="${escapeHtml(state.draftReservationHours)}"></label>
        <button class="button primary" data-action="create" type="button">Add demo task</button>
      </form>
      <div class="flow-note">
        <strong>Next in the API-backed app:</strong>
        fund the reward, attach collectibles when needed, then open the task for discovery.
      </div>
    </section>
    <section class="panel">
      <span class="eyebrow">Tasks owned by ${selectedUser().name}</span>
      ${taskTable(state.tasks.filter((task) => task.requester === state.userId))}
    </section>
  `;
}

function reviewPage() {
  const task = selectedTask();
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Requester review flow</span>
          <h1>Review reservations and submissions</h1>
          <p>Reservation decisions and submission outcomes are kept together because they are usually handled by the same requester or reviewer.</p>
        </div>
        ${taskSelect(task)}
      </div>
      <div class="review-layout">
        <section class="sub-panel">
          <h2>Reservation queue</h2>
          ${reservationQueue(task)}
        </section>
        <section class="sub-panel">
          <h2>Submission decision</h2>
          ${submissionList(task)}
        </section>
      </div>
    </section>
  `;
}

function integrationsPage() {
  const task = selectedTask();
  return `
    <section class="panel">
      <div class="page-header">
        <div>
          <span class="eyebrow">Agent and API flow</span>
          <h1>Provide a result for ${escapeHtml(task.title)}</h1>
          <p>Task pages should expose direct REST and MCP instructions so people, scripts, and local agents can all submit results.</p>
        </div>
        ${taskSelect(task)}
      </div>
      <div class="code-grid">
        <article class="sub-panel">
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
          <p class="hint">Demo scopes: tasks_read, submissions_write, submissions_review. Revocation is available in the API-backed app.</p>
        </article>
        <article class="sub-panel">
          <span class="eyebrow">Worker REST</span>
          <h2>Submit result</h2>
          <pre>curl -X POST https://sharecrop.example/api/tasks/${task.id}/submissions \\
  -H "Authorization: Bearer &lt;ACCESS_TOKEN&gt;" \\
  -H "Content-Type: application/json" \\
  -d '{"response_json":"{}"}'</pre>
        </article>
        <article class="sub-panel">
          <span class="eyebrow">Worker MCP</span>
          <h2>Session workflow</h2>
          <pre>initialize -> Mcp-Session-Id
sharecrop.reserve_task
sharecrop.get_task_schema
sharecrop.submit_response</pre>
        </article>
        <article class="sub-panel">
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
          <h1>Theme, state, and placeholders</h1>
          <p>Settings are kept out of task workflows so the demo does not compete with the actual work surfaces.</p>
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
          <p>The demo stores edits only in this browser.</p>
          <button class="button primary" data-action="reset">Clear state</button>
        </section>
      </div>
    </section>
  `;
}

function taskSelect(task) {
  return `
    <select class="task-select" data-field="selectedTaskId">
      ${state.tasks.map((item) => `<option value="${item.id}" ${item.id === task.id ? "selected" : ""}>${escapeHtml(item.title)}</option>`).join("")}
    </select>
  `;
}

function modeButton(value, label) {
  return `<button class="button ${state.mode === value ? "primary" : "secondary"}" data-mode="${value}">${label}</button>`;
}

function themeButton(value, label) {
  return `<button class="theme-chip ${state.theme === value ? "selected" : ""}" data-theme="${value}">${label}</button>`;
}

function metricCard(label, value) {
  return `<div class="metric-card"><span>${label}</span><strong>${value}</strong></div>`;
}

function flowCard(page, title, copy) {
  return `
    <button class="flow-card" data-page="${page}">
      <strong>${title}</strong>
      <span>${copy}</span>
    </button>
  `;
}

function taskTable(tasks) {
  if (tasks.length === 0) {
    return `<p class="empty-state">No tasks match this view.</p>`;
  }
  return `
    <div class="task-table">
      <div class="task-table-head">
        <span>Task</span><span>Policy</span><span>Reward</span><span>Status</span>
      </div>
      ${tasks.map(taskRow).join("")}
    </div>
  `;
}

function taskRow(task) {
  return `
    <button class="task-table-row ${task.id === state.selectedTaskId ? "selected" : ""}" data-task="${task.id}">
      <span><strong>${escapeHtml(task.title)}</strong><small>${escapeHtml(task.description)}</small></span>
      <span>${escapeHtml(task.policy)}</span>
      <span>${escapeHtml(task.reward)}</span>
      <span><em>${escapeHtml(task.availability)}</em></span>
    </button>
  `;
}

function taskActionPanel() {
  const task = selectedTask();
  const action = nextAction(task);
  return `
    <section class="panel task-action-panel">
      <div>
        <span class="eyebrow">Selected task</span>
        <h2>${escapeHtml(task.title)}</h2>
        <p>${escapeHtml(task.description)}</p>
        <div class="badge-row">
          <span>${escapeHtml(task.state)}</span>
          <span>${escapeHtml(task.visibility)}</span>
          <span>${escapeHtml(task.reward)}</span>
          <span>${escapeHtml(task.policy)}</span>
        </div>
      </div>
      <div class="sub-panel">
        <h2>Implementor action</h2>
        <textarea data-field="responseText">${escapeHtml(state.responseText)}</textarea>
        <div class="row-actions">
          ${actionButtons(action)}
        </div>
        <p class="hint">${action.explanation}</p>
      </div>
    </section>
  `;
}

function nextAction(task) {
  if (task.availability === "closed" || task.state === "closed") {
    return { kind: "none", explanation: "This task is closed." };
  }
  if (task.policy === "Approval required" && task.assignee !== state.userId) {
    return { kind: "requestApproval", explanation: "This task requires requester approval before work can be submitted." };
  }
  if (task.policy === "Reservation required" && task.assignee !== state.userId) {
    return { kind: "reserve", explanation: "This task needs an exclusive reservation before submission." };
  }
  return { kind: "submit", explanation: "You are eligible to submit a response for this task." };
}

function actionButtons(action) {
  if (action.kind === "reserve") {
    return `<button class="button primary" data-action="reserve">Reserve task</button><button class="button secondary" disabled>Submit after reservation</button>`;
  }
  if (action.kind === "requestApproval") {
    return `<button class="button primary" data-action="requestApproval">Request approval</button><button class="button secondary" disabled>Submit after approval</button>`;
  }
  if (action.kind === "submit") {
    return `<button class="button primary" data-action="submit">Submit response</button>`;
  }
  return `<button class="button secondary" disabled>No action available</button>`;
}

function reservationQueue(task) {
  if (task.reservations.length === 0) {
    return `<p class="empty-state">No reservations for this task.</p>`;
  }
  return task.reservations.map((reservation) => `
    <div class="queue-row">
      <div>
        <strong>${userName(reservation.by)}</strong>
        <span>${escapeHtml(reservation.state)}</span>
      </div>
      <div class="row-actions">
        ${reservation.state === "requested" ? `<button class="button secondary" data-action="declineReservation" data-user="${reservation.by}">Decline</button><button class="button primary" data-action="approve" data-user="${reservation.by}">Approve</button>` : ""}
        ${reservation.state === "active" ? `<button class="button secondary" data-action="releaseReservation" data-user="${reservation.by}">Release</button>` : ""}
      </div>
    </div>
  `).join("");
}

function submissionList(task) {
  if (task.submissions.length === 0) {
    return `<p class="empty-state">No submissions for this task.</p>`;
  }
  return task.submissions.map((submission) => `
    <div class="submission-row">
      <div>
        <strong>${userName(submission.by)}</strong>
        <span>${escapeHtml(submission.state)}</span>
      </div>
      <code>${escapeHtml(submission.response)}</code>
      <label>Review note<textarea data-field="reviewNote">${escapeHtml(state.reviewNote)}</textarea></label>
      <div class="mini-grid">
        <label>Partial payout<input data-field="partialPayout" value="${escapeHtml(state.partialPayout)}"></label>
        <label>Tip<input data-field="tipAmount" value="${escapeHtml(state.tipAmount)}"></label>
      </div>
      <label class="check-row"><input type="checkbox" data-field="banImplementor" ${state.banImplementor ? "checked" : ""}> Ban implementor from this task</label>
      <div class="row-actions">
        <button class="button secondary" data-action="requestChanges" data-user="${submission.by}">Request changes</button>
        <button class="button secondary" data-action="reject" data-user="${submission.by}">Reject</button>
        <button class="button primary" data-action="accept" data-user="${submission.by}">Accept</button>
      </div>
    </div>
  `).join("");
}

function handleClick(event) {
  const target = event.target.closest("[data-action], [data-page], [data-mode], [data-theme], [data-task], [data-provider]");
  if (target === null) return;

  if (target.dataset.page !== undefined) {
    setState({ page: target.dataset.page, loginOpen: false });
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
  if (target.dataset.task !== undefined) {
    setState({ selectedTaskId: target.dataset.task });
    return;
  }
  if (target.dataset.provider !== undefined) {
    setState({ socialProvider: target.dataset.provider, loginOpen: false });
    return;
  }

  handleAction(target.dataset.action, target.dataset.user);
}

function handleCommit(event) {
  const target = event.target;
  if (target.dataset.field === undefined) return;

  const key = target.dataset.field;
  const value = target.type === "checkbox" ? target.checked : target.value;
  if (key === "userId") {
    setState({ userId: value, loginOpen: false });
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

  state = { ...state, [key]: target.value };
  saveSoon();
}

function handleAction(action, userId) {
  const task = selectedTask();
  if (action === "reset") resetState();
  if (action === "toggleLogin") setState({ loginOpen: !state.loginOpen });
  if (action === "create") createDraftTask();
  if (action === "reserve") {
    updateTask(task.id, () => ({
      assignee: state.userId,
      availability: "reserved",
      reservations: [{ by: state.userId, state: "active" }],
    }));
  }
  if (action === "requestApproval") {
    updateTask(task.id, (current) => ({
      availability: "awaiting approval",
      reservations: [
        ...current.reservations.filter((reservation) => reservation.by !== state.userId),
        { by: state.userId, state: "requested" },
      ].slice(-maxDemoReservations),
    }));
  }
  if (action === "submit") {
    updateTask(task.id, (current) => ({
      submissions: [
        ...current.submissions.filter((submission) => submission.by !== state.userId || submission.state !== "submitted"),
        { by: state.userId, state: "submitted", response: state.responseText },
      ].slice(-maxDemoSubmissions),
    }));
  }
  if (action === "approve") {
    updateTask(task.id, (current) => ({
      assignee: userId,
      availability: "reserved",
      reservations: current.reservations.map((reservation) =>
        reservation.by === userId ? { ...reservation, state: "active" } : reservation
      ),
    }));
  }
  if (action === "declineReservation") {
    updateTask(task.id, (current) => ({
      availability: current.assignee === userId ? "available" : current.availability,
      assignee: current.assignee === userId ? "" : current.assignee,
      reservations: current.reservations.map((reservation) =>
        reservation.by === userId ? { ...reservation, state: "declined" } : reservation
      ),
    }));
  }
  if (action === "releaseReservation") {
    updateTask(task.id, (current) => ({
      availability: "available",
      assignee: "",
      reservations: current.reservations.map((reservation) =>
        reservation.by === userId ? { ...reservation, state: "released" } : reservation
      ),
    }));
  }
  if (action === "reject") {
    updateTask(task.id, (current) => ({
      submissions: current.submissions.map((submission) =>
        submission.by === userId ? { ...submission, state: "rejected" } : submission
      ),
    }));
  }
  if (action === "requestChanges") {
    updateTask(task.id, (current) => ({
      submissions: current.submissions.map((submission) =>
        submission.by === userId ? { ...submission, state: "changes requested" } : submission
      ),
    }));
  }
  if (action === "accept") {
    updateTask(task.id, (current) => ({
      state: "closed",
      availability: "closed",
      submissions: current.submissions.map((submission) =>
        submission.by === userId ? { ...submission, state: "accepted" } : submission
      ),
    }));
  }
}

function headlineFor(role) {
  if (role === "Requester") return "Coordinate requested work without making Sharecrop a task runner.";
  if (role === "Implementor") return "Find eligible tasks, reserve work, and submit structured results.";
  if (role === "Organization reviewer") return "Review organization work through clear queues and outcomes.";
  return "Connect local agents through scoped HTTP and MCP instructions.";
}

function copyFor(role) {
  if (role === "Requester") return "Use focused pages for creating work, checking discovery, reviewing submissions, and copying integration instructions.";
  if (role === "Implementor") return "Discovery is a dedicated flow so available work, reservation state, and next actions are visible together.";
  if (role === "Organization reviewer") return "Review queues keep reservation and submission decisions in one place without crowding task creation.";
  return "API and MCP instructions are isolated from human review screens so agent setup stays easy to scan.";
}

function userName(userId) {
  const found = users.find((user) => user.id === userId);
  return found ? found.name : userId;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}
