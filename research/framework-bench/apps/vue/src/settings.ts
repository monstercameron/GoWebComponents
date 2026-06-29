import { reactive } from "vue";

// Shared/global state (theme + roundUp) as a reactive store module — Vue's
// dependency-free shared-state idiom. Imported by any component that needs it.
export const settings = reactive({
  theme: "light" as "light" | "dark",
  roundUp: false,
  toggleTheme() {
    this.theme = this.theme === "dark" ? "light" : "dark";
  },
  toggleRoundUp() {
    this.roundUp = !this.roundUp;
  },
});

export const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });
export const PRESETS = [10, 15, 18, 20, 25];
