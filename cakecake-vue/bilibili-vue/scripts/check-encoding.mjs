#!/usr/bin/env node
/**
 * Fail CI / pre-commit when Vue/TS sources contain mojibake placeholders.
 * Usage: node scripts/check-encoding.mjs
 */
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const ROOT = new URL("..", import.meta.url).pathname.replace(/^\/([A-Z]:)/, "$1");
const SCAN_DIRS = ["src/pages/minibili", "src/i18n", "src/components/minibili"];
const EXT = new Set([".vue", ".ts", ".js"]);
const GARBLED_LINE = /\?{4,}|\uFFFD/;
const GARBLED_STRING = /["'`][^"'`]*\?{3,}[^"'`]*["'`]/;

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    const st = statSync(p);
    if (st.isDirectory()) {
      if (name === "node_modules" || name === "dist") continue;
      walk(p, out);
    } else if (EXT.has(name.slice(name.lastIndexOf(".")))) {
      out.push(p);
    }
  }
  return out;
}

const errors = [];

for (const dir of SCAN_DIRS) {
  const base = join(ROOT, dir);
  for (const file of walk(base)) {
    let text;
    try {
      text = readFileSync(file, "utf8");
    } catch (e) {
      errors.push(`${relative(ROOT, file)}: not valid UTF-8 (${e.message})`);
      continue;
    }
    const lines = text.split(/\r?\n/);
    lines.forEach((line, i) => {
      if (line.includes("formatFavoritedMD") || line.includes("http")) return;
      if (/aria-hidden="true">\?<\/span/.test(line)) return;
      if (/>\s*×\s*<\/button/.test(line)) return;
      if (GARBLED_LINE.test(line) || GARBLED_STRING.test(line)) {
        errors.push(`${relative(ROOT, file)}:${i + 1}: suspected mojibake`);
      }
    });
  }
}

if (errors.length) {
  console.error("Encoding check failed:\n" + errors.slice(0, 40).join("\n"));
  if (errors.length > 40) {
    console.error(`... and ${errors.length - 40} more`);
  }
  process.exit(1);
}

console.log("Encoding check passed.");
