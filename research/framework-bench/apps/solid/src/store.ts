import { createSignal } from "solid-js";

// Shared/global state as module-level signals — Solid's idiomatic cross-component
// primitive (any component importing these subscribes fine-grained).
export const [theme, setTheme] = createSignal<"light" | "dark">("light");
export const [roundUp, setRoundUp] = createSignal(false);

export const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });
export const PRESETS = [10, 15, 18, 20, 25];
