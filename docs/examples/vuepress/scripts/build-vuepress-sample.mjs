import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";

const root = resolve(process.cwd());
const markdown = readFileSync(resolve(root, "docs/README.md"), "utf8");
const out = resolve(root, "docs/.vuepress/dist/index.html");
mkdirSync(dirname(out), { recursive: true });
writeFileSync(out, `<!doctype html><html><head><meta charset="utf-8"><title>VuePress Guide</title></head><body><main><pre>${markdown.replace(/[<&>]/g, (c) => ({ "<": "&lt;", ">": "&gt;", "&": "&amp;" })[c])}</pre></main></body></html>`);
