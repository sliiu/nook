// Mount a scene HTML document into #scene-host.
//
// Scenes are same-document fragments (not iframe srcdoc) so they can call
// the Control bridge. Style tags and body children are adopted; scripts run
// in order after window.Control / window.events are injected.

/**
 * @param {HTMLElement} host
 * @param {string} htmlText
 * @param {{ Control: object, events: object }} api
 */
export async function mountScene(host, htmlText, api) {
  clearHost(host);

  const doc = new DOMParser().parseFromString(htmlText, "text/html");
  const styles = [...doc.querySelectorAll("style")];
  const scripts = [...doc.querySelectorAll("script")];
  const bodyKids = [...doc.body.childNodes].filter((n) => {
    return !(n.nodeName === "SCRIPT" || n.nodeName === "STYLE");
  });

  window.Control = api.Control;
  window.events = api.events;

  const adoptedStyles = [];
  for (const style of styles) {
    const node = document.createElement("style");
    node.textContent = style.textContent;
    node.dataset.sceneStyle = "1";
    document.head.appendChild(node);
    adoptedStyles.push(node);
  }
  host._sceneStyles = adoptedStyles;

  for (const kid of bodyKids) {
    host.appendChild(document.importNode(kid, true));
  }

  for (const script of scripts) {
    await runScript(script);
  }
}

function clearHost(host) {
  if (typeof host._sceneCleanup === "function") host._sceneCleanup();
  host._sceneCleanup = null;
  if (Array.isArray(host._sceneStyles)) {
    for (const node of host._sceneStyles) node.remove();
  }
  host._sceneStyles = [];
  host.replaceChildren();
}

function runScript(source) {
  return new Promise((resolve, reject) => {
    const el = document.createElement("script");
    if (source.type) el.type = source.type;
    if (source.src) {
      el.src = source.src;
      el.onload = () => resolve();
      el.onerror = () => reject(new Error("scene script failed: " + source.src));
      document.body.appendChild(el);
      el.remove();
      return;
    }
    // Inline: classic scripts run immediately; modules need a blob URL.
    if ((source.type || "").includes("module")) {
      const blob = new Blob([source.textContent], { type: "text/javascript" });
      const url = URL.createObjectURL(blob);
      el.type = "module";
      el.src = url;
      el.onload = () => {
        URL.revokeObjectURL(url);
        resolve();
      };
      el.onerror = () => {
        URL.revokeObjectURL(url);
        reject(new Error("scene module failed"));
      };
      document.body.appendChild(el);
      el.remove();
      return;
    }
    el.textContent = source.textContent;
    document.body.appendChild(el);
    el.remove();
    resolve();
  });
}

/**
 * @param {HTMLElement} host
 * @param {string} url
 * @param {{ Control: object, events: object }} api
 */
export async function mountSceneURL(host, url, api) {
  const res = await fetch(url);
  if (!res.ok) throw new Error("failed to load scene " + url);
  const text = await res.text();
  await mountScene(host, text, api);
}

// A catalog scene runs in a sandboxed iframe. The frame has its own origin, so
// it cannot read the shell, and the only bridge it gets is the one below.
const MARKET_CALLS = new Set([
  "getState",
  "setVolume",
  "setMute",
  "setBrightness",
  "noteDrag",
  "noteIdle",
  "setFlyoutSize",
]);
const MARKET_EVENTS = [
  ["stateChanged", "state-changed"],
  ["pillForm", "pill-form"],
];
const MAX_FLYOUT = 1200;

/**
 * @param {HTMLElement} host
 * @param {string} htmlText
 * @param {{ Control: object, events: object }} api
 */
export function mountMarketScene(host, htmlText, api) {
  clearHost(host);

  const frame = document.createElement("iframe");
  frame.setAttribute("sandbox", "allow-scripts");
  frame.setAttribute("referrerpolicy", "no-referrer");
  frame.style.cssText = "border:0;display:block;width:360px;height:360px;background:transparent";
  frame.setAttribute("title", "scene");
  host.appendChild(frame);

  let loads = 0;
  const fail = () => {
    clearHost(host);
    const box = document.createElement("div");
    box.className = "boot";
    box.textContent = "场景已停止";
    host.appendChild(box);
  };

  function onMessage(ev) {
    if (ev.source !== frame.contentWindow) return;
    const msg = ev.data;
    if (!msg || msg.nook !== 1) return;
    if (msg.type === "size") {
      const w = Math.round(Number(msg.w));
      const h = Math.round(Number(msg.h));
      if (w > 0 && h > 0 && w <= MAX_FLYOUT && h <= MAX_FLYOUT) {
        frame.style.width = w + "px";
        frame.style.height = h + "px";
      }
      return;
    }
    if (msg.type !== "call" || !MARKET_CALLS.has(msg.method)) {
      reply(frame, msg.id, null, "not available");
      return;
    }
    const args = Array.isArray(msg.args) ? msg.args : [];
    if (msg.method === "setFlyoutSize") {
      const w = Number(args[0]);
      const h = Number(args[1]);
      if (w < 1 || h < 1 || w > MAX_FLYOUT || h > MAX_FLYOUT) {
        reply(frame, msg.id, null, "size out of range");
        return;
      }
    }
    const fn = api.Control[msg.method];
    Promise.resolve()
      .then(() => fn.apply(api.Control, args))
      .then((value) => reply(frame, msg.id, value, null))
      .catch((err) => reply(frame, msg.id, null, String(err && err.message ? err.message : err)));
  }

  const offs = [];
  for (const [key, name] of MARKET_EVENTS) {
    const off = api.events[key].on((payload) => {
      if (!frame.contentWindow) return;
      frame.contentWindow.postMessage({ nook: 1, type: "event", name, payload }, "*");
    });
    if (typeof off === "function") offs.push(off);
  }

  host._sceneCleanup = () => {
    window.removeEventListener("message", onMessage);
    for (const off of offs) off();
  };
  window.addEventListener("message", onMessage);
  frame.addEventListener("load", () => {
    loads += 1;
    // The first load is the srcdoc. A later one means the page navigated
    // away from the reviewed document, so it does not keep running.
    if (loads > 1) fail();
  });

  const nonce = newNonce();
  frame.srcdoc = marketDocument(htmlText, nonce);
}

