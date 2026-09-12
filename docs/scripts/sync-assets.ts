// Keeps served artifacts in lockstep with their repo sources:
//   packaging/install.sh  ->  public/install.sh   (the curl one-liner target)
// Runs before every build so the installer page and the raw file at
// https://fyrwall.docs.potenfyr.in/install.sh always match the tree.
import { copyFileSync, mkdirSync, readFileSync, statSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, "..");
const repoRoot = resolve(root, "..");

mkdirSync(resolve(root, "public"), { recursive: true });
const src = resolve(repoRoot, "packaging", "install.sh");
const dst = resolve(root, "public", "install.sh");
copyFileSync(src, dst);
if (statSync(src).size < 200 || !readFileSync(src, "utf8").includes("fyrwall")) {
  throw new Error("packaging/install.sh looks wrong; refusing to publish it");
}
console.log("[sync-assets] packaging/install.sh -> docs/public/install.sh");
