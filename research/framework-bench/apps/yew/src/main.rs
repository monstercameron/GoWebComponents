// Bill Splitter — Yew 0.23 (Rust → wasm). use_state for local; shared theme +
// round_up via use_reducer + ContextProvider. Run with `trunk serve`.
// Yew 0.23 renamed #[function_component] → #[component]; the function name is the
// component. Status: to-spec scaffold, not build-verified here.
#![allow(non_snake_case)]
use std::rc::Rc;
use wasm_bindgen::JsCast;
use web_sys::HtmlInputElement;
use yew::prelude::*;

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
fn input_value(e: &InputEvent) -> String {
    e.target().unwrap().unchecked_into::<HtmlInputElement>().value()
}

#[derive(Clone, PartialEq)]
struct Settings {
    theme: String,
    round_up: bool,
}
enum SAction {
    ToggleTheme,
    ToggleRound,
}
impl Reducible for Settings {
    type Action = SAction;
    fn reduce(self: Rc<Self>, action: Self::Action) -> Rc<Self> {
        let mut s = (*self).clone();
        match action {
            SAction::ToggleTheme => s.theme = if s.theme == "dark" { "light".into() } else { "dark".into() },
            SAction::ToggleRound => s.round_up = !s.round_up,
        }
        s.into()
    }
}
type SettingsCtx = UseReducerHandle<Settings>;

#[component]
fn Header() -> Html {
    let settings = use_context::<SettingsCtx>().unwrap();
    let toggle_theme = {
        let s = settings.clone();
        Callback::from(move |_| s.dispatch(SAction::ToggleTheme))
    };
    let toggle_round = {
        let s = settings.clone();
        Callback::from(move |_| s.dispatch(SAction::ToggleRound))
    };
    html! {
        <header class="bs-header">
            <h1 class="bs-title">{ "Bill Splitter" }</h1>
            <div class="bs-header-actions">
                <button class="bs-toggle" aria-pressed={settings.round_up.to_string()} onclick={toggle_round}>{ "Round up" }</button>
                <button class="bs-toggle" aria-pressed={(settings.theme == "dark").to_string()} onclick={toggle_theme}>{ "Dark" }</button>
            </div>
        </header>
    }
}

#[component]
fn App() -> Html {
    let settings = use_reducer(|| Settings { theme: "light".into(), round_up: false });
    let bill = use_state(|| 0.0_f64);
    let tip = use_state(|| 18.0_f64);
    let people = use_state(|| 1_i32);

    let round = settings.round_up;
    let tip_amount = *bill * *tip / 100.0;
    let total = *bill + tip_amount;
    let per_person_raw = if *people > 0 { total / *people as f64 } else { 0.0 };
    let per_person = if round { per_person_raw.ceil() } else { per_person_raw };
    let total_collected = if round { per_person * *people as f64 } else { total };
    let rounding_extra = (total_collected - total).max(0.0);
    let eff_tip = if *bill > 0.0 { (total_collected - *bill) / *bill * 100.0 } else { *tip };

    let on_bill = {
        let bill = bill.clone();
        Callback::from(move |e: InputEvent| bill.set(parse_num(&input_value(&e))))
    };
    let on_tip = {
        let tip = tip.clone();
        Callback::from(move |e: InputEvent| tip.set(parse_num(&input_value(&e))))
    };

    html! {
        <ContextProvider<SettingsCtx> context={settings.clone()}>
            <main class="bs-app" data-theme={settings.theme.clone()}>
                <Header/>

                <section class="bs-card bs-inputs">
                    <label class="bs-field">
                        <span class="bs-label">{ "Bill amount" }</span>
                        <div class="bs-input-wrap">
                            <span class="bs-prefix">{ "$" }</span>
                            <input class="bs-input" type="number" min="0" step="0.01"
                                value={if *bill == 0.0 { String::new() } else { bill.to_string() }} oninput={on_bill} />
                        </div>
                    </label>

                    <div class="bs-field">
                        <span class="bs-label">{ "Tip" }</span>
                        <div class="bs-presets">
                            { for PRESETS.iter().map(|&p| {
                                let tip = tip.clone();
                                let cls = if *tip == p as f64 { "bs-preset bs-preset--active" } else { "bs-preset" };
                                html! { <button class={cls} onclick={Callback::from(move |_| tip.set(p as f64))}>{ format!("{}%", p) }</button> }
                            }) }
                            <input class="bs-preset-custom" type="number" min="0" placeholder="Custom %"
                                value={if *tip == 0.0 { String::new() } else { tip.to_string() }} oninput={on_tip} />
                        </div>
                    </div>

                    <div class="bs-field">
                        <span class="bs-label">{ "People" }</span>
                        <div class="bs-stepper">
                            <button class="bs-step" aria-label="Fewer people" disabled={*people <= 1}
                                onclick={{ let p = people.clone(); Callback::from(move |_| p.set((*p - 1).max(1))) }}>{ "−" }</button>
                            <span class="bs-count">{ *people }</span>
                            <button class="bs-step" aria-label="More people"
                                onclick={{ let p = people.clone(); Callback::from(move |_| p.set(*p + 1)) }}>{ "+" }</button>
                        </div>
                    </div>
                </section>

                <section class="bs-card bs-results">
                    <div class="bs-result-row"><span>{ "Tip" }</span><span>{ usd(tip_amount) }</span></div>
                    <div class="bs-result-row"><span>{ "Total" }</span><span>{ usd(total) }</span></div>
                    <div class="bs-result-hero">
                        <span class="bs-result-hero-label">{ "Per person" }</span>
                        <span class="bs-result-hero-value">{ usd(per_person) }</span>
                    </div>
                    { if round && rounding_extra > 0.0 {
                        html! { <p class="bs-note">{ format!("Rounding up collects {} extra · effective tip {:.1}%", usd(rounding_extra), eff_tip) }</p> }
                    } else { Html::default() } }
                    { if *bill <= 0.0 { html! { <p class="bs-empty">{ "Enter a bill amount to begin." }</p> } } else { Html::default() } }
                </section>

                <section class="bs-card bs-breakdown">
                    <h2 class="bs-subtitle">{ "Per-person breakdown" }</h2>
                    <ul class="bs-people">
                        { for (1..=*people).map(|n| html! {
                            <li class="bs-person"><span>{ format!("Person {}", n) }</span><span>{ usd(per_person) }</span></li>
                        }) }
                    </ul>
                </section>

                <footer class="bs-footer">
                    { format!("Splitting {} between {} · {} theme", usd(total), *people, settings.theme) }
                </footer>
            </main>
        </ContextProvider<SettingsCtx>>
    }
}

fn main() {
    yew::Renderer::<App>::new().render();
}
