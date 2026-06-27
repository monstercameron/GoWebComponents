import type { ReactiveController, ReactiveControllerHost } from "lit";

// Shared/global state as a singleton EventTarget store — consumed by multiple
// custom elements through a Lit ReactiveController (the idiomatic shared-state path).
class SettingsStore extends EventTarget {
  theme: "light" | "dark" = "light";
  roundUp = false;

  toggleTheme() {
    this.theme = this.theme === "dark" ? "light" : "dark";
    this.dispatchEvent(new Event("change"));
  }
  toggleRoundUp() {
    this.roundUp = !this.roundUp;
    this.dispatchEvent(new Event("change"));
  }
}

export const settings = new SettingsStore();

// Subscribes a host element to the store and requests an update on change.
export class SettingsController implements ReactiveController {
  private onChange = () => this.host.requestUpdate();
  constructor(private host: ReactiveControllerHost) {
    host.addController(this);
  }
  hostConnected() {
    settings.addEventListener("change", this.onChange);
  }
  hostDisconnected() {
    settings.removeEventListener("change", this.onChange);
  }
}

export const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });
export const PRESETS = [10, 15, 18, 20, 25];
