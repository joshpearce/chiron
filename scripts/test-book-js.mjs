#!/usr/bin/env node
// The offline grader in ipad-app/Chiron/Resources/book.js must agree with
// the server's checkers package: a beat graded one way while reading and
// the other way at the chapter boundary is a contradiction the learner
// sees. The cases here mirror server-go/checkers/checkers_test.go.
//
//   node scripts/test-book-js.mjs
import { readFileSync } from "node:fs";
import vm from "node:vm";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const src = readFileSync(join(here, "../ipad-app/Chiron/Resources/book.js"), "utf8");
const noop = () => {};
const sandbox = {
  document: { addEventListener: noop, getElementById: () => null, querySelector: () => null },
  window: { addEventListener: noop, scrollTo: noop, scrollY: 0, webkit: undefined },
  setTimeout, clearTimeout, console,
};
sandbox.globalThis = sandbox;
vm.createContext(sandbox);
vm.runInContext(src, sandbox);

const cases = [
  ["numeric(1)", "50331648", "50331648", true, "exact hit"],
  ["numeric(1)", "50331648", "100663296", false, "double the answer"],
  ["numeric(1)", "50331648", "50331649", true, "within +/-1"],
  ["numeric(0.001)", "0.50349", "0.503", true, "rounded to three places"],
  ["numeric(0.001)", "0.50349", "0.5", false, "the naive symmetry guess"],
  ["numeric(0.01)", "5.5", "-5.5", false, "sign flip"],
  ["exact", "4, 1; 11, 6", "4,1;11,6", true, "separator spacing is not meaning"],
  ["exact", "4, 1; 11, 6", " 4 , 1 ; 11 , 6 ", true, "padded separators"],
  ["exact", "4, 1; 11, 6", "4, 1; 11, 7", false, "one entry wrong"],
  ["exact", "4 x 7", "4 X 7", true, "case-insensitive"],
  ["exact", "4 x 7", "4x7", true, "spacing inside an expression is not meaning"],
  ["exact", "4 x 7", "4 x7", true, "uneven spacing"],
  ["exact", "w|i|d|est_", "w | i | d | est_", true, "spaces around pipes"],
  ["exact", "4 x 7", "47", false, "the operator is meaning"],
  ["exact", "4 x 7", "7 x 4", false, "order is meaning"],
  ["numeric(0.01)", "6", "the answer is 6", true, "prose around a number still parses"],
];

let failed = 0;
for (const [check, expected, given, want, why] of cases) {
  const got = sandbox.gradeMechanical(check, expected, given);
  if (got !== want) {
    failed++;
    console.error(`FAIL ${check} expected=${JSON.stringify(expected)} given=${JSON.stringify(given)} -> ${got}, wanted ${want} (${why})`);
  }
}
// A choice beat grades from its own options, the way checkers.CheckMCQ does.
const options = [
  { text: "scale by sqrt(d_k)", correct: true, explain: "Right." },
  { text: "overflow", misconception: "M7", explain: "Numerics, not gradients." },
];
const choiceCases = [
  [0, true, "Right.", "the correct option"],
  [1, false, "Numerics, not gradients.", "a distractor's own reveal"],
  [5, false, "no option selected", "out of range"],
];
for (const [index, correct, explain, label] of choiceCases) {
  const g = sandbox.gradeChoice(options, index);
  if (g.correct !== correct || g.explain !== explain || g.answer !== "scale by sqrt(d_k)") {
    failed++;
    console.log(`FAIL choice ${label}: ${JSON.stringify(g)}`);
  }
}
const total = cases.length + choiceCases.length;
console.log(`${total - failed}/${total} book.js grading cases pass`);
process.exit(failed ? 1 : 0);
