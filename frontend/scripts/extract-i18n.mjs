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

const problems = [
  ["missing in en-US", missingInEn],
  ["missing in zh-CN", missingInZh],
  ["empty en-US value", emptyEn],
  ["placeholder mismatch", varMismatch]
].filter(([, list]) => list.length);

for (const [label, list] of problems) {
  console.error(`${label}: ${list.join(", ")}`);
}

console.log(`Checked ${zhKeys.size} messages.`);

if (process.argv.includes("--check") && problems.length) {
  process.exit(1);
}
