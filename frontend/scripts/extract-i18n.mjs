// i18n catalog consistency check.
//
// The app uses explicit t("key") calls (no runtime DOM translation), so message
// keys live directly in the catalogs. This script validates that zh-CN and
// en-US stay in sync; it never rewrites the catalogs. Run with --check in CI.
import fs from "node:fs";
import path from "node:path";

const root = process.cwd();
const zh = JSON.parse(fs.readFileSync(path.join(root, "messages/zh-CN.json"), "utf8"));
const en = JSON.parse(fs.readFileSync(path.join(root, "messages/en-US.json"), "utf8"));

const zhKeys = new Set(Object.keys(zh));
const enKeys = new Set(Object.keys(en));

const missingInEn = [...zhKeys].filter((k) => !enKeys.has(k));
const missingInZh = [...enKeys].filter((k) => !zhKeys.has(k));
const emptyEn = [...zhKeys].filter((k) => enKeys.has(k) && String(en[k]).trim() === "");

// Interpolation placeholders ({{value1}} ...) must match between locales.
const varMismatch = [...zhKeys]
  .filter((k) => enKeys.has(k))
  .filter((k) => {
    const sv = [...String(zh[k]).matchAll(/\{\{\w+\}\}/g)].map((m) => m[0]).sort();
    const tv = [...String(en[k]).matchAll(/\{\{\w+\}\}/g)].map((m) => m[0]).sort();
    return JSON.stringify(sv) !== JSON.stringify(tv);
  });

function collectSourceFiles(dir) {
  if (!fs.existsSync(dir)) return [];
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  return entries.flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return collectSourceFiles(full);
    return /\.[tj]sx?$/.test(entry.name) ? [full] : [];
  });
}

const sourceFiles = ["app", "components", "lib"].flatMap((dir) => collectSourceFiles(path.join(root, dir)));
const referencedKeys = new Set();
const tCallPattern = /(?<![\w$])t\(\s*["'`]([A-Za-z0-9_.-]+)["'`]/g;

for (const file of sourceFiles) {
  const content = fs.readFileSync(file, "utf8");
  for (const match of content.matchAll(tCallPattern)) {
    referencedKeys.add(match[1]);
  }
}

const missingReferenced = [...referencedKeys]
  .filter((k) => !zhKeys.has(k) || !enKeys.has(k))
  .sort();

const problems = [
  ["missing in en-US", missingInEn],
  ["missing in zh-CN", missingInZh],
  ["empty en-US value", emptyEn],
  ["placeholder mismatch", varMismatch],
  ["referenced key missing from catalogs", missingReferenced]
].filter(([, list]) => list.length);

for (const [label, list] of problems) {
  console.error(`${label}: ${list.join(", ")}`);
}

console.log(`Checked ${zhKeys.size} messages and ${referencedKeys.size} referenced keys.`);

if (process.argv.includes("--check") && problems.length) {
  process.exit(1);
}
