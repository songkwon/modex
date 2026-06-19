import fs from "node:fs";
import path from "node:path";

const root = process.cwd();
const zhPath = path.join(root, "messages/zh-CN.json");
const enPath = path.join(root, "messages/en-US.json");
const scanDirs = ["app", "components", "lib"];
const exts = new Set([".ts", ".tsx", ".js", ".jsx"]);

function walk(dir, out = []) {
  if (!fs.existsSync(dir)) return out;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === "node_modules" || entry.name === ".next" || entry.name === "public") continue;
    const item = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(item, out);
    else if (exts.has(path.extname(entry.name))) out.push(item);
  }
  return out;
}

function scrub(source) {
  return source
    .replace(/\/\*[\s\S]*?\*\//g, " ")
    .replace(/(^|[^:])\/\/.*$/gm, "$1 ")
    .replace(/import\s+[\s\S]*?from\s+["'][^"']+["'];?/g, " ")
    .replace(/className=\{?\s*["'][^"']*["']\s*\}?/g, " ");
}

function normalize(value) {
  return value
    .replace(/\\n/g, " ")
    .replace(/\s+/g, " ")
    .replace(/\{[^{}]*\}/g, "{{value}}")
    .trim();
}

function isUseful(value) {
  if (!/[\u4e00-\u9fff]/.test(value)) return false;
  if (value.length < 2 || value.length > 180) return false;
  return !/^[\s{}()[\].,;:<>/+*\-=`'"|&?!]+$/.test(value);
}

function collectStrings() {
  const strings = new Set();
  const patterns = [
    />([^<>]*[\u4e00-\u9fff][^<>]*)</g,
    /["'`]([^"'`]*[\u4e00-\u9fff][^"'`]*)["'`]/g,
    /title=\{?["'`]([^"'`]*[\u4e00-\u9fff][^"'`]*)["'`]\}?/g,
    /aria-label=\{?["'`]([^"'`]*[\u4e00-\u9fff][^"'`]*)["'`]\}?/g,
    /placeholder=\{?["'`]([^"'`]*[\u4e00-\u9fff][^"'`]*)["'`]\}?/g,
  ];

  for (const dir of scanDirs) {
    for (const file of walk(path.join(root, dir))) {
      const source = scrub(fs.readFileSync(file, "utf8"));
      for (const pattern of patterns) {
        let match;
        while ((match = pattern.exec(source))) {
          const text = normalize(match[1]);
          if (isUseful(text)) strings.add(text);
        }
      }
    }
  }
  return strings;
}

function main() {
  const zh = JSON.parse(fs.readFileSync(zhPath, "utf8"));
  const en = JSON.parse(fs.readFileSync(enPath, "utf8"));

  for (const key of Object.keys(zh)) if (key.startsWith("unmigrated.")) delete zh[key];
  for (const key of Object.keys(en)) if (key.startsWith("unmigrated.")) delete en[key];

  const existing = new Set(Object.values(zh).filter((value) => typeof value === "string"));
  const pending = [...collectStrings()]
    .filter((text) => !existing.has(text))
    .sort((a, b) => a.localeCompare(b, "zh-Hans-CN"));

  pending.forEach((text, index) => {
    const key = `unmigrated.${String(index + 1).padStart(4, "0")}`;
    zh[key] = text;
    en[key] = "";
  });

  fs.writeFileSync(zhPath, `${JSON.stringify(zh, null, 2)}\n`);
  fs.writeFileSync(enPath, `${JSON.stringify(en, null, 2)}\n`);
  console.log(`Extracted ${pending.length} untranslated frontend strings.`);
}

main();
