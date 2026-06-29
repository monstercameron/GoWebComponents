import { Injectable, signal } from "@angular/core";

// Shared/global state as an injectable singleton service holding signals —
// Angular's idiomatic cross-component shared-state mechanism.
@Injectable({ providedIn: "root" })
export class SettingsService {
  readonly theme = signal<"light" | "dark">("light");
  readonly roundUp = signal(false);

  toggleTheme() {
    this.theme.update((t) => (t === "dark" ? "light" : "dark"));
  }
  toggleRoundUp() {
    this.roundUp.update((r) => !r);
  }
}
