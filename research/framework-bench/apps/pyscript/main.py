# Bill Splitter — PyScript (Python in the browser via Pyodide).
#
# PyScript has no reactive UI framework: you manipulate the DOM imperatively. This
# keeps a module-level state dict and re-renders #app on each change, with one
# delegated click/input listener (so handlers survive innerHTML replacement). That
# imperative model — and re-render focus loss on text inputs — is the comparison
# point for PyScript. Status: to-spec scaffold, not build-verified here.
import math
from pyscript import document
from pyscript.ffi import create_proxy

state = {"bill": 0.0, "tip": 18.0, "people": 1, "theme": "light", "roundUp": False}
PRESETS = [10, 15, 18, 20, 25]


def parse_num(s):
    try:
        v = float(s)
        return v if v >= 0 else 0.0
    except (TypeError, ValueError):
        return 0.0


def num_str(v):
    return "" if v == 0 else (str(int(v)) if v == int(v) else str(v))


def usd(v):
    cents = max(0, round(v * 100))
    dollars, frac = divmod(cents, 100)
    return "${:,}.{:02d}".format(dollars, frac)


def render():
    s = state
    tip_amount = s["bill"] * s["tip"] / 100
    total = s["bill"] + tip_amount
    per_person_raw = total / s["people"] if s["people"] > 0 else 0
    per_person = math.ceil(per_person_raw) if s["roundUp"] else per_person_raw
    total_collected = per_person * s["people"] if s["roundUp"] else total
    rounding_extra = max(0, total_collected - total)
    eff_tip = (total_collected - s["bill"]) / s["bill"] * 100 if s["bill"] > 0 else s["tip"]

    presets_html = "".join(
        '<button class="bs-preset{}" data-act="preset" data-p="{}">{}%</button>'.format(
            " bs-preset--active" if s["tip"] == p else "", p, p
        )
        for p in PRESETS
    )
    people_html = "".join(
        '<li class="bs-person"><span>Person {}</span><span>{}</span></li>'.format(n, usd(per_person))
        for n in range(1, s["people"] + 1)
    )
    note_html = (
        '<p class="bs-note">Rounding up collects {} extra · effective tip {:.1f}%</p>'.format(usd(rounding_extra), eff_tip)
        if s["roundUp"] and rounding_extra > 0
        else ""
    )
    empty_html = '<p class="bs-empty">Enter a bill amount to begin.</p>' if s["bill"] <= 0 else ""

    app = document.querySelector("#app")
    app.innerHTML = """
    <main class="bs-app" data-theme="{theme}">
      <header class="bs-header">
        <h1 class="bs-title">Bill Splitter</h1>
        <div class="bs-header-actions">
          <button class="bs-toggle" aria-pressed="{round}" data-act="toggle-round">Round up</button>
          <button class="bs-toggle" aria-pressed="{dark}" data-act="toggle-theme">Dark</button>
        </div>
      </header>
      <section class="bs-card bs-inputs">
        <label class="bs-field"><span class="bs-label">Bill amount</span>
          <div class="bs-input-wrap"><span class="bs-prefix">$</span>
            <input class="bs-input" type="number" min="0" step="0.01" value="{billv}" data-act="bill" /></div>
        </label>
        <div class="bs-field"><span class="bs-label">Tip</span>
          <div class="bs-presets">{presets}
            <input class="bs-preset-custom" type="number" min="0" placeholder="Custom %" value="{tipv}" data-act="tip" /></div>
        </div>
        <div class="bs-field"><span class="bs-label">People</span>
          <div class="bs-stepper">
            <button class="bs-step" aria-label="Fewer people" data-act="dec"{decdis}>−</button>
            <span class="bs-count">{people}</span>
            <button class="bs-step" aria-label="More people" data-act="inc">+</button></div>
        </div>
      </section>
      <section class="bs-card bs-results">
        <div class="bs-result-row"><span>Tip</span><span>{tipamt}</span></div>
        <div class="bs-result-row"><span>Total</span><span>{total}</span></div>
        <div class="bs-result-hero"><span class="bs-result-hero-label">Per person</span>
          <span class="bs-result-hero-value">{pp}</span></div>
        {note}{empty}
      </section>
      <section class="bs-card bs-breakdown">
        <h2 class="bs-subtitle">Per-person breakdown</h2>
        <ul class="bs-people">{peoplelist}</ul>
      </section>
      <footer class="bs-footer">Splitting {total} between {people} · {theme} theme</footer>
    </main>
    """.format(
        theme=s["theme"],
        round="true" if s["roundUp"] else "false",
        dark="true" if s["theme"] == "dark" else "false",
        billv=num_str(s["bill"]),
        tipv=num_str(s["tip"]),
        presets=presets_html,
        decdis=" disabled" if s["people"] <= 1 else "",
        people=s["people"],
        tipamt=usd(tip_amount),
        total=usd(total),
        pp=usd(per_person),
        note=note_html,
        empty=empty_html,
        peoplelist=people_html,
    )


def on_click(event):
    act = event.target.dataset.act
    if act == "toggle-theme":
        state["theme"] = "light" if state["theme"] == "dark" else "dark"
    elif act == "toggle-round":
        state["roundUp"] = not state["roundUp"]
    elif act == "inc":
        state["people"] += 1
    elif act == "dec":
        state["people"] = max(1, state["people"] - 1)
    elif act == "preset":
        state["tip"] = float(event.target.dataset.p)
    else:
        return
    render()


def on_input(event):
    act = event.target.dataset.act
    if act == "bill":
        state["bill"] = parse_num(event.target.value)
    elif act == "tip":
        state["tip"] = parse_num(event.target.value)
    else:
        return
    render()


app = document.querySelector("#app")
app.addEventListener("click", create_proxy(on_click))
app.addEventListener("input", create_proxy(on_input))
render()
