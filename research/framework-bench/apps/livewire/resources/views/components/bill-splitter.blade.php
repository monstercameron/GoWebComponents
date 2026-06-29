{{--
  Bill Splitter — Livewire v4 single-file component.
  In v4, component logic + view live in one file under resources/views/components/
  (optionally prefixed with ⚡). State is server-side public properties; every
  wire:* interaction round-trips and morphs the DOM. Drop a layout that links the
  canonical ../../shared/styles.css and render with <livewire:bill-splitter /> (or
  the ⚡ tag). Status: to-spec scaffold, needs a Laravel v4 app; not build-verified.
--}}
<?php

use Livewire\Component;

new class extends Component {
    public float $bill = 0;
    public float $tip = 18;
    public int $people = 1;
    public string $theme = 'light';
    public bool $roundUp = false;
    public array $presets = [10, 15, 18, 20, 25];

    public function toggleTheme(): void
    {
        $this->theme = $this->theme === 'dark' ? 'light' : 'dark';
    }

    public function toggleRoundUp(): void
    {
        $this->roundUp = ! $this->roundUp;
    }

    public function inc(): void
    {
        $this->people++;
    }

    public function dec(): void
    {
        $this->people = max(1, $this->people - 1);
    }

    public function preset(float $p): void
    {
        $this->tip = $p;
    }
};
?>

@php
    $usd = fn ($v) => '$' . number_format(max(0, $v), 2);
    $tipAmount = $bill * $tip / 100;
    $total = $bill + $tipAmount;
    $perPersonRaw = $people > 0 ? $total / $people : 0;
    $perPerson = $roundUp ? ceil($perPersonRaw) : $perPersonRaw;
    $totalCollected = $roundUp ? $perPerson * $people : $total;
    $roundingExtra = max(0, $totalCollected - $total);
    $effTip = $bill > 0 ? ($totalCollected - $bill) / $bill * 100 : $tip;
@endphp

<main class="bs-app" data-theme="{{ $theme }}">
    <header class="bs-header">
        <h1 class="bs-title">Bill Splitter</h1>
        <div class="bs-header-actions">
            <button class="bs-toggle" aria-pressed="{{ $roundUp ? 'true' : 'false' }}" wire:click="toggleRoundUp">Round up</button>
            <button class="bs-toggle" aria-pressed="{{ $theme === 'dark' ? 'true' : 'false' }}" wire:click="toggleTheme">Dark</button>
        </div>
    </header>

    <section class="bs-card bs-inputs">
        <label class="bs-field">
            <span class="bs-label">Bill amount</span>
            <div class="bs-input-wrap">
                <span class="bs-prefix">$</span>
                <input class="bs-input" type="number" min="0" step="0.01" wire:model.live="bill" />
            </div>
        </label>

        <div class="bs-field">
            <span class="bs-label">Tip</span>
            <div class="bs-presets">
                @foreach ($presets as $p)
                    <button class="bs-preset @if ((float) $tip === (float) $p) bs-preset--active @endif" wire:click="preset({{ $p }})">{{ $p }}%</button>
                @endforeach
                <input class="bs-preset-custom" type="number" min="0" placeholder="Custom %" wire:model.live="tip" />
            </div>
        </div>

        <div class="bs-field">
            <span class="bs-label">People</span>
            <div class="bs-stepper">
                <button class="bs-step" aria-label="Fewer people" @disabled($people <= 1) wire:click="dec">−</button>
                <span class="bs-count">{{ $people }}</span>
                <button class="bs-step" aria-label="More people" wire:click="inc">+</button>
            </div>
        </div>
    </section>

    <section class="bs-card bs-results">
        <div class="bs-result-row"><span>Tip</span><span>{{ $usd($tipAmount) }}</span></div>
        <div class="bs-result-row"><span>Total</span><span>{{ $usd($total) }}</span></div>
        <div class="bs-result-hero">
            <span class="bs-result-hero-label">Per person</span>
            <span class="bs-result-hero-value">{{ $usd($perPerson) }}</span>
        </div>
        @if ($roundUp && $roundingExtra > 0)
            <p class="bs-note">Rounding up collects {{ $usd($roundingExtra) }} extra · effective tip {{ number_format($effTip, 1) }}%</p>
        @endif
        @if ($bill <= 0)
            <p class="bs-empty">Enter a bill amount to begin.</p>
        @endif
    </section>

    <section class="bs-card bs-breakdown">
        <h2 class="bs-subtitle">Per-person breakdown</h2>
        <ul class="bs-people">
            @for ($n = 1; $n <= $people; $n++)
                <li class="bs-person"><span>Person {{ $n }}</span><span>{{ $usd($perPerson) }}</span></li>
            @endfor
        </ul>
    </section>

    <footer class="bs-footer">Splitting {{ $usd($total) }} between {{ $people }} · {{ $theme }} theme</footer>
</main>
