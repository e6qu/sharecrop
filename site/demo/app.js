const storageKey = "sharecrop-demo-state-v1";

const seedState = {
  mode: "light",
  theme: "corporate",
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

function setState(patch) {
  state = { ...state, ...patch };
  saveState();
  render();
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
        <aside class="sidebar">${sidebar()}</aside>
        <section class="content">
          ${heroPanel()}
          ${workflowGrid()}
          ${taskDetail()}
          ${apiPanel()}
        </section>
      </section>
    </main>
  `;
  bindEvents();
}

function topbar() {
  return `
    <nav class="topbar">
      <a class="brand" href="../">Sharecrop</a>
      <div class="topbar-actions">
        <a class="button secondary" href="../docs/">Docs</a>
        <button class="button ghost" data-action="reset">Clear demo state</button>
      </div>
    </nav>
  `;
}

function sidebar() {
  const user = selectedUser();
  return `
    <section class="panel compact">
      <span class="eyebrow">Demo session</span>
      <label for="demo-user">User</label>
      <select id="demo-user" data-field="userId">
        ${users.map((item) => `<option value="${item.id}" ${item.id === state.userId ? "selected" : ""}>${item.name} - ${item.role}</option>`).join("")}
      </select>
      <div class="metric">
        <span>Balance</span>
        <strong>${user.balance} credits</strong>
      </div>
      <div class="mock-login">
        ${providers.map((provider) => `<button class="button secondary" data-provider="${provider}">${provider}</button>`).join("")}
      </div>
      <p class="hint">${state.socialProvider ? `${state.socialProvider} sign-in is a demo placeholder.` : "Provider buttons are visual placeholders."}</p>
    </section>
    <section class="panel compact">
      <span class="eyebrow">Theme</span>
      <label>Mode</label>
      <div class="segmented">
        ${modeButton("light", "Light")}
        ${modeButton("dark", "Dark")}
      </div>
      <label>Visual theme</label>
      <div class="theme-list">
        ${themeButton("corporate", "Corporate", "Habbo-style work rooms")}
        ${themeButton("rustic", "Rustic", "Farm workflow board")}
        ${themeButton("blocky", "Blocky", "Voxel task map")}
        ${themeButton("showcase", "Showcase", "Modern startup page")}
      </div>
    </section>
    <section class="panel compact">
      <span class="eyebrow">Create task</span>
      <label for="draft-title">Title</label>
      <input id="draft-title" data-field="draftTitle" value="${escapeHtml(state.draftTitle)}">
      <label for="draft-reward">Reward</label>
      <input id="draft-reward" data-field="draftReward" value="${escapeHtml(state.draftReward)}">
      <label for="draft-policy">Access</label>
      <select id="draft-policy" data-field="draftPolicy">
        <option value="reservation" ${state.draftPolicy === "reservation" ? "selected" : ""}>Reservation required</option>
        <option value="approval" ${state.draftPolicy === "approval" ? "selected" : ""}>Requester approval required</option>
      </select>
      <button class="button primary wide" data-action="create">Add demo task</button>
    </section>
  `;
}

function modeButton(value, label) {
  return `<button class="button ${state.mode === value ? "primary" : "secondary"}" data-mode="${value}">${label}</button>`;
}

function themeButton(value, label, description) {
  return `
    <button class="theme-choice ${state.theme === value ? "selected" : ""}" data-theme="${value}">
      <strong>${label}</strong>
      <span>${description}</span>
    </button>
  `;
}

function heroPanel() {
  const user = selectedUser();
  return `
    <section class="hero-panel">
      <div>
        <span class="eyebrow">${user.role}</span>
        <h1>${headlineFor(user.role)}</h1>
        <p>${copyFor(user.role)}</p>
      </div>
      <div class="visual-stack" aria-hidden="true">
        <span></span><span></span><span></span><span></span>
      </div>
    </section>
  `;
}

function workflowGrid() {
  return `
    <section class="grid three">
      <article class="panel">
        <span class="eyebrow">Discover</span>
        <h2>Available work</h2>
        <label class="check-row">
          <input type="checkbox" data-field="includeReserved" ${state.includeReserved ? "checked" : ""}>
          Include reserved
        </label>
        <div class="task-list">
          ${visibleTasks().map(taskRow).join("")}
        </div>
      </article>
      <article class="panel">
        <span class="eyebrow">Reservations</span>
        <h2>Approval queue</h2>
        ${reservationQueue()}
      </article>
      <article class="panel">
        <span class="eyebrow">Review</span>
        <h2>Submission outcomes</h2>
        ${reviewPanel()}
      </article>
    </section>
  `;
}

function taskRow(task) {
  return `
    <button class="task-row ${task.id === state.selectedTaskId ? "selected" : ""}" data-task="${task.id}">
      <strong>${escapeHtml(task.title)}</strong>
      <span>${task.reward} - ${task.policy}</span>
      <em>${task.availability}</em>
    </button>
  `;
}

function reservationQueue() {
  const task = selectedTask();
  if (task.reservations.length === 0) {
    return `<p class="hint">No reservations for the selected task.</p>`;
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

function reviewPanel() {
  const task = selectedTask();
  return `
    <label for="review-note">Review note</label>
    <textarea id="review-note" data-field="reviewNote">${escapeHtml(state.reviewNote)}</textarea>
    <div class="mini-grid">
      <label>Partial payout<input data-field="partialPayout" value="${escapeHtml(state.partialPayout)}"></label>
      <label>Tip<input data-field="tipAmount" value="${escapeHtml(state.tipAmount)}"></label>
    </div>
    <label class="check-row"><input type="checkbox" data-field="banImplementor" ${state.banImplementor ? "checked" : ""}> Ban implementor from this task</label>
    <div class="submission-list">
      ${task.submissions.length === 0 ? `<p class="hint">No submissions yet.</p>` : task.submissions.map(submissionRow).join("")}
    </div>
  `;
}

function submissionRow(submission) {
  return `
    <div class="queue-row vertical">
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
  `;
}

function taskDetail() {
  const task = selectedTask();
  return `
    <section class="panel detail-panel">
      <div>
        <span class="eyebrow">Task</span>
        <h2>${escapeHtml(task.title)}</h2>
        <p>${escapeHtml(task.description)}</p>
        <div class="badge-row">
          <span>${task.state}</span>
          <span>${task.visibility}</span>
          <span>${task.reward}</span>
          <span>${task.policy}</span>
        </div>
      </div>
      <div class="action-card">
        <h3>Implementor action</h3>
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

function apiPanel() {
  const task = selectedTask();
  return `
    <section class="grid two">
      <article class="panel">
        <span class="eyebrow">REST API</span>
        <h2>Submit result</h2>
        <pre>curl -X POST https://sharecrop.example/api/tasks/${task.id}/submissions \\
  -H "Authorization: Bearer &lt;ACCESS_TOKEN&gt;" \\
  -H "Content-Type: application/json" \\
  -d '{"response_json":"{}"}'</pre>
      </article>
      <article class="panel">
        <span class="eyebrow">MCP</span>
        <h2>Session tool call</h2>
        <pre>sharecrop.reserve_task
sharecrop.get_task_schema
sharecrop.submit_response
sharecrop.accept_submission</pre>
      </article>
    </section>
  `;
}

function bindEvents() {
  document.querySelectorAll("[data-field]").forEach((node) => {
    node.addEventListener("input", () => {
      const key = node.dataset.field;
      const value = node.type === "checkbox" ? node.checked : node.value;
      setState({ [key]: value });
    });
  });
  document.querySelectorAll("[data-mode]").forEach((node) => {
    node.addEventListener("click", () => setState({ mode: node.dataset.mode }));
  });
  document.querySelectorAll("[data-theme]").forEach((node) => {
    node.addEventListener("click", () => setState({ theme: node.dataset.theme }));
  });
  document.querySelectorAll("[data-task]").forEach((node) => {
    node.addEventListener("click", () => setState({ selectedTaskId: node.dataset.task }));
  });
  document.querySelectorAll("[data-provider]").forEach((node) => {
    node.addEventListener("click", () => setState({ socialProvider: node.dataset.provider }));
  });
  document.querySelectorAll("[data-action]").forEach((node) => {
    node.addEventListener("click", () => handleAction(node.dataset.action, node.dataset.user));
  });
}

function handleAction(action, userId) {
  const task = selectedTask();
  if (action === "reset") resetState();
  if (action === "create") createDraftTask();
  if (action === "reserve") reserveTask(task.id);
  if (action === "requestApproval") requestApproval(task.id);
  if (action === "submit") submitResponse(task.id);
  if (action === "approve") approveReservation(task.id, userId);
  if (action === "reject") rejectSubmission(task.id, userId);
  if (action === "accept") acceptSubmission(task.id, userId);
}

function headlineFor(role) {
  if (role === "Requester") return "Design tasks, reserve implementors, and pay fair rewards.";
  if (role === "Implementor") return "Find work, reserve it, and submit structured results.";
  if (role === "Organization reviewer") return "Review submissions and manage team-facing work.";
  return "Connect local agents through scoped MCP credentials.";
}

function copyFor(role) {
  if (role === "Requester") return "The demo keeps every change in this browser so task creation, review notes, partial payouts, and tips can be tested without a backend.";
  if (role === "Implementor") return "Discovery, reservation, approval, schema inspection, and submission are shown as the implementor sees them.";
  if (role === "Organization reviewer") return "Organization review stories include scoped visibility, approval queues, and reviewer outcomes.";
  return "Agent stories include REST examples, MCP tool names, and session-based workflow hints.";
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
