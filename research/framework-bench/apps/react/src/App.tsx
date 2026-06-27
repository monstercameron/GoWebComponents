import { createContext, useContext, useState, type ReactNode } from "react";

// --- shared formatting (SPEC: $1,234.56) ---
const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });

const PRESETS = [10, 15, 18, 20, 25];

// --- shared/global state: theme + roundUp via Context (React's shared-state primitive) ---
type Settings = {
  theme: "light" | "dark";
  roundUp: boolean;
  toggleTheme: () => void;
  toggleRoundUp: () => void;
};
const SettingsContext = createContext<Settings | null>(null);
const useSettings = () => {
  const ctx = useContext(SettingsContext);
  if (!ctx) throw new Error("SettingsContext missing");
  return ctx;
};

export function App() {
  // local component state
  const [bill, setBill] = useState(0);
  const [tipPercent, setTipPercent] = useState(18);
  const [people, setPeople] = useState(1);
  // shared state
  const [theme, setTheme] = useState<"light" | "dark">("light");
  const [roundUp, setRoundUp] = useState(false);

  const settings: Settings = {
    theme,
    roundUp,
    toggleTheme: () => setTheme((t) => (t === "dark" ? "light" : "dark")),
    toggleRoundUp: () => setRoundUp((r) => !r),
  };

  // derived
  const tipAmount = (bill * tipPercent) / 100;
  const total = bill + tipAmount;
  const perPersonRaw = people > 0 ? total / people : 0;
  const perPerson = roundUp ? Math.ceil(perPersonRaw) : perPersonRaw;

  return (
    <SettingsContext.Provider value={settings}>
      <main className="bs-app" data-theme={theme}>
        <Header />
        <Inputs
          bill={bill}
          tipPercent={tipPercent}
          people={people}
          setBill={setBill}
          setTipPercent={setTipPercent}
          setPeople={setPeople}
        />
        <Results bill={bill} tipPercent={tipPercent} people={people} />
        <Breakdown perPerson={perPerson} people={people} />
        <Footer total={total} people={people} theme={theme} />
      </main>
    </SettingsContext.Provider>
  );
}

function Header() {
  const { theme, roundUp, toggleTheme, toggleRoundUp } = useSettings();
  return (
    <header className="bs-header">
      <h1 className="bs-title">Bill Splitter</h1>
      <div className="bs-header-actions">
        <button className="bs-toggle" aria-pressed={roundUp} onClick={toggleRoundUp}>
          Round up
        </button>
        <button className="bs-toggle" aria-pressed={theme === "dark"} onClick={toggleTheme}>
          Dark
        </button>
      </div>
    </header>
  );
}

type InputsProps = {
  bill: number;
  tipPercent: number;
  people: number;
  setBill: (n: number) => void;
  setTipPercent: (n: number) => void;
  setPeople: (updater: (n: number) => number) => void;
};

function Inputs({ bill, tipPercent, people, setBill, setTipPercent, setPeople }: InputsProps) {
  const parse = (s: string) => {
    const v = parseFloat(s);
    return Number.isFinite(v) && v >= 0 ? v : 0;
  };
  return (
    <section className="bs-card bs-inputs">
      <label className="bs-field">
        <span className="bs-label">Bill amount</span>
        <div className="bs-input-wrap">
          <span className="bs-prefix">$</span>
          <input
            className="bs-input"
            type="number"
            min="0"
            step="0.01"
            value={bill || ""}
            onChange={(e) => setBill(parse(e.target.value))}
          />
        </div>
      </label>

      <div className="bs-field">
        <span className="bs-label">Tip</span>
        <div className="bs-presets">
          {PRESETS.map((p) => (
            <button
              key={p}
              className={"bs-preset" + (tipPercent === p ? " bs-preset--active" : "")}
              onClick={() => setTipPercent(p)}
            >
              {p}%
            </button>
          ))}
          <input
            className="bs-preset-custom"
            type="number"
            min="0"
            placeholder="Custom %"
            value={tipPercent || ""}
            onChange={(e) => setTipPercent(parse(e.target.value))}
          />
        </div>
      </div>

      <div className="bs-field">
        <span className="bs-label">People</span>
        <div className="bs-stepper">
          <button
            className="bs-step"
            aria-label="Fewer people"
            disabled={people <= 1}
            onClick={() => setPeople((n) => Math.max(1, n - 1))}
          >
            −
          </button>
          <span className="bs-count">{people}</span>
          <button className="bs-step" aria-label="More people" onClick={() => setPeople((n) => n + 1)}>
            +
          </button>
        </div>
      </div>
    </section>
  );
}

function Results({ bill, tipPercent, people }: { bill: number; tipPercent: number; people: number }) {
  const { roundUp } = useSettings();
  const tipAmount = (bill * tipPercent) / 100;
  const total = bill + tipAmount;
  const perPersonRaw = people > 0 ? total / people : 0;
  const perPerson = roundUp ? Math.ceil(perPersonRaw) : perPersonRaw;
  const totalCollected = roundUp ? perPerson * people : total;
  const roundingExtra = Math.max(0, totalCollected - total);
  const effTip = bill > 0 ? ((totalCollected - bill) / bill) * 100 : tipPercent;

  return (
    <section className="bs-card bs-results">
      <div className="bs-result-row">
        <span>Tip</span>
        <span>{usd.format(tipAmount)}</span>
      </div>
      <div className="bs-result-row">
        <span>Total</span>
        <span>{usd.format(total)}</span>
      </div>
      <div className="bs-result-hero">
        <span className="bs-result-hero-label">Per person</span>
        <span className="bs-result-hero-value">{usd.format(perPerson)}</span>
      </div>
      {roundUp && roundingExtra > 0 && (
        <p className="bs-note">
          Rounding up collects {usd.format(roundingExtra)} extra · effective tip {effTip.toFixed(1)}%
        </p>
      )}
      {bill <= 0 && <p className="bs-empty">Enter a bill amount to begin.</p>}
    </section>
  );
}

function Breakdown({ perPerson, people }: { perPerson: number; people: number }) {
  return (
    <section className="bs-card bs-breakdown">
      <h2 className="bs-subtitle">Per-person breakdown</h2>
      <ul className="bs-people">
        {Array.from({ length: people }, (_, i) => (
          <li className="bs-person" key={i}>
            <span>Person {i + 1}</span>
            <span>{usd.format(perPerson)}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function Footer({ total, people, theme }: { total: number; people: number; theme: string }) {
  return (
    <footer className="bs-footer">
      Splitting {usd.format(total)} between {people} · {theme} theme
    </footer>
  );
}
