// Builds the document a catalog scene actually runs in and checks the fence.
// Run with: node frontend/scene-host.test.mjs

import { readFileSync } from "node:fs";

const source = readFileSync(new URL("./scene-host.js", import.meta.url), "utf8");

class Node {
  constructor(name) {
    this.nodeName = name.toUpperCase();
    this.attrs = {};
    this.children = [];
    this.text = "";
    this.style = {};
  }
  get tagName() {
    return this.nodeName;
  }
  get src() {
    return this.attrs.src || "";
  }
  get textContent() {
    return this.text + this.children.map((child) => child.textContent || "").join("");
  }
  setAttribute(name, value) {
    this.attrs[name] = value;
    if (name === "srcdoc") this.srcdoc = value;
  }
  getAttribute(name) {
    return this.attrs[name] ?? null;
  }
  appendChild(node) {
    this.children.push(node);
  }
  addEventListener() {}
  remove() {
    this.removed = true;
  }
  replaceChildren() {
    this.children = [];
  }
  querySelectorAll(selector) {
    const want = selector.toLowerCase();
    const found = [];
    const walk = (node) => {
      for (const child of node.children) {
        if (child.nodeName === want.toUpperCase()) found.push(child);
        walk(child);
      }
    };
    walk(this);
    return found;
  }
}

for (const proto of [Node.prototype]) {
  if (!proto.querySelectorAll) {
    proto.querySelectorAll = function (selector) {
      const names = selector.split(",").map((part) => part.trim().toLowerCase());
      const found = [];
      const roots = this instanceof Document ? [this.head, this.body] : [this];
      const walk = (node) => {
        for (const child of node.children || []) {
          if (names.includes(child.nodeName.toLowerCase())) found.push(child);
          walk(child);
        }
      };
      for (const root of roots) walk(root);
      return found;
    };
  }
}

class Document {
  constructor() {
    this.head = new Node("head");
    this.body = new Node("body");
  }
  createElement(name) {
    return new Node(name);
  }
  querySelectorAll(selector) {
    const head = this.head.querySelectorAll(selector);
    const body = this.body.querySelectorAll(selector);
    return [...head, ...body];
  }
}

globalThis.document = new Document();
globalThis.window = {
  addEventListener() {},
  removeEventListener() {},
};
crypto.getRandomValues = (bytes) => bytes.fill(7);
globalThis.DOMParser = class {
  parseFromString(html) {
    const doc = new Document();
    const tags = html.match(/<\/?[a-zA-Z]+[^>]*>|[^<]+/g) || [];
    const stack = [doc.body];
    for (const token of tags) {
      const close = /^<\/([a-zA-Z]+)>/.exec(token);
      if (close) {
        if (stack.length > 1) stack.pop();
        continue;
      }
      const open = /^<([a-zA-Z]+)([^>]*)>/.exec(token);
      if (!open) {
        stack.at(-1).text += token;
        continue;
      }
      const node = new Node(open[1]);
      for (const attr of open[2].matchAll(/([a-zA-Z:-]+)\s*=\s*"([^"]*)"/g)) node.attrs[attr[1]] = attr[2];
      const parent = open[1].toLowerCase() === "html" ? null : /^(head|body|title|style|script|meta|link|base)$/i.test(open[1]) && stack.length === 1 ? doc[open[1].toLowerCase() === "body" ? "body" : "head"] ?? doc.head : stack.at(-1);
      if (parent) parent.children.push(node);
      if (!token.endsWith("/>") && !/^(meta|link|img|br|hr)$/i.test(open[1])) stack.push(node);
    }
    return doc;
  }
};

const body = source
  .replace(/^export async function /gm, "async function ")
  .replace(/^export function /gm, "function ");
const fn = new Function(`${body}\nreturn { mountMarketScene };`);
const { mountMarketScene } = fn();

const host = new Node("div");
const page = [
  '<div style="width:200px;height:100px"><span id="v"></span></div>',
  "<script>",
  "Control.getState().then((s) => { v.textContent = s.volume });",
  "</script>",
  '<script src="https://evil.example/x.js"></script>',
].join("\n");

mountMarketScene(host, page, {
  Control: {},
  events: { stateChanged: { on() {} }, pillForm: { on() {} } },
});

const frame = host.children.find((child) => child.nodeName === "IFRAME");
if (!frame) throw new Error("no iframe");
if (frame.attrs.sandbox !== "allow-scripts") throw new Error("sandbox is " + frame.attrs.sandbox);
const doc = frame.srcdoc;
if (!doc.includes("default-src 'none'")) throw new Error("missing CSP");
if (!doc.includes("connect-src 'none'")) throw new Error("connect not blocked");
if (doc.includes("https://evil.example")) throw new Error("remote script survived");
if (!doc.includes("getState")) throw new Error("bridge missing");
if (!doc.includes("s.volume")) throw new Error("scene script missing");
if ((doc.match(/<script/g) || []).length !== 2) throw new Error("unexpected script count: " + doc);
const nonce = /script-src 'nonce-([0-9a-f]+)'/.exec(doc);
if (!nonce) throw new Error("no nonce");
if (!doc.includes(`nonce="${nonce[1]}"`)) throw new Error("script nonce mismatch");

console.log("scene-host sandbox checks passed");
