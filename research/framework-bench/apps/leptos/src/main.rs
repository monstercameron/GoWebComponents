// Bill Splitter — Leptos 0.7 (Rust → wasm). Signals for local state; shared theme +
// round_up provided via context. Build/serve with Trunk: `trunk serve`.
// Status: to-spec scaffold, not build-verified here (cargo deps fetched on build).
use leptos::prelude::*;

const PRESETS: [i32; 5] = [10, 15, 18, 20, 25];

fn usd(v: f64) -> String {
    let cents = ((v * 100.0).round()).max(0.0) as i64;
    let s = (cents / 100).to_string();
    let len = s.len();
    let mut out = String::new();
    for (i, ch) in s.chars().enumerate() {
        if i > 0 && (len - i) % 3 == 0 {
            out.push(',');
        }
        out.push(ch);
    }
    format!("${}.{:02}", out, cents % 100)
}

fn parse_num(s: &str) -> f64 {
    s.parse::<f64>().ok().filter(|v| *v >= 0.0).unwrap_or(0.0)
}

#[component]
fn App() -> impl IntoView {
    // local state
    let bill = RwSignal::new(0.0_f64);
    let tip = RwSignal::new(18.0_f64);
    let people = RwSignal::new(1_i32);
    // shared state via context
    let theme = RwSignal::new(String::from("light"));
    let round_up = RwSignal::new(false);
    provide_context(theme);
    provide_context(round_up);

    // derived
    let tip_amount = move || bill.get() * tip.get() / 100.0;
    let total = move || bill.get() + tip_amount();
    let per_person_raw = move || if people.get() > 0 { total() / people.get() as f64 } else { 0.0 };
    let per_person = move || if round_up.get() { per_person_raw().ceil() } else { per_person_raw() };
    let total_collected = move || if round_up.get() { per_person() * people.get() as f64 } else { total() };
    let rounding_extra = move || (total_collected() - total()).max(0.0);
    let eff_tip = move || if bill.get() > 0.0 { (total_collected() - bill.get()) / bill.get() * 100.0 } else { tip.get() };

    view! {
        <main class="bs-app" attr:data-theme=move || theme.get()>
            <Header/>

            <section class="bs-card bs-inputs">
                <label class="bs-field">
                    <span class="bs-label">"Bill amount"</span>
                    <div class="bs-input-wrap">
                        <span class="bs-prefix">"$"</span>
                        <input class="bs-input" type="number" min="0" step="0.01"
                            prop:value=move || if bill.get() == 0.0 { String::new() } else { bill.get().to_string() }
                            on:input=move |e| bill.set(parse_num(&event_target_value(&e))) />
                    </div>
                </label>

                <div class="bs-field">
                    <span class="bs-label">"Tip"</span>
                    <div class="bs-presets">
                        <For each=move || PRESETS key=|p| *p children=move |p| {
                            view! {
                                <button class="bs-preset" class:bs-preset--active=move || tip.get() == p as f64
                                    on:click=move |_| tip.set(p as f64)>{p}"%"</button>
                            }
                        }/>
                        <input class="bs-preset-custom" type="number" min="0" placeholder="Custom %"
                            prop:value=move || if tip.get() == 0.0 { String::new() } else { tip.get().to_string() }
                            on:input=move |e| tip.set(parse_num(&event_target_value(&e))) />
                    </div>
                </div>

                <div class="bs-field">
                    <span class="bs-label">"People"</span>
                    <div class="bs-stepper">
                        <button class="bs-step" aria-label="Fewer people" prop:disabled=move || people.get() <= 1
                            on:click=move |_| people.update(|n| *n = (*n - 1).max(1))>"−"</button>
                        <span class="bs-count">{move || people.get()}</span>
                        <button class="bs-step" aria-label="More people" on:click=move |_| people.update(|n| *n += 1)>"+"</button>
                    </div>
                </div>
            </section>

            <section class="bs-card bs-results">
                <div class="bs-result-row"><span>"Tip"</span><span>{move || usd(tip_amount())}</span></div>
                <div class="bs-result-row"><span>"Total"</span><span>{move || usd(total())}</span></div>
                <div class="bs-result-hero">
                    <span class="bs-result-hero-label">"Per person"</span>
                    <span class="bs-result-hero-value">{move || usd(per_person())}</span>
                </div>
                {move || (round_up.get() && rounding_extra() > 0.0).then(|| view! {
                    <p class="bs-note">"Rounding up collects "{usd(rounding_extra())}" extra · effective tip "{format!("{:.1}", eff_tip())}"%"</p>
                })}
                {move || (bill.get() <= 0.0).then(|| view! { <p class="bs-empty">"Enter a bill amount to begin."</p> })}
            </section>

            <section class="bs-card bs-breakdown">
                <h2 class="bs-subtitle">"Per-person breakdown"</h2>
                <ul class="bs-people">
                    <For each=move || (1..=people.get()).collect::<Vec<_>>() key=|n| *n children=move |n| {
                        view! { <li class="bs-person"><span>"Person "{n}</span><span>{usd(per_person())}</span></li> }
                    }/>
                </ul>
            </section>

            <footer class="bs-footer">
                {move || format!("Splitting {} between {} · {} theme", usd(total()), people.get(), theme.get())}
            </footer>
        </main>
    }
}

#[component]
fn Header() -> impl IntoView {
    // consume shared context — demonstrates cross-component shared state
    let theme = use_context::<RwSignal<String>>().unwrap();
    let round_up = use_context::<RwSignal<bool>>().unwrap();
    view! {
        <header class="bs-header">
            <h1 class="bs-title">"Bill Splitter"</h1>
            <div class="bs-header-actions">
                <button class="bs-toggle" attr:aria-pressed=move || round_up.get().to_string()
                    on:click=move |_| round_up.update(|r| *r = !*r)>"Round up"</button>
                <button class="bs-toggle" attr:aria-pressed=move || (theme.get() == "dark").to_string()
                    on:click=move |_| theme.update(|t| *t = if t == "dark" { "light".into() } else { "dark".into() })>"Dark"</button>
            </div>
        </header>
    }
}

fn main() {
    leptos::mount_to_body(App);
}
