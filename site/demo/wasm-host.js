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

  // The host no longer resolves "who is acting" itself (actorID/setActor)
  // or looks up users by email (userIDForEmail): the WASM binary now routes
  // every request through the real internal/http mux, which authenticates
  // via the Authorization header WasmXHR forwards below, and looks up users
  // by email directly against its own browser-storage-backed auth store.
  function makeHost() {
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
    };
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
      const authorization = this._headers.Authorization ||
        this._headers.authorization || "";
      const raw = handle(this._method, this._url, body || "", authorization);
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
