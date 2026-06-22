const storageKey = "sharecrop-demo-state-v2";

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
  draftReward: "45 credits + 1 collectible",
  draftPolicy: "reservation",
  responseText: '{"labels":["ripe","needs review"]}',
  reviewNote: "Please include confidence values for each label.",
  partialPayout: "18",
  tipAmount: "4",
  banImplementor: false,
  tasks: [
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
  ],
};

const pages = [
  { id: "overview", label: "Overview" },
  { id: "discover", label: "Discover" },
  { id: "requester", label: "Requester" },
  { id: "review", label: "Review" },
  { id: "integrations", label: "API & MCP" },
  { id: "settings", label: "Demo settings" },
];

const users = [
  { id: "mara", name: "Mara Chen", role: "Requester", balance: 180 },
  { id: "jules", name: "Jules Park", role: "Implementor", balance: 64 },
  { id: "ren", name: "Ren Ito", role: "Organization reviewer", balance: 112 },
  { id: "sol", name: "Sol Rivera", role: "Agent operator", balance: 100 },
];

const providers = ["Google", "Apple", "Microsoft", "Facebook", "X.com"];

let state = loadState();

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

function saveState() {
  localStorage.setItem(storageKey, JSON.stringify(state));
}

function setState(patch) {
  state = { ...state, ...patch };
  saveState();
  render();
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
  saveState();
  render();
}

function reserveTask(taskId) {
  updateTask(taskId, () => ({
    assignee: state.userId,
    availability: "reserved",
    reservations: [{ by: state.userId, state: "active" }],
  }));
}

function requestApproval(taskId) {
  updateTask(taskId, (task) => ({
    availability: "awaiting approval",
    reservations: [...task.reservations, { by: state.userId, state: "requested" }],
  }));
}

function submitResponse(taskId) {
  updateTask(taskId, (task) => ({
    submissions: [...task.submissions, { by: state.userId, state: "submitted", response: state.responseText }],
  }));
}

function approveReservation(taskId, userId) {
  updateTask(taskId, (task) => ({
    assignee: userId,
    availability: "reserved",
    reservations: task.reservations.map((reservation) =>
      reservation.by === userId ? { ...reservation, state: "active" } : reservation
    ),
  }));
}

function rejectSubmission(taskId, userId) {
  updateTask(taskId, (task) => ({
    submissions: task.submissions.map((submission) =>
      submission.by === userId ? { ...submission, state: "rejected" } : submission
    ),
  }));
}

function acceptSubmission(taskId, userId) {
  updateTask(taskId, (task) => ({
    state: "closed",
    availability: "closed",
    submissions: task.submissions.map((submission) =>
      submission.by === userId ? { ...submission, state: "accepted" } : submission
    ),
  }));
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
        visibility: "Public marketplace",
        policy: state.draftPolicy === "approval" ? "Approval required" : "Reservation required",
        reward: state.draftReward,
        state: "draft",
        availability: "available",
        schema: '{"kind":"freeform"}',
        description: "Demo-created task stored only in this browser.",
        submissions: [],
        reservations: [],
      },
      ...state.tasks,
    ],
  };
  saveState();
  render();
}

function render() {
  document.body.dataset.mode = state.mode;
  document.body.dataset.theme = state.theme;
  const app = document.getElementById("app");
  app.innerHTML = `
    <main class="app-shell">
      ${topbar()}
      <section class="workspace">
        <aside class="rail">${rail()}</aside>
        <section class="content-shell">
          ${pageNavigation()}
          ${pageView()}
        </section>
      </section>
    </main>
  `;
  bindEvents();
}

function topbar() {
  const user = selectedUser();
  return `
    <nav class="topbar">
      <a class="brand" href="../">Sharecrop</a>
      <div class="topbar-actions">
        <a class="button secondary" href="../docs/">Docs</a>
        <button class="button ghost" data-action="reset">Clear demo state</button>
        <div class="account-menu">
          <button class="account-button" data-action="toggleLogin">
            <span>Guest</span>
            <strong>${user.name}</strong>
          </button>
          ${state.loginOpen ? loginPanel() : ""}
        </div>
      </div>
    </nav>
  `;
}

