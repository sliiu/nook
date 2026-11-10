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
