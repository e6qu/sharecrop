(function () {
  "use strict";

  const STORAGE_PREFIX = "sharecrop-wasm:";
  const COUNTER_PREFIX = "sharecrop-wasm-counter:";

  function requiredFunction(name) {
    const value = window[name];
    if (typeof value !== "function") {
      throw new Error(name + " export is required");
    }
    return value;
  }

  function parseResponse(raw, label) {
    if (typeof raw !== "string" || raw.trim() === "") {
      throw new Error(label + " returned an empty response");
    }
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      throw new Error(label + " returned a non-record response");
    }
    return parsed;
  }

  function storageKey(key) {
    if (typeof key !== "string" || key.trim() === "") {
      throw new Error("WASM host storage key is required");
    }
    return STORAGE_PREFIX + key;
  }

  function makeHost() {
    let actor = window.__sharecropWasmActor || "user-mara";
    return {
      storageHas(key) {
        return window.localStorage.getItem(storageKey(key)) !== null;
      },
      storageGet(key) {
        const value = window.localStorage.getItem(storageKey(key));
        if (value === null) {
          throw new Error("WASM host storage key was not found: " + key);
        }
        return value;
      },
      storagePut(key, value) {
        if (typeof value !== "string") {
          throw new Error("WASM host storage value must be a string");
        }
        window.localStorage.setItem(storageKey(key), value);
        return true;
      },
      now() {
        return new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
      },
      actorID() {
        return actor;
      },
      nextID(kind) {
        if (typeof kind !== "string" || kind.trim() === "") {
          throw new Error("WASM host id kind is required");
        }
        const key = COUNTER_PREFIX + kind;
        const current = Number(window.localStorage.getItem(key) || "0");
        if (!Number.isInteger(current) || current < 0) {
          throw new Error("WASM host id counter is invalid: " + kind);
        }
        const next = current + 1;
        window.localStorage.setItem(key, String(next));
        return kind + "-" + next;
      },
      userIDForEmail(email) {
        const users = {
          "mara@sharecrop.demo": "user-mara",
          "jules@sharecrop.demo": "user-jules",
          "ren@sharecrop.demo": "user-ren",
          "tala@sharecrop.demo": "user-tala",
          "sol@sharecrop.demo": "user-sol",
        };
        return users[email] || "";
      },
      setActor(nextActor) {
        if (typeof nextActor !== "string" || nextActor.trim() === "") {
          throw new Error("WASM host actor is required");
        }
        actor = nextActor;
      },
    };
  }

  function putSeed(key, value) {
    window.localStorage.setItem(storageKey(key), JSON.stringify(value));
  }

  function seedStorage() {
    const users = [
      { id: "user-mara", email: "mara@sharecrop.demo", status: "active" },
      { id: "user-jules", email: "jules@sharecrop.demo", status: "active" },
      { id: "user-ren", email: "ren@sharecrop.demo", status: "active" },
      { id: "user-tala", email: "tala@sharecrop.demo", status: "active" },
      { id: "user-sol", email: "sol@sharecrop.demo", status: "active" },
    ];
    users.forEach((user) => {
      putSeed("user:" + user.id, user);
      putSeed("user_email:" + user.email, user.id);
    });
    putSeed("user:index", users.map((user) => user.id));
    putSeed("platform_admin:user-mara", {
      user_id: "user-mara",
      source: "bootstrap",
      state: "active",
      created_at: "2026-07-01T00:00:00Z",
    });
    putSeed("platform_admin:index", ["user-mara"]);
    putSeed("organization:org-field", {
      id: "org-field",
      name: "Field Operations",
      created_by: "user-mara",
    });
    putSeed("organization:index", ["org-field"]);
    putSeed("organization_member:org-field:user-mara", {
      id: "organization-member-mara",
      organization_id: "org-field",
      user_id: "user-mara",
      status: "active",
      roles: ["owner", "admin", "reviewer"],
    });
    putSeed("organization_member:index:org-field", ["user-mara"]);
    putSeed("ledger:seed-org-balance", {
      id: "seed-org-balance",
      owner_kind: "organization",
      owner_id: "org-field",
      kind: "organization_funding",
      amount: 7200,
      task_id: "",
    });
    putSeed("ledger:index:organization:org-field", ["seed-org-balance"]);
    putSeed("ledger:seed-signup-grant", {
      id: "seed-signup-grant",
      owner_kind: "user",
      owner_id: "user-mara",
      kind: "signup_grant",
      amount: 1150,
      task_id: "",
    });
    putSeed("ledger:index:user:user-mara", ["seed-signup-grant"]);
    const tasks = [
      {
        id: "task-invoices",
        title: "Extract line items from 6 vendor invoices",
        description: "OCR'd text of 6 vendor invoices.",
        task_type: "qa_testing",
        payload_json:
          '{"vendor":"Birch Supply Co","fields":["invoice_id","total","due_date"]}',
        response_schema_json:
          '{"kind":"object","fields":[{"name":"invoices","presence":"required","schema":{"kind":"array"}}]}',
      },
      {
        id: "task-support",
        title: "Classify 8 support tickets by category",
        description:
          "Classify support tickets into billing, bug, account, feature_request, or other.",
        task_type: "qa_testing",
        payload_json: '{"tickets":["billing question","bug report"]}',
        response_schema_json:
          '{"kind":"object","fields":[{"name":"labels","presence":"required","schema":{"kind":"array"}}]}',
      },
      {
        id: "task-release-notes",
        title: "Write release notes for 5 changelog entries",
        description: "Convert changelog entries into concise release notes.",
        task_type: "general",
        payload_json: '{"entries":["Added WASM demo backend"]}',
        response_schema_json: '{"kind":"freeform"}',
      },
      {
        id: "task-ledger-review",
        title: "Verify 10 ledger transfers for fraud signals",
        description: "Review ledger movements and flag suspicious transfers.",
        task_type: "code_review",
        payload_json: '{"transfers":10}',
        response_schema_json: '{"kind":"freeform"}',
      },
    ];
    tasks.forEach((task) => {
      putSeed("task:" + task.id, {
        id: task.id,
        created_by: task.id === "task-ledger-review"
          ? "user-mara"
          : "user-jules",
        owner_kind: "user",
        owner_id: task.id === "task-ledger-review" ? "user-mara" : "user-jules",
        title: task.title,
        state: "open",
        description: task.description,
        task_type: task.task_type,
        reward_kind: "credit",
        reward_collectible_ids: [],
        reward_collectible_count: 0,
        reward_credit_amount: 25,
        participation_policy: "reservation_required",
        reservation_expiry_hours: 48,
        assignee_scope: "user",
        visibility_kind: "public",
        visibility_id: "",
        series_kind: "standalone",
        series_position: 0,
        series_id: "",
        reference_url: "",
        response_schema_json: task.response_schema_json,
        payload_kind: "json",
        payload_json: task.payload_json,
        escrow_amount: 25,
        funded_organization_id: "",
      });
      putSeed("attachments:task:" + task.id, []);
    });
    putSeed("task:index", tasks.map((task) => task.id));
  }

  function resetStorage() {
    const remove = [];
    for (let index = 0; index < window.localStorage.length; index += 1) {
      const key = window.localStorage.key(index);
      if (
        key &&
        (key.startsWith(STORAGE_PREFIX) || key.startsWith(COUNTER_PREFIX))
      ) {
        remove.push(key);
      }
    }
    remove.forEach((key) => window.localStorage.removeItem(key));
  }

  async function loadScript(src) {
    await new Promise((resolve, reject) => {
      const script = document.createElement("script");
      script.src = src;
      script.onload = resolve;
      script.onerror = () => reject(new Error("failed to load " + src));
      document.head.appendChild(script);
    });
  }

  async function install(options) {
    const wasmExecURL = options && options.wasmExecURL;
    const wasmURL = options && options.wasmURL;
    if (!wasmExecURL || !wasmURL) {
      throw new Error("WASM host requires wasmExecURL and wasmURL");
    }
    resetStorage();
    seedStorage();
    await loadScript(wasmExecURL);
    if (typeof window.Go !== "function") {
      throw new Error("Go WASM runtime constructor is required");
    }
    const go = new window.Go();
    const bytes = await fetch(wasmURL).then((response) => {
      if (!response.ok) {
        throw new Error("failed to load " + wasmURL);
      }
      return response.arrayBuffer();
    });
    const result = await WebAssembly.instantiate(bytes, go.importObject);
    go.run(result.instance);
    await new Promise((resolve) => setTimeout(resolve, 0));

    const configure = requiredFunction("sharecropConfigureHost");
    const handle = requiredFunction("sharecropHandleRequest");
    const host = makeHost();
    const configured = parseResponse(configure(host), "sharecropConfigureHost");
    if (configured.status !== "configured") {
      throw new Error(configured.error || "WASM host configuration failed");
    }

    const RealXHR = window.XMLHttpRequest;
    function WasmXHR() {
      this._listeners = {};
      this._real = null;
      this.readyState = 0;
      this.status = 0;
      this.statusText = "";
      this.responseText = "";
      this.response = "";
      this.responseType = "";
      this._headers = {};
    }
    WasmXHR.prototype.open = function (method, url) {
      this._method = (method || "GET").toUpperCase();
      this._url = url;
      this._intercept = new URL(url, window.location.href).pathname
        .startsWith("/api/");
      if (!this._intercept) {
        this._real = new RealXHR();
        this._real.open.apply(this._real, arguments);
      }
    };
    WasmXHR.prototype.setRequestHeader = function (key, value) {
      if (this._real) this._real.setRequestHeader(key, value);
      else this._headers[key] = value;
    };
    WasmXHR.prototype.getAllResponseHeaders = function () {
      return this._real
        ? this._real.getAllResponseHeaders()
        : "content-type: application/json\r\n";
    };
    WasmXHR.prototype.getResponseHeader = function (name) {
      if (this._real) return this._real.getResponseHeader(name);
      return name.toLowerCase() === "content-type" ? "application/json" : null;
    };
    WasmXHR.prototype.addEventListener = function (type, callback) {
      (this._listeners[type] = this._listeners[type] || []).push(callback);
      if (this._real) this._real.addEventListener(type, callback);
    };
    WasmXHR.prototype.removeEventListener = function (type, callback) {
      if (this._real) this._real.removeEventListener(type, callback);
    };
    WasmXHR.prototype.abort = function () {
      if (this._real) this._real.abort();
    };
    WasmXHR.prototype._emit = function (type) {
      const event = { type: type, target: this, currentTarget: this };
      if (typeof this["on" + type] === "function") this["on" + type](event);
      (this._listeners[type] || []).forEach((callback) =>
        callback.call(this, event)
      );
    };
    WasmXHR.prototype.send = function (body) {
      if (this._real) {
        return this._real.send(body);
      }
      const auth = this._headers.Authorization || this._headers.authorization ||
        "";
      const prefix = "Bearer wasm-access-";
      if (typeof auth === "string" && auth.startsWith(prefix)) {
        host.setActor(auth.slice(prefix.length));
      }
      const raw = handle(this._method, this._url, body || "");
      const result = parseResponse(raw, "sharecropHandleRequest");
      this.status = result.status;
      this.statusText = result.status >= 400 ? "Error" : "OK";
      this.responseText = result.body ||
        (result.error ? JSON.stringify({ error: result.error }) : "");
      this.response = this.responseText;
      this.readyState = 4;
      this._emit("readystatechange");
      this._emit("load");
      this._emit("loadend");
    };
    window.XMLHttpRequest = WasmXHR;
    window.__sharecropWasmHost = host;
  }

  window.SharecropWasmDemo = { install, resetStorage };
})();
