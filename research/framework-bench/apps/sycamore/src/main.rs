// Bill Splitter — Sycamore (Rust → wasm). create_signal for local; shared theme +
// round_up via provide_context/use_context. Run with `trunk serve`.
// Status: to-spec scaffold, not build-verified here.
use sycamore::prelude::*;
use wasm_bindgen::JsCast;
use web_sys::HtmlInputElement;

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
fn target_value(e: web_sys::Event) -> String {
    e.target().unwrap().unchecked_into::<HtmlInputElement>().value()
}

#[derive(Clone, Copy)]
struct Settings {
    theme: Signal<&'static str>,
    round_up: Signal<bool>,
}

#[component]
fn Header<G: Html>() -> View<G> {
    let s = use_context::<Settings>();
    view! {
        header(class="bs-header") {
            h1(class="bs-title") { "Bill Splitter" }
            div(class="bs-header-actions") {
                button(class="bs-toggle", "aria-pressed"=s.round_up.get().to_string(),
                    on:click=move |_| s.round_up.set(!s.round_up.get())) { "Round up" }
                button(class="bs-toggle", "aria-pressed"=(s.theme.get() == "dark").to_string(),
                    on:click=move |_| s.theme.set(if s.theme.get() == "dark" { "light" } else { "dark" })) { "Dark" }
            }
        }
    }
}

#[component]
fn App<G: Html>() -> View<G> {
    let settings = Settings {
        theme: create_signal("light"),
        round_up: create_signal(false),
    };
    provide_context(settings);

    let bill = create_signal(0.0_f64);
    let tip = create_signal(18.0_f64);
    let people = create_signal(1_i32);

    let tip_amount = create_memo(move || bill.get() * tip.get() / 100.0);
    let total = create_memo(move || bill.get() + tip_amount.get());
    let per_person = create_memo(move || {
        let raw = if people.get() > 0 { total.get() / people.get() as f64 } else { 0.0 };
        if settings.round_up.get() { raw.ceil() } else { raw }
    });
    let total_collected = create_memo(move || if settings.round_up.get() { per_person.get() * people.get() as f64 } else { total.get() });
    let rounding_extra = create_memo(move || (total_collected.get() - total.get()).max(0.0));
    let eff_tip = create_memo(move || if bill.get() > 0.0 { (total_collected.get() - bill.get()) / bill.get() * 100.0 } else { tip.get() });

    // list= sources for Indexed (Sycamore 0.9): a static signal for presets, a
    // derived memo for the per-person rows.
    let presets = create_signal(PRESETS.to_vec());
    let people_rows = create_memo(move || (1..=people.get()).collect::<Vec<i32>>());

    view! {
        main(class="bs-app", "data-theme"=settings.theme.get()) {
            Header() {}

            section(class="bs-card bs-inputs") {
                label(class="bs-field") {
                    span(class="bs-label") { "Bill amount" }
                    div(class="bs-input-wrap") {
                        span(class="bs-prefix") { "$" }
                        input(class="bs-input", r#type="number", min="0", step="0.01",
                            value=(if bill.get() == 0.0 { String::new() } else { bill.get().to_string() }),
                            on:input=move |e| bill.set(parse_num(&target_value(e))))
                    }
                }

                div(class="bs-field") {
                    span(class="bs-label") { "Tip" }
                    div(class="bs-presets") {
                        Indexed(
                            list=presets,
                            view=move |p| view! {
                                button(
                                    class=move || if tip.get() == p as f64 { "bs-preset bs-preset--active" } else { "bs-preset" },
                                    on:click=move |_| tip.set(p as f64),
                                ) { (format!("{}%", p)) }
                            },
                        )
                        input(class="bs-preset-custom", r#type="number", min="0", placeholder="Custom %",
                            value=(if tip.get() == 0.0 { String::new() } else { tip.get().to_string() }),
                            on:input=move |e| tip.set(parse_num(&target_value(e))))
                    }
                }

                div(class="bs-field") {
                    span(class="bs-label") { "People" }
                    div(class="bs-stepper") {
                        button(class="bs-step", "aria-label"="Fewer people", disabled=people.get() <= 1,
                            on:click=move |_| people.set((people.get() - 1).max(1))) { "−" }
                        span(class="bs-count") { (people.get()) }
                        button(class="bs-step", "aria-label"="More people",
                            on:click=move |_| people.set(people.get() + 1)) { "+" }
                    }
                }
            }

            section(class="bs-card bs-results") {
                div(class="bs-result-row") { span { "Tip" } span { (usd(tip_amount.get())) } }
                div(class="bs-result-row") { span { "Total" } span { (usd(total.get())) } }
                div(class="bs-result-hero") {
                    span(class="bs-result-hero-label") { "Per person" }
                    span(class="bs-result-hero-value") { (usd(per_person.get())) }
                }
                (if settings.round_up.get() && rounding_extra.get() > 0.0 {
                    view! { p(class="bs-note") { (format!("Rounding up collects {} extra · effective tip {:.1}%", usd(rounding_extra.get()), eff_tip.get())) } }
                } else { view! {} })
                (if bill.get() <= 0.0 { view! { p(class="bs-empty") { "Enter a bill amount to begin." } } } else { view! {} })
            }

            section(class="bs-card bs-breakdown") {
                h2(class="bs-subtitle") { "Per-person breakdown" }
                ul(class="bs-people") {
                    Indexed(
                        list=people_rows,
                        view=move |n| view! {
                            li(class="bs-person") { span { (format!("Person {}", n)) } span { (usd(per_person.get())) } }
                        },
                    )
                }
            }

            footer(class="bs-footer") {
                (format!("Splitting {} between {} · {} theme", usd(total.get()), people.get(), settings.theme.get()))
            }
        }
    }
}

fn main() {
    sycamore::render(App);
}
