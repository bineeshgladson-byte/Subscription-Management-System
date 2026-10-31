import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/workspace_theme.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../discounts/data/discounts_api.dart';
import '../../products/data/products_api.dart';
import '../../products/presentation/line_editor.dart';

/// Price calculator (admin + staff): POST /pricing/preview with real products,
/// taxes and today's discounts. Nothing is saved.
class PricingScreen extends ConsumerStatefulWidget {
  const PricingScreen({super.key});
  @override
  ConsumerState<PricingScreen> createState() => _PricingScreenState();
}

class _PricingScreenState extends ConsumerState<PricingScreen> {
  final _form = GlobalKey<FormState>();
  final _lines = [LineDraft()];
  final _date = TextEditingController(text: Fmt.apiDate(DateTime.now()));
  PricePreview? _result;
  String? _error;
  bool _busy = false;

  @override
  void dispose() {
    for (final l in _lines) {
      l.dispose();
    }
    _date.dispose();
    super.dispose();
  }

  Future<void> _calculate() async {
    if (!_form.currentState!.validate()) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final r = await ref.read(discountsRepositoryProvider).preview([
        for (final l in _lines)
          ?l.toJson(withOverride: true, overrideKey: 'unit_price'),
      ], date: _date.text.trim());
      setState(() => _result = r);
    } catch (e) {
      setState(() => _error = errorText(e));
    } finally {
      setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => PageFrame(
    title: 'Price calculator',
    subtitle:
        'See exactly what a customer would pay: catalogue prices, taxes and the discounts that apply on a date.',
    child: AsyncView(
      value: ref.watch(activeProductsProvider),
      onRetry: () => ref.invalidate(activeProductsProvider),
      builder: (products) => Form(
        key: _form,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            LineEditor(
              products: products,
              lines: _lines,
              onChanged: () => setState(() {}),
              allowOverride: true,
              overrideLabel: 'Unit price',
              errorKey: 'lines',
            ),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: [
                SizedBox(
                  width: 220,
                  child: DateField(
                    controller: _date,
                    label: 'Price on',
                    required: true,
                  ),
                ),
                FilledButton.icon(
                  onPressed: _busy ? null : _calculate,
                  icon: const Icon(Icons.calculate_outlined),
                  label: Text(_busy ? 'Calculating…' : 'Calculate'),
                ),
              ],
            ),
            if (_error != null)
              Padding(
                padding: const EdgeInsets.only(top: 12),
                child: Text(
                  _error!,
                  style: const TextStyle(color: Color(0xFFB3261E)),
                ),
              ),
            if (_result != null) ...[
              const SizedBox(height: 20),
              _Breakdown(result: _result!),
            ],
          ],
        ),
      ),
    ),
  );
}

class _Breakdown extends StatelessWidget {
  const _Breakdown({required this.result});
  final PricePreview result;
  @override
  Widget build(BuildContext context) {
    final r = result;
    Widget row(String label, String value, {bool bold = false}) => Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        children: [
          Expanded(
            child: Text(
              label,
              style: TextStyle(
                fontWeight: bold ? FontWeight.w600 : FontWeight.w400,
              ),
            ),
          ),
          Text(
            value,
            style: TextStyle(
              fontWeight: bold ? FontWeight.w600 : FontWeight.w400,
              fontFeatures: const [FontFeature.tabularFigures()],
            ),
          ),
        ],
      ),
    );
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Breakdown for ${Fmt.date(r.date)}',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 12),
            for (final l in r.lines) ...[
              Text(
                '${l.quantity} × ${l.description} @ ${Fmt.inr(l.unitPrice)}',
                style: const TextStyle(fontWeight: FontWeight.w500),
              ),
              row('  Subtotal', Fmt.inr(l.subtotal)),
              if (Fmt.money(l.productDiscount) > Fmt.money('0'))
                row(
                  '  ${l.productDiscountName ?? 'Product discount'}',
                  '−${Fmt.inr(l.productDiscount)}',
                ),
              if (Fmt.money(l.subscriptionDiscount) > Fmt.money('0'))
                row(
                  '  Order discount share',
                  '−${Fmt.inr(l.subscriptionDiscount)}',
                ),
              row('  Tax ${l.taxRate}%', Fmt.inr(l.taxAmount)),
              const Divider(height: 18),
            ],
            row('Subtotal', Fmt.inr(r.subtotal)),
            row('Discounts', '−${Fmt.inr(r.discountTotal)}'),
            row('Tax', Fmt.inr(r.taxTotal)),
            const Divider(height: 18),
            row('Total', Fmt.inr(r.total), bold: true),
            const SizedBox(height: 10),
            if (r.discounts.isEmpty)
              Text(
                'No discounts apply to this order on this date.',
                style: Theme.of(context).textTheme.bodySmall,
              )
            else
              Wrap(
                spacing: 6,
                runSpacing: 6,
                children: [
                  for (final d in r.discounts)
                    Pill('${d.name}: −${Fmt.inr(d.amount)}', tone: 'good'),
                ],
              ),
            const SizedBox(height: 4),
            Text(
              'Prices from the live catalogue · nothing is saved.',
              style: const TextStyle(fontSize: 11, color: WorkspaceTheme.muted),
            ),
          ],
        ),
      ),
    );
  }
}
