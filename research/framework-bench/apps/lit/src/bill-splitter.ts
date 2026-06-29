import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";
import { settings, SettingsController, usd, PRESETS } from "./settings";

const parse = (s: string) => {
  const v = parseFloat(s);
  return Number.isFinite(v) && v >= 0 ? v : 0;
};

// Render into light DOM (not shadow DOM) so the canonical global stylesheet applies.
class LightElement extends LitElement {
  protected createRenderRoot() {
    return this;
  }
}

// Header is its own custom element sharing the settings store — demonstrates
// cross-component shared state in Lit.
@customElement("bs-header")
export class BsHeader extends LightElement {
  private settings = new SettingsController(this);
  render() {
    return html`
      <header class="bs-header">
        <h1 class="bs-title">Bill Splitter</h1>
        <div class="bs-header-actions">
          <button class="bs-toggle" aria-pressed=${settings.roundUp} @click=${() => settings.toggleRoundUp()}>
            Round up
          </button>
          <button
            class="bs-toggle"
            aria-pressed=${settings.theme === "dark"}
            @click=${() => settings.toggleTheme()}
          >
            Dark
          </button>
        </div>
      </header>
    `;
  }
}

@customElement("bill-splitter")
export class BillSplitter extends LightElement {
  private settingsCtrl = new SettingsController(this);

  @state() private bill = 0;
  @state() private tipPercent = 18;
  @state() private people = 1;

  render() {
    const round = settings.roundUp;
    const tipAmount = (this.bill * this.tipPercent) / 100;
    const total = this.bill + tipAmount;
    const perPersonRaw = this.people > 0 ? total / this.people : 0;
    const perPerson = round ? Math.ceil(perPersonRaw) : perPersonRaw;
    const totalCollected = round ? perPerson * this.people : total;
    const roundingExtra = Math.max(0, totalCollected - total);
    const effTip = this.bill > 0 ? ((totalCollected - this.bill) / this.bill) * 100 : this.tipPercent;
    void this.settingsCtrl;

    return html`
      <main class="bs-app" data-theme=${settings.theme}>
        <bs-header></bs-header>

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
                .value=${this.bill || ""}
                @input=${(e: Event) => (this.bill = parse((e.target as HTMLInputElement).value))}
              />
            </div>
          </label>

          <div class="bs-field">
            <span class="bs-label">Tip</span>
            <div class="bs-presets">
              ${PRESETS.map(
                (p) => html`
                  <button
                    class="bs-preset ${this.tipPercent === p ? "bs-preset--active" : ""}"
                    @click=${() => (this.tipPercent = p)}
                  >
                    ${p}%
                  </button>
                `,
              )}
              <input
                class="bs-preset-custom"
                type="number"
                min="0"
                placeholder="Custom %"
                .value=${this.tipPercent || ""}
                @input=${(e: Event) => (this.tipPercent = parse((e.target as HTMLInputElement).value))}
              />
            </div>
          </div>

          <div class="bs-field">
            <span class="bs-label">People</span>
            <div class="bs-stepper">
              <button
                class="bs-step"
                aria-label="Fewer people"
                ?disabled=${this.people <= 1}
                @click=${() => (this.people = Math.max(1, this.people - 1))}
              >
                −
              </button>
              <span class="bs-count">${this.people}</span>
              <button class="bs-step" aria-label="More people" @click=${() => (this.people = this.people + 1)}>
                +
              </button>
            </div>
          </div>
        </section>

        <section class="bs-card bs-results">
          <div class="bs-result-row"><span>Tip</span><span>${usd.format(tipAmount)}</span></div>
          <div class="bs-result-row"><span>Total</span><span>${usd.format(total)}</span></div>
          <div class="bs-result-hero">
            <span class="bs-result-hero-label">Per person</span>
            <span class="bs-result-hero-value">${usd.format(perPerson)}</span>
          </div>
          ${round && roundingExtra > 0
            ? html`<p class="bs-note">
                Rounding up collects ${usd.format(roundingExtra)} extra · effective tip ${effTip.toFixed(1)}%
              </p>`
            : nothing}
          ${this.bill <= 0 ? html`<p class="bs-empty">Enter a bill amount to begin.</p>` : nothing}
        </section>

        <section class="bs-card bs-breakdown">
          <h2 class="bs-subtitle">Per-person breakdown</h2>
          <ul class="bs-people">
            ${Array.from({ length: this.people }, (_, i) => i + 1).map(
              (n) => html`<li class="bs-person"><span>Person ${n}</span><span>${usd.format(perPerson)}</span></li>`,
            )}
          </ul>
        </section>

        <footer class="bs-footer">
          Splitting ${usd.format(total)} between ${this.people} · ${settings.theme} theme
        </footer>
      </main>
    `;
  }
}
