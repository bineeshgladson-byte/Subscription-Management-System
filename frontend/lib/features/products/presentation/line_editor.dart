import 'package:flutter/material.dart';

import '../../../core/utils/formatters.dart';
import '../../../core/widgets/ui_kit.dart';
import '../data/products_api.dart';

/// One editable product line (plan line, template line or price-preview line).
class LineDraft {
  LineDraft({
    this.productId,
    this.variantId,
    int quantity = 1,
    String? priceOverride,
  }) : quantity = TextEditingController(text: '$quantity'),
       priceOverride = TextEditingController(text: priceOverride ?? '');
  int? productId;
  int? variantId;
  final TextEditingController quantity;
  final TextEditingController priceOverride;

  void dispose() {
    quantity.dispose();
    priceOverride.dispose();
  }

  /// JSON for the API (null if no product chosen).
  Map<String, dynamic>? toJson({
    bool withOverride = false,
    String overrideKey = 'price_override',
  }) {
    if (productId == null) return null;
    return {
      'product_id': productId,
      'variant_id': ?variantId,
      'quantity': int.tryParse(quantity.text.trim()) ?? 1,
      if (withOverride && priceOverride.text.trim().isNotEmpty)
        overrideKey: priceOverride.text.trim(),
    };
  }
}

/// Edits a list of [LineDraft]s against the real product catalogue.
class LineEditor extends StatelessWidget {
  const LineEditor({
    super.key,
    required this.products,
    required this.lines,
    required this.onChanged,
    this.allowOverride = false,
    this.overrideLabel = 'Price override',
    this.errors = const {},
    this.errorKey = 'products',
  });
  final List<Product> products;
  final List<LineDraft> lines;
  final VoidCallback onChanged;
  final bool allowOverride;
  final String overrideLabel;
  final Map<String, String> errors;
  final String errorKey;

  @override
  Widget build(BuildContext context) {
    if (products.isEmpty) {
      return const Text(
        'No active products yet. Create a product first.',
        style: TextStyle(color: Color(0xFFB3261E)),
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text('Products', style: Theme.of(context).textTheme.titleSmall),
        const SizedBox(height: 8),
        for (final (i, line) in lines.indexed) _row(context, i, line),
        if (fieldError(errors, errorKey) != null)
          Text(
            fieldError(errors, errorKey)!,
            style: const TextStyle(color: Color(0xFFB3261E), fontSize: 12),
          ),
        Align(
          alignment: Alignment.centerLeft,
          child: TextButton.icon(
            onPressed: () {
              lines.add(LineDraft());
              onChanged();
            },
            icon: const Icon(Icons.add, size: 18),
            label: const Text('Add product'),
          ),
        ),
      ],
    );
  }

  Widget _row(BuildContext context, int i, LineDraft line) {
    final product = products.where((p) => p.id == line.productId).firstOrNull;
    final variants = product?.activeVariants ?? const <Variant>[];
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Wrap(
        spacing: 8,
        runSpacing: 8,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          SizedBox(
            width: 210,
            child: DropdownButtonFormField<int>(
              initialValue: line.productId,
              isExpanded: true,
              decoration: InputDecoration(
                labelText: 'Product',
                errorText: fieldError(errors, '$errorKey[$i].product_id'),
              ),
              items: [
                for (final p in products)
                  DropdownMenuItem(
                    value: p.id,
                    child: Text(
                      '${p.name} · ${Fmt.inr(p.salesPrice)}',
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
              ],
              validator: (v) => v == null ? 'Choose a product' : null,
              onChanged: (v) {
                line.productId = v;
                line.variantId = null;
                onChanged();
              },
            ),
          ),
          if (variants.isNotEmpty)
            SizedBox(
              width: 170,
              child: DropdownButtonFormField<int?>(
                key: ValueKey('variant-${line.productId}'),
                initialValue: line.variantId,
                isExpanded: true,
                decoration: InputDecoration(
                  labelText: 'Variant',
                  errorText: fieldError(errors, '$errorKey[$i].variant_id'),
                ),
                items: [
                  const DropdownMenuItem<int?>(
                    value: null,
                    child: Text('Base product'),
                  ),
                  for (final v in variants)
                    DropdownMenuItem<int?>(
                      value: v.id,
                      child: Text(
                        '${v.name} · ${Fmt.inr(v.price)}',
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                ],
                onChanged: (v) {
                  line.variantId = v;
                  onChanged();
                },
              ),
            ),
          SizedBox(
            width: 80,
            child: TextFormField(
              controller: line.quantity,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(labelText: 'Qty'),
              validator: (v) => intValidator(v),
            ),
          ),
          if (allowOverride)
            SizedBox(
              width: 130,
              child: TextFormField(
                controller: line.priceOverride,
                keyboardType: const TextInputType.numberWithOptions(
                  decimal: true,
                ),
                decoration: InputDecoration(
                  labelText: overrideLabel,
                  errorText: fieldError(errors, '$errorKey[$i].price_override'),
                ),
                validator: moneyValidator(required: false),
              ),
            ),
          IconButton(
            tooltip: 'Remove line',
            onPressed: lines.length > 1
                ? () {
                    lines.removeAt(i).dispose();
                    onChanged();
                  }
                : null,
            icon: const Icon(Icons.remove_circle_outline),
          ),
        ],
      ),
    );
  }
}
