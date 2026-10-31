import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../products/data/products_api.dart';
import '../data/discounts_api.dart';

/// Discount rules (admin only). Eligible discounts apply automatically when a
/// price is calculated; they stop after their end date or usage limit.
class DiscountsScreen extends ConsumerStatefulWidget {
  const DiscountsScreen({super.key});
  @override
  ConsumerState<DiscountsScreen> createState() => _DiscountsScreenState();
}

class _DiscountsScreenState extends ConsumerState<DiscountsScreen> {
  DiscountQuery _q = (q: '', state: null, page: 1);

  static String tone(String state) => switch (state) {
    'active' => 'good',
    'scheduled' => 'brand',
    'expired' || 'used_up' => 'warn',
    _ => 'neutral',
  };

  @override
  Widget build(BuildContext context) {
    return PageFrame(
      title: 'Discounts',
      subtitle:
          'Rules applied automatically: the best product discount per line, then the best order-wide discount.',
      action: FilledButton.icon(
        onPressed: () => _edit(null),
        icon: const Icon(Icons.add),
        label: const Text('New discount'),
      ),
      toolbar: Toolbar(
        children: [
          SearchBox(
            hint: 'Search name or code',
            onChanged: (v) =>
                setState(() => _q = (q: v, state: _q.state, page: 1)),
          ),
          FilterDropdown<String>(
            label: 'State',
            value: _q.state,
            options: discountStates,
            onChanged: (v) => setState(() => _q = (q: _q.q, state: v, page: 1)),
          ),
        ],
      ),
      child: AsyncView(
        value: ref.watch(discountsProvider(_q)),
        onRetry: () => ref.invalidate(discountsProvider),
        builder: (page) => Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (page.items.isEmpty)
              const EmptyState(
                icon: Icons.local_offer_outlined,
                title: 'No discounts',
                message: 'Create a rule, e.g. 10% off orders over ₹10,000.',
              ),
            for (final d in page.items)
              ResourceTile(
                leading: CircleAvatar(
                  child: Icon(
                    d.scope == 'product'
                        ? Icons.sell_outlined
                        : Icons.receipt_long_outlined,
                    size: 18,
                  ),
                ),
                title: '${d.name}${d.code == null ? '' : ' · ${d.code}'}',
                subtitle: [
                  d.label,
                  d.scope == 'product'
                      ? 'on ${d.productNames.join(', ')}'
                      : 'on the whole order',
                  if (d.minPurchaseAmount != null)
                    'orders over ${Fmt.inr(d.minPurchaseAmount!)}',
                  if (d.minQuantity != null) '${d.minQuantity}+ units',
                  'from ${Fmt.date(d.startsOn)}${d.endsOn == null ? '' : ' to ${Fmt.date(d.endsOn!)}'}',
                ].join(' · '),
                pills: [
                  Pill(discountStates[d.state] ?? d.state, tone: tone(d.state)),
                  Pill(
                    d.maxUsage == null
                        ? 'Used ${d.usageCount}× · no limit'
                        : 'Used ${d.usageCount} of ${d.maxUsage}',
                  ),
                ],
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    IconButton(
                      tooltip: 'Edit ${d.name}',
                      icon: const Icon(Icons.edit_outlined),
                      onPressed: () => _edit(d),
                    ),
                    Switch(
                      value: d.isActive,
                      onChanged: (v) async {
                        final repo = ref.read(discountsRepositoryProvider);
                        if (await runAction(
                          context,
                          () => v
                              ? repo.update(d.id, {'is_active': true})
                              : repo.deactivate(d.id),
                          success: v
                              ? 'Discount switched on'
                              : 'Discount switched off',
                        )) {
                          ref.invalidate(discountsProvider);
                        }
                      },
                    ),
                  ],
                ),
              ),
            Pager(
              page: page.page,
              pageSize: page.pageSize,
              total: page.total,
              onPage: (n) =>
                  setState(() => _q = (q: _q.q, state: _q.state, page: n)),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _edit(Discount? d) async {
    final products = await ref
        .read(activeProductsProvider.future)
        .catchError((_) => <Product>[]);
    if (!mounted) return;
    final name = TextEditingController(text: d?.name);
    final code = TextEditingController(text: d?.code);
    final value = TextEditingController(text: d?.value);
    final minPurchase = TextEditingController(text: d?.minPurchaseAmount ?? '');
    final minQty = TextEditingController(
      text: d?.minQuantity?.toString() ?? '',
    );
    final starts = TextEditingController(
      text: d?.startsOn ?? Fmt.apiDate(DateTime.now()),
    );
    final ends = TextEditingController(text: d?.endsOn ?? '');
    final maxUsage = TextEditingController(text: d?.maxUsage?.toString() ?? '');
    var type = d?.discountType ?? 'percentage';
    var scope = d?.scope ?? 'subscription';
    final chosen = {...?d?.productIds};
    final repo = ref.read(discountsRepositoryProvider);
    final saved = await showApiForm<bool>(
      context,
      title: d == null ? 'New discount rule' : 'Edit ${d.name}',
      width: 620,
      fields: (_, setState, e) => [
        formField(
          name,
          'Name',
          errors: e,
          errorKey: 'name',
          hint: '10% off orders over ₹10,000',
          validator: requiredValidator,
        ),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            SizedBox(
              width: 170,
              child: DropdownButtonFormField<String>(
                initialValue: type,
                decoration: const InputDecoration(labelText: 'Type'),
                items: const [
                  DropdownMenuItem(
                    value: 'percentage',
                    child: Text('Percentage'),
                  ),
                  DropdownMenuItem(value: 'fixed', child: Text('Fixed amount')),
                ],
                onChanged: (v) => setState(() => type = v ?? type),
              ),
            ),
            SizedBox(
              width: 150,
              child: formField(
                value,
                type == 'percentage' ? 'Percent off' : 'Amount off (₹)',
                errors: e,
                errorKey: 'value',
                validator: moneyValidator(),
              ),
            ),
            SizedBox(
              width: 160,
              child: formField(
                code,
                'Code (optional)',
                errors: e,
                errorKey: 'code',
              ),
            ),
          ],
        ),
        SegmentedButton<String>(
          segments: const [
            ButtonSegment(
              value: 'subscription',
              label: Text('Whole order'),
              icon: Icon(Icons.receipt_long_outlined),
            ),
            ButtonSegment(
              value: 'product',
              label: Text('Specific products'),
              icon: Icon(Icons.sell_outlined),
            ),
          ],
          selected: {scope},
          onSelectionChanged: (s) => setState(() => scope = s.first),
        ),
        if (scope == 'product') ...[
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: [
              for (final p in products)
                FilterChip(
                  label: Text(p.name),
                  selected: chosen.contains(p.id),
                  onSelected: (v) => setState(
                    () => v ? chosen.add(p.id) : chosen.remove(p.id),
                  ),
                ),
            ],
          ),
          if (fieldError(e, 'product_ids') != null)
            Text(
              fieldError(e, 'product_ids')!,
              style: const TextStyle(color: Color(0xFFB3261E), fontSize: 12),
            ),
        ],
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            SizedBox(
              width: 190,
              child: formField(
                minPurchase,
                'Minimum purchase (₹)',
                errors: e,
                errorKey: 'min_purchase_amount',
                validator: moneyValidator(required: false),
              ),
            ),
            SizedBox(
              width: 160,
              child: formField(
                minQty,
                'Minimum quantity',
                errors: e,
                errorKey: 'min_quantity',
                keyboard: TextInputType.number,
                validator: (v) => intValidator(v, required: false),
              ),
            ),
            SizedBox(
              width: 160,
              child: formField(
                maxUsage,
                'Max uses',
                errors: e,
                errorKey: 'max_usage',
                keyboard: TextInputType.number,
                validator: (v) => intValidator(v, required: false),
              ),
            ),
          ],
        ),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            SizedBox(
              width: 220,
              child: DateField(
                controller: starts,
                label: 'Starts on',
                required: true,
                errorText: fieldError(e, 'starts_on'),
              ),
            ),
            SizedBox(
              width: 220,
              child: DateField(
                controller: ends,
                label: 'Ends on (optional)',
                errorText: fieldError(e, 'ends_on'),
              ),
            ),
          ],
        ),
      ],
      onSave: () async {
        final body = <String, dynamic>{
          'name': name.text.trim(),
          'code': code.text.trim(),
          'discount_type': type,
          'value': value.text.trim(),
          'scope': scope,
          'product_ids': scope == 'product' ? chosen.toList() : <int>[],
          'min_purchase_amount': minPurchase.text.trim(),
          'min_quantity': int.tryParse(minQty.text.trim()) ?? 0,
          'max_usage': int.tryParse(maxUsage.text.trim()) ?? 0,
          'starts_on': starts.text.trim(),
          'ends_on': ends.text.trim(),
        };
        if (d == null) {
          body.removeWhere((k, v) => v == '' || (v == 0 && k != 'value'));
          if (scope != 'product') body.remove('product_ids');
          await repo.create(body);
        } else {
          await repo.update(d.id, body);
        }
        return true;
      },
    );
    for (final c in [
      name,
      code,
      value,
      minPurchase,
      minQty,
      starts,
      ends,
      maxUsage,
    ]) {
      c.dispose();
    }
    if (saved == true && mounted) {
      ref.invalidate(discountsProvider);
      showMessage(context, d == null ? 'Discount created' : 'Discount updated');
    }
  }
}
