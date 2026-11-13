// Fills the product page before it deploys.
//
// The page must not ask api.github.com from a browser: that endpoint answers
// 403 once an IP has spent its 60 unauthenticated requests an hour. So the
// scene ids and the latest release are written in here, where the request
// carries the workflow token and does not spend anyone's limit.
//
//   GH_TOKEN=... node site/build.mjs site/index.html
//
// It replaces one marker: RELEASE, { tag, dmg, downloads } for the newest
// release of this repository. The scene cards come from scenes.json in
// sliiu/nook-scenes, which the page fetches itself.

import { readFileSync, writeFileSync } from "node:fs"

const page = process.argv[2]
if (!page) {
  console.error("usage: node site/build.mjs <page>")
  process.exit(1)
}

const token = process.env.GH_TOKEN
const headers = {
  Accept: "application/vnd.github+json",
  "User-Agent": "nook-site",
  ...(token ? { Authorization: `Bearer ${token}` } : {}),
}

async function get(url) {
  const res = await fetch(url, { headers })
  if (!res.ok) throw new Error(`${res.status} ${url}`)
  return res.json()
}

const release = await get("https://api.github.com/repos/sliiu/nook/releases/latest")
const dmg = (release.assets || []).find((asset) => asset.name.endsWith(".dmg"))
const info = {
  tag: release.tag_name || "",
  dmg: dmg ? dmg.browser_download_url : "",
  downloads: dmg ? dmg.download_count || 0 : 0,
}

let html = readFileSync(page, "utf8")
for (const [marker, value] of [["RELEASE", JSON.stringify(info)]]) {
  const spot = `/* ${marker} */ null /* /${marker} */`
  if (!html.includes(spot)) {
    console.error(`${page} 里没有 ${spot}`)
    process.exit(1)
  }
  html = html.replace(spot, value)
}

writeFileSync(page, html)
console.log(`release: ${info.tag} ${info.downloads} 次 ${info.dmg || "没有 dmg"}`)
