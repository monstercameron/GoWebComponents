<script lang="ts">
  // Svelte 5 runes: $state for local, $derived for computed; shared theme/roundUp
  // come from the settings.svelte.ts runes module.
  import { settings, usd, PRESETS } from "./settings.svelte";

  let bill = $state(0);
  let tipPercent = $state(18);
  let people = $state(1);

  const parse = (s: string) => {
    const v = parseFloat(s);
    return Number.isFinite(v) && v >= 0 ? v : 0;
  };

  let tipAmount = $derived((bill * tipPercent) / 100);
  let total = $derived(bill + tipAmount);
  let perPersonRaw = $derived(people > 0 ? total / people : 0);
  let perPerson = $derived(settings.roundUp ? Math.ceil(perPersonRaw) : perPersonRaw);
  let totalCollected = $derived(settings.roundUp ? perPerson * people : total);
  let roundingExtra = $derived(Math.max(0, totalCollected - total));
  let effTip = $derived(bill > 0 ? ((totalCollected - bill) / bill) * 100 : tipPercent);
</script>

<main class="bs-app" data-theme={settings.theme}>
  <header class="bs-header">
    <h1 class="bs-title">Bill Splitter</h1>
    <div class="bs-header-actions">
      <button class="bs-toggle" aria-pressed={settings.roundUp} onclick={() => (settings.roundUp = !settings.roundUp)}>
        Round up
      </button>
      <button
        class="bs-toggle"
        aria-pressed={settings.theme === "dark"}
        onclick={() => (settings.theme = settings.theme === "dark" ? "light" : "dark")}
      >
        Dark
      </button>
    </div>
  </header>

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
          value={bill || ""}
          oninput={(e) => (bill = parse(e.currentTarget.value))}
        />
      </div>
    </label>

    <div class="bs-field">
      <span class="bs-label">Tip</span>
      <div class="bs-presets">
        {#each PRESETS as p (p)}
          <button class="bs-preset" class:bs-preset--active={tipPercent === p} onclick={() => (tipPercent = p)}>
            {p}%
          </button>
        {/each}
        <input
          class="bs-preset-custom"
          type="number"
          min="0"
          placeholder="Custom %"
          value={tipPercent || ""}
          oninput={(e) => (tipPercent = parse(e.currentTarget.value))}
        />
      </div>
    </div>

    <div class="bs-field">
      <span class="bs-label">People</span>
      <div class="bs-stepper">
        <button class="bs-step" aria-label="Fewer people" disabled={people <= 1} onclick={() => (people = Math.max(1, people - 1))}>
          −
        </button>
        <span class="bs-count">{people}</span>
        <button class="bs-step" aria-label="More people" onclick={() => (people = people + 1)}>+</button>
      </div>
    </div>
  </section>

  <section class="bs-card bs-results">
    <div class="bs-result-row"><span>Tip</span><span>{usd.format(tipAmount)}</span></div>
    <div class="bs-result-row"><span>Total</span><span>{usd.format(total)}</span></div>
    <div class="bs-result-hero">
      <span class="bs-result-hero-label">Per person</span>
      <span class="bs-result-hero-value">{usd.format(perPerson)}</span>
    </div>
    {#if settings.roundUp && roundingExtra > 0}
      <p class="bs-note">
        Rounding up collects {usd.format(roundingExtra)} extra · effective tip {effTip.toFixed(1)}%
      </p>
    {/if}
    {#if bill <= 0}
      <p class="bs-empty">Enter a bill amount to begin.</p>
    {/if}
  </section>

  <section class="bs-card bs-breakdown">
    <h2 class="bs-subtitle">Per-person breakdown</h2>
    <ul class="bs-people">
      {#each Array(people) as _, i (i)}
        <li class="bs-person">
          <span>Person {i + 1}</span>
          <span>{usd.format(perPerson)}</span>
        </li>
      {/each}
    </ul>
  </section>

  <footer class="bs-footer">
    Splitting {usd.format(total)} between {people} · {settings.theme} theme
  </footer>
</main>
