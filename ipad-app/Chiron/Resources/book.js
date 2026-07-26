/* Chapter renderer + interaction beats.
 *
 * The native side calls initChapter(payload) once the page reports ready.
 * Beats render as interactive cards; mechanical beats (check: exact/numeric)
 * grade locally so reading works fully detached; llm-checked beats reveal the
 * reference + rubric for self-comparison and the committed response is sent to
 * the bridge for authoritative grading at the next exchange.
 */

let CH = null;
const answered = {};

document.addEventListener("DOMContentLoaded", () => {
  post({ type: "ready" });
});

function post(msg) {
  if (window.webkit?.messageHandlers?.bridge) {
    window.webkit.messageHandlers.bridge.postMessage(msg);
  }
}

function initChapter(payload) {
  CH = payload;
  const root = document.getElementById("chapter");
  root.innerHTML =
    `<h1>${escapeHtml(payload.title)}</h1>` +
    `<div class="chapter-meta">~${payload.minutes} min · work every beat before its reveal</div>` +
    payload.html;

  for (const beat of payload.beats) {
    const holder = root.querySelector(`.beat[data-beat-id="${beat.id}"]`);
    if (holder) renderBeat(holder, beat);
  }
  renderMathIn(root);
  window.scrollTo(0, 0);
}

function renderMathIn(el) {
  if (typeof renderMathInElement === "function") {
    renderMathInElement(el, {
      delimiters: [
        { left: "$$", right: "$$", display: true },
        { left: "$", right: "$", display: false },
      ],
      throwOnError: false,
    });
  }
}

const BEAT_LABELS = {
  predict: "Predict before you read on",
  completion: "Fill in the blanked steps",
  "self-explain": "Explain it yourself first",
  compute: "Compute it by hand",
};

function renderBeat(holder, beat) {
  const mechanical = beat.check && beat.check !== "llm";
  const isCompute = beat.type === "compute";
  holder.innerHTML = `
    <div class="beat-label">${BEAT_LABELS[beat.type] || "Your turn"}</div>
    <div class="beat-prompt">${mdLite(beat.prompt)}</div>
    ${isCompute
      ? `<input type="text" inputmode="decimal" placeholder="your answer">`
      : `<textarea placeholder="write your answer - committing before the reveal is the point"></textarea>`}
    <button class="commit">Commit &amp; reveal</button>
    <div class="reveal" hidden></div>`;
  renderMathIn(holder);

  const input = holder.querySelector("input, textarea");
  const btn = holder.querySelector("button.commit");
  btn.addEventListener("click", () => {
    const response = input.value.trim();
    if (!response) { input.focus(); return; }
    input.disabled = true;
    btn.remove();
    const reveal = holder.querySelector(".reveal");
    reveal.hidden = false;

    if (mechanical) {
      const ok = gradeMechanical(beat.check, beat.answer, response);
      reveal.innerHTML =
        `<div class="${ok ? "verdict-ok" : "verdict-bad"}">${ok ? "Correct." : "Not quite."}</div>` +
        `<p><strong>Answer:</strong> ${mdLite(String(beat.answer))}</p>`;
      finishBeat(holder, beat, response, { mechanicalVerdict: ok ? "pass" : "fail" });
    } else {
      reveal.innerHTML =
        `<p><strong>Reference:</strong></p>${mdLite(beat.answer || "")}` +
        (beat.rubric ? `<p><strong>What a full answer contains:</strong></p>${mdLite(beat.rubric)}` : "") +
        `<p>Honest self-check - did you have it?</p>
         <button class="secondary" data-v="pass">I had it</button>
         <button class="secondary" data-v="partial">Partially</button>
         <button class="secondary" data-v="fail">I did not</button>`;
      reveal.querySelectorAll("button").forEach((b) =>
        b.addEventListener("click", () => {
          reveal.querySelectorAll("button").forEach((x) => (x.disabled = true));
          b.style.opacity = "1";
          finishBeat(holder, beat, response, { selfVerdict: b.dataset.v });
        })
      );
    }
    renderMathIn(reveal);
  });
}

function finishBeat(holder, beat, response, extra) {
  holder.classList.add("done");
  answered[beat.id] = true;
  post({ type: "beat", beatId: beat.id, response, ...extra });
}

/* Mirrors server/checkers.py for the detached path. */
function gradeMechanical(check, expected, given) {
  if (check === "exact") return norm(given) === norm(expected);
  const m = /^numeric\(([\d.eE+-]+)\)$/.exec(check.trim());
  if (m) {
    const tol = parseFloat(m[1]);
    const exp = parseNumber(expected), got = parseNumber(given);
    if (exp === null || got === null) return false;
    // Absolute tolerance - must match server/checkers.py exactly, or a beat
    // grades one way offline and the other way at the boundary.
    return Math.abs(got - exp) <= tol + Math.abs(exp) * 1e-9;
  }
  return norm(given) === norm(expected);
}

function parseNumber(text) {
  const m = String(text).replace(/,/g, "").match(/-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?(?:\s*\/\s*-?\d+(?:\.\d+)?)?/);
  if (!m) return null;
  const t = m[0];
  if (t.includes("/")) {
    const [a, b] = t.split("/").map(parseFloat);
    return b ? a / b : null;
  }
  return parseFloat(t);
}

function norm(s) {
  // Must match server/checkers.py `_norm`: multi-value answers like
  // "4, 1; 11, 6" are the common exact-match case, and spacing around
  // separators carries no meaning. Diverging here means a beat grades one way
  // offline and the other way at the chapter boundary.
  return String(s).replace(/\s+/g, " ").trim().toLowerCase()
    .replace(/\s*([,;:])\s*/g, "$1");
}

/* Tiny markdown: paragraphs, bold, italics, inline code. The heavy conversion
 * happened server-side; beat prompts/answers are close to plain text. */
function mdLite(text) {
  return String(text)
    .split(/\n{2,}/)
    .map((p) =>
      "<p>" +
      escapeHtml(p)
        .replace(/`([^`]+)`/g, "<code>$1</code>")
        .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
        .replace(/\n/g, "<br>") +
      "</p>")
    .join("");
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"]/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
}
