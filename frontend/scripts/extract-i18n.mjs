import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import ts from "typescript";

const root = process.cwd();
const zhPath = path.join(root, "messages/zh-CN.json");
const enPath = path.join(root, "messages/en-US.json");
const scanDirs = ["app", "components"];
const extensions = new Set([".ts", ".tsx"]);

function walk(dir, out = []) {
  if (!fs.existsSync(dir)) return out;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const item = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(item, out);
    else if (extensions.has(path.extname(entry.name))) out.push(item);
  }
  return out;
}

function normalize(value) {
  return value.replace(/\s+/g, " ").trim();
}

function isTranslatable(value) {
  if (!/[\u4e00-\u9fff]/.test(value) || value.length < 2 || value.length > 300) return false;
  // Code samples are content, not application chrome. Translating them would
  // corrupt examples and previously produced most of the noisy backlog.
  return !/(?:```|\bfunction\b|=>|<\/?[A-Za-z]|\.gitlab-ci\.yml|\bconst\b|\breturn\b)/.test(value);
}

function templateText(node) {
  let value = node.head.text;
  node.templateSpans.forEach((span, index) => {
    value += `{{value${index + 1}}}${span.literal.text}`;
  });
  return value;
}

function collectStrings() {
  const values = new Set();
  const add = (value) => {
    const normalized = normalize(value);
    if (isTranslatable(normalized)) values.add(normalized);
  };
  for (const dir of scanDirs) {
    for (const file of walk(path.join(root, dir))) {
      const source = fs.readFileSync(file, "utf8");
      const tree = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
      const visit = (node) => {
        if (ts.isJsxText(node) || ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
          add(node.text);
        } else if (ts.isTemplateExpression(node)) {
          add(templateText(node));
        }
        ts.forEachChild(node, visit);
      };
      visit(tree);
    }
  }
  return [...values].sort((a, b) => a.localeCompare(b, "zh-Hans-CN"));
}

function legacyKey(value) {
  return `legacy.${crypto.createHash("sha256").update(value).digest("hex").slice(0, 12)}`;
}

function main() {
  const zh = JSON.parse(fs.readFileSync(zhPath, "utf8"));
  const en = JSON.parse(fs.readFileSync(enPath, "utf8"));
	const previousEnglish = { ...en };
  for (const key of Object.keys(zh)) if (key.startsWith("unmigrated.") || key.startsWith("legacy.")) delete zh[key];
  for (const key of Object.keys(en)) if (key.startsWith("unmigrated.") || key.startsWith("legacy.")) delete en[key];

  const semanticValues = new Set(Object.values(zh));
  for (const value of collectStrings()) {
    if (semanticValues.has(value)) continue;
    const key = legacyKey(value);
    zh[key] = value;
	  en[key] = previousEnglish[key] || "";
  }

  fs.writeFileSync(zhPath, `${JSON.stringify(zh, null, 2)}\n`);
  fs.writeFileSync(enPath, `${JSON.stringify(en, null, 2)}\n`);
  const pending = Object.keys(zh).filter((key) => key.startsWith("legacy.")).length;
	const invalid = Object.entries(zh).filter(([key, source]) => {
	  if (!key.startsWith("legacy.")) return false;
	  const target = en[key];
	  if (typeof target !== "string" || target === "") return true;
	  const sourceVars = [...source.matchAll(/\{\{\w+\}\}/g)].map((match) => match[0]).sort();
	  const targetVars = [...target.matchAll(/\{\{\w+\}\}/g)].map((match) => match[0]).sort();
	  return JSON.stringify(sourceVars) !== JSON.stringify(targetVars);
	});
	if (process.argv.includes("--check") && invalid.length) {
	  console.error(`Invalid or untranslated messages: ${invalid.map(([key]) => key).join(", ")}`);
	  process.exit(1);
	}
  console.log(`Extracted ${pending} clean legacy messages.`);
}

main();
