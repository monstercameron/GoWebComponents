<script setup lang="ts">
import { computed, ref } from "vue";
import { settings, usd, PRESETS } from "./settings";

// local component state
const bill = ref(0);
const tipPercent = ref(18);
const people = ref(1);

const parse = (s: string) => {
  const v = parseFloat(s);
  return Number.isFinite(v) && v >= 0 ? v : 0;
};

// derived
const tipAmount = computed(() => (bill.value * tipPercent.value) / 100);
const total = computed(() => bill.value + tipAmount.value);
const perPersonRaw = computed(() => (people.value > 0 ? total.value / people.value : 0));
const perPerson = computed(() => (settings.roundUp ? Math.ceil(perPersonRaw.value) : perPersonRaw.value));
const totalCollected = computed(() => (settings.roundUp ? perPerson.value * people.value : total.value));
const roundingExtra = computed(() => Math.max(0, totalCollected.value - total.value));
const effTip = computed(() =>
  bill.value > 0 ? ((totalCollected.value - bill.value) / bill.value) * 100 : tipPercent.value,
);
</script>

<template>
  <main class="bs-app" :data-theme="settings.theme">
    <header class="bs-header">
      <h1 class="bs-title">Bill Splitter</h1>
      <div class="bs-header-actions">
        <button class="bs-toggle" :aria-pressed="settings.roundUp" @click="settings.toggleRoundUp()">
          Round up
        </button>
        <button class="bs-toggle" :aria-pressed="settings.theme === 'dark'" @click="settings.toggleTheme()">
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
            :value="bill || ''"
            @input="bill = parse(($event.target as HTMLInputElement).value)"
          />
        </div>
      </label>

      <div class="bs-field">
        <span class="bs-label">Tip</span>
        <div class="bs-presets">
          <button
            v-for="p in PRESETS"
            :key="p"
            class="bs-preset"
            :class="{ 'bs-preset--active': tipPercent === p }"
            @click="tipPercent = p"
          >
            {{ p }}%
          </button>
          <input
            class="bs-preset-custom"
            type="number"
            min="0"
            placeholder="Custom %"
            :value="tipPercent || ''"
            @input="tipPercent = parse(($event.target as HTMLInputElement).value)"
          />
        </div>
      </div>

      <div class="bs-field">
        <span class="bs-label">People</span>
        <div class="bs-stepper">
          <button class="bs-step" aria-label="Fewer people" :disabled="people <= 1" @click="people = Math.max(1, people - 1)">
            −
          </button>
          <span class="bs-count">{{ people }}</span>
          <button class="bs-step" aria-label="More people" @click="people = people + 1">+</button>
        </div>
      </div>
    </section>

    <section class="bs-card bs-results">
      <div class="bs-result-row"><span>Tip</span><span>{{ usd.format(tipAmount) }}</span></div>
      <div class="bs-result-row"><span>Total</span><span>{{ usd.format(total) }}</span></div>
      <div class="bs-result-hero">
        <span class="bs-result-hero-label">Per person</span>
        <span class="bs-result-hero-value">{{ usd.format(perPerson) }}</span>
      </div>
      <p v-if="settings.roundUp && roundingExtra > 0" class="bs-note">
        Rounding up collects {{ usd.format(roundingExtra) }} extra · effective tip {{ effTip.toFixed(1) }}%
      </p>
      <p v-if="bill <= 0" class="bs-empty">Enter a bill amount to begin.</p>
    </section>

    <section class="bs-card bs-breakdown">
      <h2 class="bs-subtitle">Per-person breakdown</h2>
      <ul class="bs-people">
        <li v-for="n in people" :key="n" class="bs-person">
          <span>Person {{ n }}</span>
          <span>{{ usd.format(perPerson) }}</span>
        </li>
      </ul>
    </section>

    <footer class="bs-footer">
      Splitting {{ usd.format(total) }} between {{ people }} · {{ settings.theme }} theme
    </footer>
  </main>
</template>