function rail() {
  return `
    <section class="rail-card">
      <span class="eyebrow">Theme</span>
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
    <section class="rail-card">
      <span class="eyebrow">Session</span>
      <div class="account-line">
        <span>${selectedUser().role}</span>
        <strong>${selectedUser().balance} credits</strong>
      </div>
      <p class="hint">Use the account menu in the top-right corner to switch demo users or preview login methods.</p>
    </section>
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

function modeButton(value, label) {
  return `<button class="button ${state.mode === value ? "primary" : "secondary"}" data-mode="${value}">${label}</button>`;
}

function themeButton(value, label) {
  return `<button class="theme-chip ${state.theme === value ? "selected" : ""}" data-theme="${value}">${label}</button>`;
}

function pageNavigation() {
  return `
    <nav class="page-tabs" aria-label="Demo pages">
      ${pages.map((page) => `<button class="tab ${state.page === page.id ? "active" : ""}" data-page="${page.id}">${page.label}</button>`).join("")}
    </nav>
  `;
}

function pageView() {
  if (state.page === "discover") {
    return discoverPage();
  }
  if (state.page === "requester") {
    return requesterPage();
  }
  if (state.page === "review") {
    return reviewPage();
  }
  if (state.page === "integrations") {
    return integrationsPage();
  }
  if (state.page === "settings") {
    return settingsPage();
  }
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
        ${metricCard("Demo balance", `${user.balance}`)}
      </div>
    </section>
    <section class="panel">
      <div class="section-header">
        <div>
          <span class="eyebrow">Workflow map</span>
          <h2>What this demo covers</h2>
        </div>
      </div>
      <div class="story-grid">
        ${storyCard("Requester", "Create tasks, set rewards, choose participation policy, and review results.")}
        ${storyCard("Implementor", "Discover available work, reserve or request approval, and submit JSON responses.")}
        ${storyCard("Reviewer", "Approve reservations, accept submissions, reject work, request changes, and issue fair payouts.")}
        ${storyCard("Agent", "Use REST and MCP instructions from a task page to provide the result.")}
      </div>
    </section>
  `;
}

function discoverPage() {
  return `
    <section class="panel">
      <div class="section-header">
        <div>
          <span class="eyebrow">Discover</span>
          <h1>Find available work</h1>
          <p>Workers see open public tasks. Reserved tasks stay hidden unless explicitly included.</p>
        </div>
        <label class="check-row">
          <input type="checkbox" data-field="includeReserved" ${state.includeReserved ? "checked" : ""}>
          Include reserved
        </label>
      </div>
      ${taskTable(visibleTasks(), "discover")}
    </section>
    ${taskActionPanel()}
  `;
}

function requesterPage() {
  return `
    <section class="panel split-panel">
      <div>
        <span class="eyebrow">Requester</span>
        <h1>Create and prepare work</h1>
        <p>Set the task shape first, then fund and open it in the real API-backed app.</p>
      </div>
      <form class="form-grid" data-form="create-task">
        <label for="draft-title">Title<input id="draft-title" data-field="draftTitle" value="${escapeHtml(state.draftTitle)}"></label>
        <label for="draft-reward">Reward<input id="draft-reward" data-field="draftReward" value="${escapeHtml(state.draftReward)}"></label>
        <label for="draft-policy">Participation
          <select id="draft-policy" data-field="draftPolicy">
            <option value="reservation" ${state.draftPolicy === "reservation" ? "selected" : ""}>Reservation required</option>
            <option value="approval" ${state.draftPolicy === "approval" ? "selected" : ""}>Requester approval required</option>
          </select>
        </label>
        <button class="button primary" data-action="create" type="button">Add demo task</button>
      </form>
    </section>
    <section class="panel">
      <div class="section-header">
        <div>
          <span class="eyebrow">Requester task list</span>
          <h2>Tasks you own</h2>
        </div>
      </div>
      ${taskTable(state.tasks.filter((task) => task.requester === state.userId), "requester")}
    </section>
  `;
}

