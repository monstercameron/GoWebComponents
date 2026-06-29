import { createSignal, For, Show } from "solid-js";
import { theme, setTheme, roundUp, setRoundUp, usd, PRESETS } from "./store";

export function App() {
  // local component state
  const [bill, setBill] = createSignal(0);
  const [tipPercent, setTipPercent] = createSignal(18);
  const [people, setPeople] = createSignal(1);

  const parse = (s: string) => {
    const v = parseFloat(s);
    return Number.isFinite(v) && v >= 0 ? v : 0;
  };

  // derived (plain functions recompute reactively)
  const tipAmount = () => (bill() * tipPercent()) / 100;
  const total = () => bill() + tipAmount();
  const perPersonRaw = () => (people() > 0 ? total() / people() : 0);
  const perPerson = () => (roundUp() ? Math.ceil(perPersonRaw()) : perPersonRaw());
  const totalCollected = () => (roundUp() ? perPerson() * people() : total());
  const roundingExtra = () => Math.max(0, totalCollected() - total());
  const effTip = () => (bill() > 0 ? ((totalCollected() - bill()) / bill()) * 100 : tipPercent());

  return (
    <main class="bs-app" data-theme={theme()}>
      <header class="bs-header">
        <h1 class="bs-title">Bill Splitter</h1>
        <div class="bs-header-actions">
          <button class="bs-toggle" aria-pressed={roundUp()} onClick={() => setRoundUp((r) => !r)}>
            Round up
          </button>
          <button
            class="bs-toggle"
            aria-pressed={theme() === "dark"}
            onClick={() => setTheme((t) => (t === "dark" ? "light" : "dark"))}
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
              value={bill() || ""}
              onInput={(e) => setBill(parse(e.currentTarget.value))}
            />
          </div>
        </label>

        <div class="bs-field">
          <span class="bs-label">Tip</span>
          <div class="bs-presets">
            <For each={PRESETS}>
              {(p) => (
                <button
                  classList={{ "bs-preset": true, "bs-preset--active": tipPercent() === p }}
                  onClick={() => setTipPercent(p)}
                >
                  {p}%
                </button>
              )}
            </For>
            <input
              class="bs-preset-custom"
              type="number"
              min="0"
              placeholder="Custom %"
              value={tipPercent() || ""}
              onInput={(e) => setTipPercent(parse(e.currentTarget.value))}
            />
          </div>
        </div>

        <div class="bs-field">
          <span class="bs-label">People</span>
          <div class="bs-stepper">
            <button
              class="bs-step"
              aria-label="Fewer people"
              disabled={people() <= 1}
              onClick={() => setPeople((n) => Math.max(1, n - 1))}
            >
              −
            </button>
            <span class="bs-count">{people()}</span>
            <button class="bs-step" aria-label="More people" onClick={() => setPeople((n) => n + 1)}>
              +
            </button>
          </div>
        </div>
      </section>

      <section class="bs-card bs-results">
        <div class="bs-result-row">
          <span>Tip</span>
          <span>{usd.format(tipAmount())}</span>
        </div>
        <div class="bs-result-row">
          <span>Total</span>
          <span>{usd.format(total())}</span>
        </div>
        <div class="bs-result-hero">
          <span class="bs-result-hero-label">Per person</span>
          <span class="bs-result-hero-value">{usd.format(perPerson())}</span>
        </div>
        <Show when={roundUp() && roundingExtra() > 0}>
          <p class="bs-note">
            Rounding up collects {usd.format(roundingExtra())} extra · effective tip {effTip().toFixed(1)}%
          </p>
        </Show>
        <Show when={bill() <= 0}>
          <p class="bs-empty">Enter a bill amount to begin.</p>
        </Show>
      </section>

      <section class="bs-card bs-breakdown">
        <h2 class="bs-subtitle">Per-person breakdown</h2>
        <ul class="bs-people">
          <For each={Array.from({ length: people() }, (_, i) => i + 1)}>
            {(n) => (
              <li class="bs-person">
                <span>Person {n}</span>
                <span>{usd.format(perPerson())}</span>
              </li>
            )}
          </For>
        </ul>
      </section>

      <footer class="bs-footer">
        Splitting {usd.format(total())} between {people()} · {theme()} theme
      </footer>
    </main>
  );
}
