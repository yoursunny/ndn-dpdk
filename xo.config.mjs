import fs from "node:fs";
import path from "node:path";

import { js, merge, preact, ts, web } from "@yoursunny/xo-config";

/** @type {import("xo").FlatXoConfig} */
const config = [
  js,
  {
    files: [
      "**/*.{ts,tsx}",
    ],
    ...ts,
  },
  {
    files: [
      "js/types/**/*.ts",
    ],
    rules: {
      "tsdoc/syntax": "off", // `@` tags are for ts-json-schema-generator
    },
  },
  {
    files: [
      "sample/benchmark/**/*.tsx",
      "sample/status/**/*.tsx",
    ],
    ...merge(web, preact),
  },
  {
    ignores: [
      "sample/activate",
      "sample/benchmark",
      "sample/status",
    ].filter((d) => !fs.statSync(path.resolve(import.meta.dirname, d, "node_modules"), { throwIfNoEntry: false })?.isDirectory()),
  },
];

export default config;