function reviewPage() {
  const task = selectedTask();
  return `
    <section class="panel">
      <div class="section-header">
        <div>
          <span class="eyebrow">Review</span>
          <h1>Reservations and submissions</h1>
          <p>Requester decisions stay separate from discovery and creation work.</p>
        </div>
        <select class="task-select" data-field="selectedTaskId">
          ${state.tasks.map((item) => `<option value="${item.id}" ${item.id === task.id ? "selected" : ""}>${item.title}</option>`).join("")}
        </select>
      </div>
      <div class="review-layout">
        <section class="sub-panel">
          <h2>Reservation queue</h2>
          ${reservationQueue(task)}
        </section>
        <section class="sub-panel">
          <h2>Submission review</h2>
          ${reviewControls()}
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
      <div class="section-header">
        <div>
          <span class="eyebrow">API & MCP</span>
          <h1>Instructions for ${escapeHtml(task.title)}</h1>
          <p>Each task page needs clear REST and MCP instructions so workers and agents can provide results.</p>
        </div>
        <select class="task-select" data-field="selectedTaskId">
          ${state.tasks.map((item) => `<option value="${item.id}" ${item.id === task.id ? "selected" : ""}>${item.title}</option>`).join("")}
        </select>
      </div>
      <div class="code-grid">
        <article class="sub-panel">
          <span class="eyebrow">REST API</span>
          <h2>Submit result</h2>
          <pre>curl -X POST https://sharecrop.example/api/tasks/${task.id}/submissions \\
  -H "Authorization: Bearer &lt;ACCESS_TOKEN&gt;" \\
  -H "Content-Type: application/json" \\
  -d '{"response_json":"{}"}'</pre>
        </article>
        <article class="sub-panel">
          <span class="eyebrow">MCP</span>
          <h2>Session workflow</h2>
          <pre>initialize -> Mcp-Session-Id
sharecrop.reserve_task
sharecrop.get_task_schema
sharecrop.submit_response
sharecrop.accept_submission</pre>
        </article>
      </div>
    </section>
  `;
}

function settingsPage() {
  return `
    <section class="panel split-panel">
      <div>
        <span class="eyebrow">Demo settings</span>
        <h1>Local state and placeholders</h1>
        <p>The demo stores edits only in this browser. Clearing state restores the seeded requester, implementor, reviewer, and agent stories.</p>
      </div>
      <div class="settings-actions">
        <button class="button primary" data-action="reset">Clear demo state</button>
        <a class="button secondary" href="../docs/">Open docs placeholder</a>
      </div>
    </section>
  `;
}

function metricCard(label, value) {
  return `<div class="metric-card"><span>${label}</span><strong>${value}</strong></div>`;
}

function storyCard(title, copy) {
  return `<article class="story-card"><h3>${title}</h3><p>${copy}</p></article>`;
}

function taskTable(tasks, context) {
  if (tasks.length === 0) {
    return `<p class="empty-state">No tasks match this view.</p>`;
  }
  return `
    <div class="task-table" data-context="${context}">
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
      <span>${task.policy}</span>
      <span>${task.reward}</span>
      <span><em>${task.availability}</em></span>
    </button>
  `;
}

function taskActionPanel() {
  const task = selectedTask();
  return `
    <section class="panel split-panel">
      <div>
        <span class="eyebrow">Selected task</span>
        <h2>${escapeHtml(task.title)}</h2>
        <p>${escapeHtml(task.description)}</p>
        <div class="badge-row">
          <span>${task.state}</span>
          <span>${task.visibility}</span>
          <span>${task.reward}</span>
          <span>${task.policy}</span>
        </div>
      </div>
      <div class="sub-panel">
        <h2>Implementor action</h2>
        <textarea data-field="responseText">${escapeHtml(state.responseText)}</textarea>
        <div class="row-actions">
          <button class="button secondary" data-action="reserve">Reserve</button>
          <button class="button secondary" data-action="requestApproval">Request approval</button>
          <button class="button primary" data-action="submit">Submit response</button>
        </div>
      </div>
    </section>
  `;
}

function reservationQueue(task) {
  if (task.reservations.length === 0) {
    return `<p class="empty-state">No reservations for this task.</p>`;
  }
  return task.reservations.map((reservation) => `
    <div class="queue-row">
      <div>
        <strong>${userName(reservation.by)}</strong>
        <span>${reservation.state}</span>
      </div>
      ${reservation.state === "requested" ? `<button class="button primary" data-action="approve" data-user="${reservation.by}">Approve</button>` : ""}
    </div>
  `).join("");
}

