// Bill Splitter — Flutter web (Dart).
//
// PARADIGM CAVEAT: Flutter renders to a canvas, not the DOM. It cannot consume the
// shared `bs-*` markup or styles.css, so this reproduces the SPEC's *visual design*
// with widgets and the same design tokens. DOM/class parity is impossible by
// construction — that divergence is the comparison point for Flutter.
//
// Local state via setState; shared theme + roundUp via a ChangeNotifier consumed by
// multiple widgets. Run with `flutter run -d chrome`.
// Status: to-spec scaffold, not build-verified here.
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

final usd = NumberFormat.currency(locale: 'en_US', symbol: '\$');
const presets = [10, 15, 18, 20, 25];

// Shared/global state.
class Settings extends ChangeNotifier {
  bool dark = false;
  bool roundUp = false;
  void toggleTheme() {
    dark = !dark;
    notifyListeners();
  }
  void toggleRoundUp() {
    roundUp = !roundUp;
    notifyListeners();
  }
}

final settings = Settings();

// Design tokens (SPEC).
class Tokens {
  final Color bg, surface, text, muted, border, accent, accentText;
  const Tokens(this.bg, this.surface, this.text, this.muted, this.border, this.accent, this.accentText);
  static Tokens of(bool dark) => dark
      ? const Tokens(Color(0xFF0F172A), Color(0xFF1E293B), Color(0xFFF1F5F9), Color(0xFF94A3B8),
          Color(0xFF334155), Color(0xFF38BDF8), Color(0xFF04293C))
      : const Tokens(Color(0xFFF8FAFC), Color(0xFFFFFFFF), Color(0xFF0F172A), Color(0xFF64748B),
          Color(0xFFE2E8F0), Color(0xFF0EA5E9), Color(0xFFFFFFFF));
}

void main() => runApp(const BillSplitterApp());

class BillSplitterApp extends StatelessWidget {
  const BillSplitterApp({super.key});
  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: settings,
      builder: (context, _) => MaterialApp(
        title: 'Bill Splitter',
        debugShowCheckedModeBanner: false,
        home: const Splitter(),
      ),
    );
  }
}

class Splitter extends StatefulWidget {
  const Splitter({super.key});
  @override
  State<Splitter> createState() => _SplitterState();
}

class _SplitterState extends State<Splitter> {
  double bill = 0, tip = 18;
  int people = 1;

  double get tipAmount => bill * tip / 100;
  double get total => bill + tipAmount;
  double get perPersonRaw => people > 0 ? total / people : 0;
  double get perPerson => settings.roundUp ? perPersonRaw.ceilToDouble() : perPersonRaw;
  double get totalCollected => settings.roundUp ? perPerson * people : total;
  double get roundingExtra => (totalCollected - total).clamp(0, double.infinity);
  double get effTip => bill > 0 ? (totalCollected - bill) / bill * 100 : tip;

  @override
  void initState() {
    super.initState();
    settings.addListener(_onSettings);
  }
  @override
  void dispose() {
    settings.removeListener(_onSettings);
    super.dispose();
  }
  void _onSettings() => setState(() {});

