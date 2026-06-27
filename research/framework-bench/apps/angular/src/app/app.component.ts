import { Component, computed, inject, signal } from "@angular/core";
import { SettingsService } from "./settings.service";

const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });
const PRESETS = [10, 15, 18, 20, 25];

@Component({
  selector: "app-root",
  standalone: true,
  template: `
    <main class="bs-app" [attr.data-theme]="settings.theme()">
      <header class="bs-header">
        <h1 class="bs-title">Bill Splitter</h1>
        <div class="bs-header-actions">
          <button class="bs-toggle" [attr.aria-pressed]="settings.roundUp()" (click)="settings.toggleRoundUp()">
            Round up
          </button>
          <button class="bs-toggle" [attr.aria-pressed]="settings.theme() === 'dark'" (click)="settings.toggleTheme()">
            Dark
          </button>
        </div>
      </header>

      <section class="bs-card bs-inputs">
        <label class="bs-field">
          <span class="bs-label">Bill amount</span>
          <div class="bs-input-wrap">
            <span class="bs-prefix">$</span>
            <input class="bs-input" type="number" min="0" step="0.01" [value]="bill() || ''" (input)="setBill($event)" />
          </div>
        </label>

        <div class="bs-field">
          <span class="bs-label">Tip</span>
          <div class="bs-presets">
            @for (p of presets; track p) {
              <button class="bs-preset" [class.bs-preset--active]="tipPercent() === p" (click)="tipPercent.set(p)">
                {{ p }}%
              </button>
            }
            <input
              class="bs-preset-custom"
              type="number"
              min="0"
              placeholder="Custom %"
              [value]="tipPercent() || ''"
              (input)="setTip($event)"
            />
          </div>
        </div>

        <div class="bs-field">
          <span class="bs-label">People</span>
          <div class="bs-stepper">
            <button class="bs-step" aria-label="Fewer people" [disabled]="people() <= 1" (click)="dec()">−</button>
            <span class="bs-count">{{ people() }}</span>
            <button class="bs-step" aria-label="More people" (click)="inc()">+</button>
          </div>
        </div>
      </section>

      <section class="bs-card bs-results">
        <div class="bs-result-row"><span>Tip</span><span>{{ fmt(tipAmount()) }}</span></div>
        <div class="bs-result-row"><span>Total</span><span>{{ fmt(total()) }}</span></div>
        <div class="bs-result-hero">
          <span class="bs-result-hero-label">Per person</span>
          <span class="bs-result-hero-value">{{ fmt(perPerson()) }}</span>
        </div>
        @if (settings.roundUp() && roundingExtra() > 0) {
          <p class="bs-note">
            Rounding up collects {{ fmt(roundingExtra()) }} extra · effective tip {{ effTip().toFixed(1) }}%
          </p>
        }
        @if (bill() <= 0) {
          <p class="bs-empty">Enter a bill amount to begin.</p>
        }
      </section>

      <section class="bs-card bs-breakdown">
        <h2 class="bs-subtitle">Per-person breakdown</h2>
        <ul class="bs-people">
          @for (n of peopleArray(); track n) {
            <li class="bs-person"><span>Person {{ n }}</span><span>{{ fmt(perPerson()) }}</span></li>
          }
        </ul>
      </section>

      <footer class="bs-footer">
        Splitting {{ fmt(total()) }} between {{ people() }} · {{ settings.theme() }} theme
      </footer>
    </main>
  `,
})
export class AppComponent {
  readonly settings = inject(SettingsService);
  readonly presets = PRESETS;

  // local component state
  readonly bill = signal(0);
  readonly tipPercent = signal(18);
  readonly people = signal(1);

  // derived
  readonly tipAmount = computed(() => (this.bill() * this.tipPercent()) / 100);
  readonly total = computed(() => this.bill() + this.tipAmount());
  readonly perPersonRaw = computed(() => (this.people() > 0 ? this.total() / this.people() : 0));
  readonly perPerson = computed(() =>
    this.settings.roundUp() ? Math.ceil(this.perPersonRaw()) : this.perPersonRaw(),
  );
  readonly totalCollected = computed(() =>
    this.settings.roundUp() ? this.perPerson() * this.people() : this.total(),
  );
  readonly roundingExtra = computed(() => Math.max(0, this.totalCollected() - this.total()));
  readonly effTip = computed(() =>
    this.bill() > 0 ? ((this.totalCollected() - this.bill()) / this.bill()) * 100 : this.tipPercent(),
  );
  readonly peopleArray = computed(() => Array.from({ length: this.people() }, (_, i) => i + 1));

  fmt(v: number) {
    return usd.format(v);
  }
  private parse(e: Event) {
    const v = parseFloat((e.target as HTMLInputElement).value);
    return Number.isFinite(v) && v >= 0 ? v : 0;
  }
  setBill(e: Event) {
    this.bill.set(this.parse(e));
  }
  setTip(e: Event) {
    this.tipPercent.set(this.parse(e));
  }
  dec() {
    this.people.update((n) => Math.max(1, n - 1));
  }
  inc() {
    this.people.update((n) => n + 1);
  }
}
