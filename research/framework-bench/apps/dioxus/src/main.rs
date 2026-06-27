// Bill Splitter — Dioxus (Rust → wasm). use_signal for local; shared theme +
// round_up via context. Run with `dx serve`. Link ../../shared/styles.css in the
// generated index (or copy to assets). Status: to-spec scaffold, not build-verified.
use dioxus::prelude::*;

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

#[derive(Clone, Copy)]
struct Settings {
    theme: Signal<&'static str>,
    round_up: Signal<bool>,
}

fn main() {
    launch(App);
}

#[component]
fn App() -> Element {
    // shared state via context
    let settings = use_context_provider(|| Settings {
        theme: Signal::new("light"),
        round_up: Signal::new(false),
    });
    // local state
    let mut bill = use_signal(|| 0.0_f64);
    let mut tip = use_signal(|| 18.0_f64);
    let mut people = use_signal(|| 1_i32);

    let round = (settings.round_up)();
    let tip_amount = bill() * tip() / 100.0;
    let total = bill() + tip_amount;
    let per_person_raw = if people() > 0 { total / people() as f64 } else { 0.0 };
    let per_person = if round { per_person_raw.ceil() } else { per_person_raw };
    let total_collected = if round { per_person * people() as f64 } else { total };
    let rounding_extra = (total_collected - total).max(0.0);
    let eff_tip = if bill() > 0.0 { (total_collected - bill()) / bill() * 100.0 } else { tip() };

    rsx! {
        main { class: "bs-app", "data-theme": (settings.theme)(),
            Header {}

            section { class: "bs-card bs-inputs",
                label { class: "bs-field",
                    span { class: "bs-label", "Bill amount" }
                    div { class: "bs-input-wrap",
                        span { class: "bs-prefix", "$" }
                        input { class: "bs-input", r#type: "number", min: "0", step: "0.01",
                            value: if bill() == 0.0 { String::new() } else { bill().to_string() },
                            oninput: move |e| bill.set(parse_num(&e.value())) }
                    }
                }

                div { class: "bs-field",
                    span { class: "bs-label", "Tip" }
                    div { class: "bs-presets",
                        for p in PRESETS {
                            button {
                                class: if tip() == p as f64 { "bs-preset bs-preset--active" } else { "bs-preset" },
                                onclick: move |_| tip.set(p as f64),
                                "{p}%"
                            }
                        }
                        input { class: "bs-preset-custom", r#type: "number", min: "0", placeholder: "Custom %",
                            value: if tip() == 0.0 { String::new() } else { tip().to_string() },
                            oninput: move |e| tip.set(parse_num(&e.value())) }
                    }
                }

                div { class: "bs-field",
                    span { class: "bs-label", "People" }
                    div { class: "bs-stepper",
                        button { class: "bs-step", "aria-label": "Fewer people", disabled: people() <= 1,
                            onclick: move |_| people.set((people() - 1).max(1)), "−" }
                        span { class: "bs-count", "{people}" }
                        button { class: "bs-step", "aria-label": "More people",
                            onclick: move |_| people.set(people() + 1), "+" }
                    }
                }
            }

            section { class: "bs-card bs-results",
                div { class: "bs-result-row", span { "Tip" } span { "{usd(tip_amount)}" } }
                div { class: "bs-result-row", span { "Total" } span { "{usd(total)}" } }
                div { class: "bs-result-hero",
                    span { class: "bs-result-hero-label", "Per person" }
                    span { class: "bs-result-hero-value", "{usd(per_person)}" }
                }
                if round && rounding_extra > 0.0 {
                    p { class: "bs-note", "Rounding up collects {usd(rounding_extra)} extra · effective tip {eff_tip:.1}%" }
                }
                if bill() <= 0.0 {
                    p { class: "bs-empty", "Enter a bill amount to begin." }
                }
            }

            section { class: "bs-card bs-breakdown",
                h2 { class: "bs-subtitle", "Per-person breakdown" }
                ul { class: "bs-people",
                    for n in 1..=people() {
                        li { key: "{n}", class: "bs-person", span { "Person {n}" } span { "{usd(per_person)}" } }
                    }
                }
            }

            footer { class: "bs-footer", "Splitting {usd(total)} between {people} · {(settings.theme)()} theme" }
        }
    }
}

#[component]
fn Header() -> Element {
    let mut settings = use_context::<Settings>();
    rsx! {
        header { class: "bs-header",
            h1 { class: "bs-title", "Bill Splitter" }
            div { class: "bs-header-actions",
                button { class: "bs-toggle", "aria-pressed": "{(settings.round_up)()}",
                    onclick: move |_| { let v = (settings.round_up)(); settings.round_up.set(!v); }, "Round up" }
                button { class: "bs-toggle", "aria-pressed": "{(settings.theme)() == \"dark\"}",
                    onclick: move |_| {
                        let t = (settings.theme)();
                        settings.theme.set(if t == "dark" { "light" } else { "dark" });
                    }, "Dark" }
            }
        }
    }
}
