import { useState } from "preact/hooks";
import { theme, roundUp, usd, PRESETS } from "./store";

const parse = (s: string) => {
  const v = parseFloat(s);
  return Number.isFinite(v) && v >= 0 ? v : 0;
};

export function App() {
  // local component state
  const [bill, setBill] = useState(0);
  const [tipPercent, setTipPercent] = useState(18);
  const [people, setPeople] = useState(1);

  // shared signals
  const dark = theme.value === "dark";
  const round = roundUp.value;

  // derived
  const tipAmount = (bill * tipPercent) / 100;
  const total = bill + tipAmount;
  const perPersonRaw = people > 0 ? total / people : 0;
  const perPerson = round ? Math.ceil(perPersonRaw) : perPersonRaw;
  const totalCollected = round ? perPerson * people : total;
  const roundingExtra = Math.max(0, totalCollected - total);
  const effTip = bill > 0 ? ((totalCollected - bill) / bill) * 100 : tipPercent;

  return (
    <main class="bs-app" data-theme={theme.value}>
      <header class="bs-header">
        <h1 class="bs-title">Bill Splitter</h1>
        <div class="bs-header-actions">
          <button class="bs-toggle" aria-pressed={round} onClick={() => (roundUp.value = !roundUp.value)}>
            Round up
          </button>
          <button
            class="bs-toggle"
            aria-pressed={dark}
            onClick={() => (theme.value = dark ? "light" : "dark")}
          >
            Dark
          </button>
        </div>
      </header>

      <section class="bs-card bs-inputs">
        <label class="bs-field">
          <span class="bs-label">Bill amount</span>
          <div class="bs-input-wrap">
            <span class="bs-prefix">$</span>
            <input
              class="bs-input"
              type="number"
              min="0"
              step="0.01"
              value={bill || ""}
              onInput={(e) => setBill(parse((e.target as HTMLInputElement).value))}
            />
          </div>
        </label>

        <div class="bs-field">
          <span class="bs-label">Tip</span>
          <div class="bs-presets">
            {PRESETS.map((p) => (
              <button
                key={p}
                class={"bs-preset" + (tipPercent === p ? " bs-preset--active" : "")}
                onClick={() => setTipPercent(p)}
              >
                {p}%
              </button>
            ))}
            <input
              class="bs-preset-custom"
              type="number"
              min="0"
              placeholder="Custom %"
              value={tipPercent || ""}
              onInput={(e) => setTipPercent(parse((e.target as HTMLInputElement).value))}
            />
          </div>
        </div>

        <div class="bs-field">
          <span class="bs-label">People</span>
          <div class="bs-stepper">
            <button
              class="bs-step"
              aria-label="Fewer people"
              disabled={people <= 1}
              onClick={() => setPeople((n) => Math.max(1, n - 1))}
            >
              −
            </button>
            <span class="bs-count">{people}</span>
            <button class="bs-step" aria-label="More people" onClick={() => setPeople((n) => n + 1)}>
              +
            </button>
          </div>
        </div>
      </section>

      <section class="bs-card bs-results">
        <div class="bs-result-row">
          <span>Tip</span>
          <span>{usd.format(tipAmount)}</span>
        </div>
        <div class="bs-result-row">
          <span>Total</span>
          <span>{usd.format(total)}</span>
        </div>
        <div class="bs-result-hero">
          <span class="bs-result-hero-label">Per person</span>
          <span class="bs-result-hero-value">{usd.format(perPerson)}</span>
        </div>
        {round && roundingExtra > 0 && (
          <p class="bs-note">
            Rounding up collects {usd.format(roundingExtra)} extra · effective tip {effTip.toFixed(1)}%
          </p>
        )}
        {bill <= 0 && <p class="bs-empty">Enter a bill amount to begin.</p>}
      </section>

      <section class="bs-card bs-breakdown">
        <h2 class="bs-subtitle">Per-person breakdown</h2>
        <ul class="bs-people">
          {Array.from({ length: people }, (_, i) => (
            <li class="bs-person" key={i}>
              <span>Person {i + 1}</span>
              <span>{usd.format(perPerson)}</span>
            </li>
          ))}
        </ul>
      </section>

      <footer class="bs-footer">
        Splitting {usd.format(total)} between {people} · {theme.value} theme
      </footer>
    </main>
  );
}
