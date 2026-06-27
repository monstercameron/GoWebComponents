import {
  component$,
  createContextId,
  useContext,
  useContextProvider,
  useSignal,
  useStore,
} from "@builder.io/qwik";

type Settings = { theme: "light" | "dark"; roundUp: boolean };
const SettingsCtx = createContextId<Settings>("bs.settings");

const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });
const PRESETS = [10, 15, 18, 20, 25];
const parse = (s: string) => {
  const v = parseFloat(s);
  return Number.isFinite(v) && v >= 0 ? v : 0;
};

// Header is its own component consuming shared context — demonstrates shared state.
export const Header = component$(() => {
  const s = useContext(SettingsCtx);
  return (
    <header class="bs-header">
      <h1 class="bs-title">Bill Splitter</h1>
      <div class="bs-header-actions">
        <button class="bs-toggle" aria-pressed={s.roundUp} onClick$={() => (s.roundUp = !s.roundUp)}>
          Round up
        </button>
        <button
          class="bs-toggle"
          aria-pressed={s.theme === "dark"}
          onClick$={() => (s.theme = s.theme === "dark" ? "light" : "dark")}
        >
          Dark
        </button>
      </div>
    </header>
  );
});

export const App = component$(() => {
  // shared/global state via store + context
  const settings = useStore<Settings>({ theme: "light", roundUp: false });
  useContextProvider(SettingsCtx, settings);

  // local component state
  const bill = useSignal(0);
  const tipPercent = useSignal(18);
  const people = useSignal(1);

  // derived
  const tipAmount = (bill.value * tipPercent.value) / 100;
  const total = bill.value + tipAmount;
  const perPersonRaw = people.value > 0 ? total / people.value : 0;
  const perPerson = settings.roundUp ? Math.ceil(perPersonRaw) : perPersonRaw;
  const totalCollected = settings.roundUp ? perPerson * people.value : total;
  const roundingExtra = Math.max(0, totalCollected - total);
  const effTip = bill.value > 0 ? ((totalCollected - bill.value) / bill.value) * 100 : tipPercent.value;

  return (
    <main class="bs-app" data-theme={settings.theme}>
      <Header />

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
              value={bill.value || ""}
              onInput$={(_, el) => (bill.value = parse(el.value))}
            />
          </div>
        </label>

        <div class="bs-field">
          <span class="bs-label">Tip</span>
          <div class="bs-presets">
            {PRESETS.map((p) => (
              <button
                key={p}
                class={"bs-preset" + (tipPercent.value === p ? " bs-preset--active" : "")}
                onClick$={() => (tipPercent.value = p)}
              >
                {p}%
              </button>
            ))}
            <input
              class="bs-preset-custom"
              type="number"
              min="0"
              placeholder="Custom %"
              value={tipPercent.value || ""}
              onInput$={(_, el) => (tipPercent.value = parse(el.value))}
            />
          </div>
        </div>

        <div class="bs-field">
          <span class="bs-label">People</span>
          <div class="bs-stepper">
            <button
              class="bs-step"
              aria-label="Fewer people"
              disabled={people.value <= 1}
              onClick$={() => (people.value = Math.max(1, people.value - 1))}
            >
              −
            </button>
            <span class="bs-count">{people.value}</span>
            <button class="bs-step" aria-label="More people" onClick$={() => (people.value = people.value + 1)}>
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
        {settings.roundUp && roundingExtra > 0 && (
          <p class="bs-note">
            Rounding up collects {usd.format(roundingExtra)} extra · effective tip {effTip.toFixed(1)}%
          </p>
        )}
        {bill.value <= 0 && <p class="bs-empty">Enter a bill amount to begin.</p>}
      </section>

      <section class="bs-card bs-breakdown">
        <h2 class="bs-subtitle">Per-person breakdown</h2>
        <ul class="bs-people">
          {Array.from({ length: people.value }, (_, i) => i + 1).map((n) => (
            <li class="bs-person" key={n}>
              <span>Person {n}</span>
              <span>{usd.format(perPerson)}</span>
            </li>
          ))}
        </ul>
      </section>

      <footer class="bs-footer">
        Splitting {usd.format(total)} between {people.value} · {settings.theme} theme
      </footer>
    </main>
  );
});