  @override
  Widget build(BuildContext context) {
    final t = Tokens.of(settings.dark);
    return Scaffold(
      backgroundColor: t.bg,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 448),
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 24),
            child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
              _header(t),
              const SizedBox(height: 16),
              _inputsCard(t),
              const SizedBox(height: 16),
              _resultsCard(t),
              const SizedBox(height: 16),
              _breakdownCard(t),
              const SizedBox(height: 16),
              Text('Splitting ${usd.format(total)} between $people · ${settings.dark ? "dark" : "light"} theme',
                  textAlign: TextAlign.center, style: TextStyle(fontSize: 12, color: t.muted)),
            ]),
          ),
        ),
      ),
    );
  }

  Widget _header(Tokens t) => Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
        Text('Bill Splitter', style: TextStyle(fontSize: 24, fontWeight: FontWeight.w800, color: t.text)),
        Row(children: [
          _toggle('Round up', settings.roundUp, t, settings.toggleRoundUp),
          const SizedBox(width: 8),
          _toggle('Dark', settings.dark, t, settings.toggleTheme),
        ]),
      ]);

  Widget _toggle(String label, bool active, Tokens t, VoidCallback onTap) => InkWell(
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          decoration: BoxDecoration(
            color: active ? t.accent : t.surface,
            border: Border.all(color: active ? t.accent : t.border),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Text(label, style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: active ? t.accentText : t.text)),
        ),
      );

  BoxDecoration _card(Tokens t) =>
      BoxDecoration(color: t.surface, border: Border.all(color: t.border), borderRadius: BorderRadius.circular(16));

  Widget _inputsCard(Tokens t) => Container(
        decoration: _card(t),
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          _label('Bill amount', t),
          const SizedBox(height: 8),
          Container(
            decoration: BoxDecoration(color: t.bg, border: Border.all(color: t.border), borderRadius: BorderRadius.circular(10)),
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Row(children: [
              Text('\$', style: TextStyle(color: t.muted, fontWeight: FontWeight.w600)),
              const SizedBox(width: 4),
              Expanded(
                child: TextField(
                  keyboardType: TextInputType.number,
                  style: TextStyle(color: t.text),
                  decoration: const InputDecoration(border: InputBorder.none),
                  onChanged: (v) => setState(() => bill = _parse(v)),
                ),
              ),
            ]),
          ),
          const SizedBox(height: 16),
          _label('Tip', t),
          const SizedBox(height: 8),
          Wrap(spacing: 8, runSpacing: 8, children: [
            for (final p in presets)
              InkWell(
                onTap: () => setState(() => tip = p.toDouble()),
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                  decoration: BoxDecoration(
                    color: tip == p ? t.accent : t.bg,
                    border: Border.all(color: tip == p ? t.accent : t.border),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text('$p%', style: TextStyle(fontWeight: FontWeight.w600, color: tip == p ? t.accentText : t.text)),
                ),
              ),
          ]),
          const SizedBox(height: 16),
          _label('People', t),
          const SizedBox(height: 8),
          Row(children: [
            _step('−', t, people > 1 ? () => setState(() => people--) : null),
            const SizedBox(width: 12),
            Text('$people', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700, color: t.text)),
            const SizedBox(width: 12),
            _step('+', t, () => setState(() => people++)),
          ]),
        ]),
      );

  Widget _step(String label, Tokens t, VoidCallback? onTap) => InkWell(
        onTap: onTap,
        child: Opacity(
          opacity: onTap == null ? 0.4 : 1,
          child: Container(
            width: 36,
            height: 36,
            alignment: Alignment.center,
            decoration: BoxDecoration(color: t.bg, border: Border.all(color: t.border), borderRadius: BorderRadius.circular(10)),
            child: Text(label, style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700, color: t.text)),
          ),
        ),
      );

  Widget _resultsCard(Tokens t) => Container(
        decoration: _card(t),
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          _row('Tip', usd.format(tipAmount), t),
          const SizedBox(height: 16),
          _row('Total', usd.format(total), t),
          const SizedBox(height: 12),
          Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, crossAxisAlignment: CrossAxisAlignment.baseline,
              textBaseline: TextBaseline.alphabetic, children: [
            Text('Per person', style: TextStyle(color: t.muted)),
            Text(usd.format(perPerson), style: TextStyle(fontSize: 32, fontWeight: FontWeight.w800, color: t.accent)),
          ]),
          if (settings.roundUp && roundingExtra > 0) ...[
            const SizedBox(height: 8),
            Text('Rounding up collects ${usd.format(roundingExtra)} extra · effective tip ${effTip.toStringAsFixed(1)}%',
                style: TextStyle(fontSize: 13, color: t.muted)),
          ],
          if (bill <= 0) ...[
            const SizedBox(height: 8),
            Text('Enter a bill amount to begin.', style: TextStyle(fontSize: 14, color: t.muted, fontStyle: FontStyle.italic)),
          ],
        ]),
      );

  Widget _breakdownCard(Tokens t) => Container(
        decoration: _card(t),
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          Text('Per-person breakdown', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w700, color: t.text)),
          const SizedBox(height: 16),
          for (var n = 1; n <= people; n++) ...[
            Container(
              margin: EdgeInsets.only(top: n == 1 ? 0 : 8),
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(color: t.bg, border: Border.all(color: t.border), borderRadius: BorderRadius.circular(10)),
              child: Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                Text('Person $n', style: TextStyle(color: t.text)),
                Text(usd.format(perPerson), style: TextStyle(fontWeight: FontWeight.w600, color: t.text)),
              ]),
            ),
          ],
        ]),
      );

  Widget _label(String s, Tokens t) => Text(s, style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: t.muted));
  Widget _row(String a, String b, Tokens t) => Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
        Text(a, style: TextStyle(color: t.muted)),
        Text(b, style: TextStyle(fontWeight: FontWeight.w600, color: t.text)),
      ]);

  double _parse(String s) {
    final v = double.tryParse(s) ?? 0;
    return v >= 0 ? v : 0;
  }
}
