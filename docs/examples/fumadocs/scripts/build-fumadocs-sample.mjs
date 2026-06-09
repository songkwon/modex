import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";

const root = resolve(process.cwd());
const mdx = readFileSync(resolve(root, "content/docs/index.mdx"), "utf8");
const out = resolve(root, "out/index.html");
mkdirSync(dirname(out), { recursive: true });
writeFileSync(out, `<!doctype html><html><head><meta charset="utf-8"><title>Fumadocs Kit</title></head><body><main><pre>${mdx.replace(/[<&>]/g, (c) => ({ "<": "&lt;", ">": "&gt;", "&": "&amp;" })[c])}</pre></main></body></html>`);
