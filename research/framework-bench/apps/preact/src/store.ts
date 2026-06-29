import { signal } from "@preact/signals";

// Shared/global state via Preact signals — read .value in any component.
export const theme = signal<"light" | "dark">("light");
export const roundUp = signal(false);

export const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });
export const PRESETS = [10, 15, 18, 20, 25];