function reply(frame, id, value, error) {
  if (!frame.contentWindow) return;
  frame.contentWindow.postMessage(
    error == null ? { nook: 1, type: "resolve", id, value } : { nook: 1, type: "reject", id, error },
    "*",
  );
}

function newNonce() {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return [...bytes].map((b) => b.toString(16).padStart(2, "0")).join("");
}

// marketDocument rebuilds the page so the bridge and the scene's own scripts
// share one nonce, and nothing else can load.
function marketDocument(html, nonce) {
  const doc = new DOMParser().parseFromString(html, "text/html");
  const styles = [...doc.querySelectorAll("style")].map((node) => node.textContent).join("\n");
  const scripts = [...doc.querySelectorAll("script")].filter((node) => !node.src).map((node) => node.textContent);
  for (const node of doc.querySelectorAll("script, style, link, meta, base, iframe, object, embed, frame")) {
    node.remove();
  }
  const csp = [
    "default-src 'none'",
    `script-src 'nonce-${nonce}'`,
    "style-src 'unsafe-inline'",
    "img-src data:",
    "font-src data:",
    "connect-src 'none'",
    "frame-src 'none'",
    "object-src 'none'",
    "base-uri 'none'",
    "form-action 'none'",
  ].join("; ");
  return `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="${csp}">
<style>html,body{margin:0}${escapeStyle(styles)}</style>
</head>
<body>
${doc.body.innerHTML}
<script nonce="${nonce}">${bridgeSource()}<\/script>
${scripts.map((code) => `<script nonce="${nonce}">${escapeScript(code)}<\/script>`).join("\n")}
</body>
</html>`;
}

function bridgeSource() {
  return `(() => {
  const pending = new Map();
  let seq = 0;
  const call = (method, args) => new Promise((resolve, reject) => {
    const id = ++seq;
    pending.set(id, { resolve, reject });
    parent.postMessage({ nook: 1, type: "call", id, method, args }, "*");
  });
  window.Control = {
    getState: () => call("getState", []),
    setVolume: (level) => call("setVolume", [level]),
    setMute: (muted) => call("setMute", [muted]),
    setBrightness: (level) => call("setBrightness", [level]),
    noteDrag: (dragging) => call("noteDrag", [dragging]),
    noteIdle: () => call("noteIdle", []),
    setFlyoutSize: (w, h) => call("setFlyoutSize", [w, h]),
  };
  const listeners = { "state-changed": new Set(), "pill-form": new Set() };
  const listen = (name) => (fn) => {
    listeners[name].add(fn);
    return () => listeners[name].delete(fn);
  };
  window.events = {
    stateChanged: { on: listen("state-changed") },
    pillForm: { on: listen("pill-form") },
  };
  window.addEventListener("message", (ev) => {
    if (ev.source !== parent) return;
    const msg = ev.data;
    if (!msg || msg.nook !== 1) return;
    if (msg.type === "resolve") { const p = pending.get(msg.id); pending.delete(msg.id); if (p) p.resolve(msg.value); }
    else if (msg.type === "reject") { const p = pending.get(msg.id); pending.delete(msg.id); if (p) p.reject(new Error(msg.error || "failed")); }
    else if (msg.type === "event") { for (const fn of listeners[msg.name] || []) fn(msg.payload); }
  });
  const report = () => {
    const el = document.body.firstElementChild;
    const w = el ? el.offsetWidth : 0;
    const h = el ? el.offsetHeight : 0;
    if (w > 0 && h > 0) parent.postMessage({ nook: 1, type: "size", w, h }, "*");
  };
  window.addEventListener("load", report);
  if (typeof ResizeObserver === "function" && document.documentElement) {
    new ResizeObserver(report).observe(document.documentElement);
  }
})();`;
}

function escapeScript(code) {
  return String(code).replace(/<\/script/gi, "<\\/script");
}

function escapeStyle(code) {
  return String(code).replace(/<\/style/gi, "<\\/style");
}
