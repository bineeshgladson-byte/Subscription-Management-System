import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../customers/data/customers_api.dart';
import '../../plans/data/plans_api.dart';
import '../../products/data/products_api.dart';
import '../../products/presentation/line_editor.dart';
import '../../sales/data/sales_api.dart';
import '../../sales/presentation/sales_widgets.dart';
import '../../subscriptions/presentation/subscriptions_screen.dart';
import '../../templates/data/templates_api.dart';

/// Quotations: Draft → Quotation (sent) → Confirmed (becomes a subscription and is billed).
class QuotationsScreen extends ConsumerStatefulWidget {
  const QuotationsScreen({super.key});
  @override
  ConsumerState<QuotationsScreen> createState() => _QuotationsScreenState();
}

class _QuotationsScreenState extends ConsumerState<QuotationsScreen> {
  SubQuery _q = (q: '', status: null, page: 1);

  @override
  Widget build(BuildContext context) => PageFrame(
    title: 'Quotations',
    subtitle:
        'Offers waiting for the customer. Confirming one creates the subscription and its first invoice.',
    action: FilledButton.icon(
      onPressed: _create,
      icon: const Icon(Icons.add),
      label: const Text('New quotation'),
    ),
    toolbar: Toolbar(
      children: [
        SearchBox(
          hint: 'Search customer or number',
          onChanged: (v) =>
              setState(() => _q = (q: v, status: _q.status, page: 1)),
        ),
        FilterDropdown<String>(
          label: 'Status',
          value: _q.status,
          options: const {'draft': 'Draft', 'quotation': 'Sent'},
          onChanged: (v) => setState(() => _q = (q: _q.q, status: v, page: 1)),
        ),
      ],
    ),
    child: AsyncView(
      value: ref.watch(quotationsProvider(_q)),
      onRetry: () => ref.invalidate(quotationsProvider),
      builder: (page) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (page.items.isEmpty)
            const EmptyState(
              icon: Icons.request_quote_outlined,
              title: 'No open quotations',
              message: 'Create one from a plan or template.',
            ),
          for (final s in page.items)
            ResourceTile(
              title: '${s.quotationNumber} · ${s.customerName}',
              subtitle:
                  '${s.planName} (${s.billingLabel}) · ${Fmt.inr(s.total)} per period incl. tax · starts ${Fmt.date(s.startDate)}'
                  '${s.validUntil == null ? '' : ' · valid until ${Fmt.date(s.validUntil!)}'}',
              pills: [statusPill(s.status == 'quotation' ? 'sent' : s.status)],
              onTap: () => showSubscriptionDetail(context, ref, s.id),
              trailing: Wrap(
                spacing: 4,
                children: [
                  if (s.status == 'draft')
                    TextButton(
                      onPressed: () =>
                          _act(s.id, 'send', 'Quotation sent to the customer'),
                      child: const Text('Send'),
                    ),
                  FilledButton(
                    onPressed: () => _act(
                      s.id,
                      'confirm',
                      'Confirmed: subscription created',
                    ),
                    child: const Text('Confirm'),
                  ),
                  IconButton(
                    tooltip: 'Cancel ${s.quotationNumber}',
                    icon: const Icon(Icons.close),
                    onPressed: () =>
                        _act(s.id, 'cancel', 'Quotation cancelled'),
                  ),
                ],
              ),
            ),
          Pager(
            page: page.page,
            pageSize: page.pageSize,
            total: page.total,
            onPage: (n) =>
                setState(() => _q = (q: _q.q, status: _q.status, page: n)),
          ),
        ],
      ),
    ),
  );

  Future<void> _act(int id, String action, String ok) async {
    if (await runAction(
      context,
      () => ref.read(salesRepositoryProvider).quotationAction(id, action),
      success: ok,
    )) {
      invalidateSales(ref);
    }
  }

  Future<void> _create() async {
    final api = ref.read(apiClientProvider);
    final customers =
        (await api
                .getPaged(
                  '/customers',
                  Customer.fromJson,
                  query: {'page_size': 100, 'sort': 'name'},
                )
                .catchError(
                  (_) => Paged<Customer>(
                    items: [],
                    page: 1,
                    pageSize: 1,
                    total: 0,
                  ),
                ))
            .items;
    final plans = await ref
        .read(activePlansProvider.future)
        .catchError((_) => <Plan>[]);
    final products = await ref
        .read(activeProductsProvider.future)
        .catchError((_) => <Product>[]);
    final templates =
        (await ref
                .read(templatesRepositoryProvider)
                .list((q: '', active: true, page: 1))
                .catchError(
                  (_) => Paged<QuotationTemplate>(
                    items: [],
                    page: 1,
                    pageSize: 1,
                    total: 0,
                  ),
                ))
            .items;
    if (!mounted) return;
    int? customerId, planId, templateId;
    final start = TextEditingController(text: Fmt.apiDate(DateTime.now()));
    final expiry = TextEditingController();
    final terms = TextEditingController();
    var custom = false;
    final lines = [LineDraft()];
    final saved = await showApiForm<bool>(
      context,
      title: 'New quotation',
      width: 680,
      fields: (_, setState, e) => [
        if (customers.isEmpty)
          const Text(
            'Add a customer first (Customers page).',
            style: TextStyle(color: Color(0xFFB3261E)),
          ),
        DropdownButtonFormField<int>(
          initialValue: customerId,
          isExpanded: true,
          decoration: InputDecoration(
            labelText: 'Customer',
            errorText: fieldError(e, 'customer_id'),
          ),
          items: [
            for (final c in customers)
              DropdownMenuItem(
                value: c.id,
                child: Text(
                  '${c.name} · ${c.email}',
                  overflow: TextOverflow.ellipsis,
                ),
              ),
          ],
          validator: (v) => v == null ? 'Choose a customer' : null,
          onChanged: (v) => customerId = v,
        ),
        DropdownButtonFormField<int>(
          initialValue: planId,
          isExpanded: true,
          decoration: InputDecoration(
            labelText: 'Plan',
            errorText:
                fieldError(e, 'plan_id') ?? (e['rule'] != null ? null : null),
          ),
          items: [
            for (final p in plans)
              DropdownMenuItem(
                value: p.id,
                child: Text(
                  '${p.name} · ${Fmt.inr(p.price)} ${p.billingLabel} · min ${p.minQuantity}',
                ),
              ),
          ],
          validator: (v) => v == null ? 'Choose a plan' : null,
          onChanged: (v) => planId = v,
        ),
        DropdownButtonFormField<int?>(
          initialValue: templateId,
          isExpanded: true,
          decoration: InputDecoration(
            labelText: 'Template (optional)',
            errorText: fieldError(e, 'template_id'),
          ),
          items: [
            const DropdownMenuItem<int?>(
              value: null,
              child: Text('No template'),
            ),
            for (final t in templates)
              DropdownMenuItem<int?>(
                value: t.id,
                child: Text('${t.name} · valid ${t.validityDays} days'),
              ),
          ],
          onChanged: (v) => templateId = v,
        ),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            SizedBox(
              width: 200,
              child: DateField(
                controller: start,
                label: 'Start date',
                required: true,
                errorText: fieldError(e, 'start_date'),
              ),
            ),
            SizedBox(
              width: 200,
              child: DateField(
                controller: expiry,
                label: 'Expiry date (optional)',
                errorText: fieldError(e, 'expiry_date'),
              ),
            ),
            SizedBox(
              width: 170,
              child: formField(
                terms,
                'Payment terms (days)',
                hint: '15',
                errors: e,
                errorKey: 'payment_terms_days',
                keyboard: TextInputType.number,
                validator: (v) => intValidator(v, min: 0, required: false),
              ),
            ),
          ],
        ),
        SwitchListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('Choose product lines myself'),
          subtitle: const Text(
            'Off: lines come from the template, or from the plan.',
          ),
          value: custom,
          onChanged: (v) => setState(() => custom = v),
        ),
        if (custom)
          LineEditor(
            products: products,
            lines: lines,
            onChanged: () => setState(() {}),
            allowOverride: true,
            overrideLabel: 'Unit price',
            errors: e,
            errorKey: 'items',
          ),
      ],
      onSave: () async {
        await ref.read(salesRepositoryProvider).createQuotation({
          'customer_id': customerId,
          'plan_id': planId,
          'template_id': ?templateId,
          'start_date': start.text.trim(),
          if (expiry.text.trim().isNotEmpty) 'expiry_date': expiry.text.trim(),
          if (terms.text.trim().isNotEmpty)
            'payment_terms_days': int.parse(terms.text.trim()),
          if (custom)
            'items': [
              for (final l in lines)
                ?l.toJson(withOverride: true, overrideKey: 'unit_price'),
            ],
        });
        return true;
      },
    );
    for (final c in [start, expiry, terms]) {
      c.dispose();
    }
    for (final l in lines) {
      l.dispose();
    }
    if (saved == true && mounted) {
      invalidateSales(ref);
      showMessage(context, 'Quotation created as a draft');
    }
  }
}