function reviewControls() {
  return `
    <div class="review-controls">
      <label for="review-note">Review note<textarea id="review-note" data-field="reviewNote">${escapeHtml(state.reviewNote)}</textarea></label>
      <div class="mini-grid">
        <label>Partial payout<input data-field="partialPayout" value="${escapeHtml(state.partialPayout)}"></label>
        <label>Tip<input data-field="tipAmount" value="${escapeHtml(state.tipAmount)}"></label>
      </div>
      <label class="check-row"><input type="checkbox" data-field="banImplementor" ${state.banImplementor ? "checked" : ""}> Ban implementor from this task</label>
    </div>
  `;
}

function submissionList(task) {
  if (task.submissions.length === 0) {
    return `<p class="empty-state">No submissions for this task.</p>`;
  }
  return task.submissions.map((submission) => `
    <div class="submission-row">
      <div>
        <strong>${userName(submission.by)}</strong>
        <span>${submission.state}</span>
      </div>
      <code>${escapeHtml(submission.response)}</code>
      <div class="row-actions">
        <button class="button secondary" data-action="reject" data-user="${submission.by}">Reject</button>
        <button class="button primary" data-action="accept" data-user="${submission.by}">Accept</button>
      </div>
    </div>
  `).join("");
}

function bindEvents() {
  document.querySelectorAll("[data-field]").forEach((node) => {
    node.addEventListener("input", () => {
      const key = node.dataset.field;
      const value = node.type === "checkbox" ? node.checked : node.value;
      if (key === "userId") {
        setState({ [key]: value, loginOpen: false });
        return;
      }
      setState({ [key]: value });
    });
  });
  document.querySelectorAll("[data-mode]").forEach((node) => {
    node.addEventListener("click", () => setState({ mode: node.dataset.mode }));
  });
  document.querySelectorAll("[data-theme]").forEach((node) => {
    node.addEventListener("click", () => setState({ theme: node.dataset.theme }));
  });
  document.querySelectorAll("[data-page]").forEach((node) => {
    node.addEventListener("click", () => setState({ page: node.dataset.page, loginOpen: false }));
  });
  document.querySelectorAll("[data-task]").forEach((node) => {
    node.addEventListener("click", () => setState({ selectedTaskId: node.dataset.task }));
  });
  document.querySelectorAll("[data-provider]").forEach((node) => {
    node.addEventListener("click", () => setState({ socialProvider: node.dataset.provider, loginOpen: false }));
  });
  document.querySelectorAll("[data-action]").forEach((node) => {
    node.addEventListener("click", () => handleAction(node.dataset.action, node.dataset.user));
  });
}

function handleAction(action, userId) {
  const task = selectedTask();
  if (action === "reset") resetState();
  if (action === "toggleLogin") setState({ loginOpen: !state.loginOpen });
  if (action === "create") createDraftTask();
  if (action === "reserve") reserveTask(task.id);
  if (action === "requestApproval") requestApproval(task.id);
  if (action === "submit") submitResponse(task.id);
  if (action === "approve") approveReservation(task.id, userId);
  if (action === "reject") rejectSubmission(task.id, userId);
  if (action === "accept") acceptSubmission(task.id, userId);
}

function headlineFor(role) {
  if (role === "Requester") return "Coordinate requested work without turning Sharecrop into a task runner.";
  if (role === "Implementor") return "Find eligible tasks, reserve work, and submit structured results.";
  if (role === "Organization reviewer") return "Review organization workflows with clear queues and outcomes.";
  return "Connect local agents through scoped HTTP and MCP instructions.";
}

function copyFor(role) {
  if (role === "Requester") return "The demo separates creation, discovery, review, and integrations so each workflow has a focused place.";
  if (role === "Implementor") return "Discovery shows what is available, what is reserved, and which action can be taken next.";
  if (role === "Organization reviewer") return "Review queues show reservation decisions and submission outcomes without crowding creation controls.";
  return "Integration instructions are kept on their own page so API and MCP usage can be inspected directly.";
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

render();
