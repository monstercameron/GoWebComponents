// Shared/global state via a runes module (.svelte.ts) — the idiomatic Svelte 5
// cross-component primitive (replaces stores for shared app state). The $state
// object's properties are deeply reactive; any component that imports `settings`
// reads/writes it directly.
export const settings = $state<{ theme: "light" | "dark"; roundUp: boolean }>({
  theme: "light",
  roundUp: false,
});

export const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });
export const PRESETS = [10, 15, 18, 20, 25];
