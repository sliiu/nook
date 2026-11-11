// Fills the product page before it deploys.
//
// The page must not ask api.github.com from a browser: that endpoint answers
// 403 once an IP has spent its 60 unauthenticated requests an hour. So the
// scene ids and the latest release are written in here, where the request
// carries the workflow token and does not spend anyone's limit.
//
//   GH_TOKEN=... node site/build.mjs site/index.html
//
// It replaces two markers: LISTING, the scene ids in sliiu/nook-scenes, and
// RELEASE, { tag, dmg } for the newest release of this repository.

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

const tree = await get("https://api.github.com/repos/sliiu/nook-scenes/git/trees/main")
const dirs = (tree.tree || []).filter((node) => node.type === "tree" && !node.path.includes("/"))
const present = await Promise.all(
  dirs.map(async (dir) => {
    const res = await fetch(
      `https://api.github.com/repos/sliiu/nook-scenes/contents/${dir.path}/scene.json`,
      { headers },
    )
    return res.ok ? dir.path : ""
  }),
)
const ids = present.filter(Boolean).sort()

const release = await get("https://api.github.com/repos/sliiu/nook/releases/latest")
const dmg = (release.assets || []).find((asset) => asset.name.endsWith(".dmg"))
const info = { tag: release.tag_name || "", dmg: dmg ? dmg.browser_download_url : "" }

let html = readFileSync(page, "utf8")
for (const [marker, value] of [
  ["LISTING", JSON.stringify(ids)],
  ["RELEASE", JSON.stringify(info)],
]) {
  const spot = `/* ${marker} */ null /* /${marker} */`
  if (!html.includes(spot)) {
    console.error(`${page} 里没有 ${spot}`)
    process.exit(1)
  }
  html = html.replace(spot, value)
}

writeFileSync(page, html)
console.log(`scenes: ${ids.join(" ") || "没有"}`)
console.log(`release: ${info.tag} ${info.dmg || "没有 dmg"}`)
