import 'dart:math';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../sales/data/sales_api.dart';
import '../../sales/presentation/sales_widgets.dart';

class InvoicesScreen extends ConsumerStatefulWidget {
  const InvoicesScreen({super.key});
  @override
  ConsumerState<InvoicesScreen> createState() => _InvoicesScreenState();
}

class _InvoicesScreenState extends ConsumerState<InvoicesScreen> {
  InvoiceQuery _q = (q: '', status: null, overdue: false, page: 1);

  @override
  Widget build(BuildContext context) => PageFrame(
    title: 'Invoices',
    subtitle:
        'Created automatically from subscriptions, with taxes and discounts already applied.',
    toolbar: Toolbar(
      children: [
        SearchBox(
          hint: 'Search invoice number or customer',
          onChanged: (v) => setState(
            () => _q = (q: v, status: _q.status, overdue: _q.overdue, page: 1),
          ),
        ),
        FilterDropdown<String>(
          label: 'Status',
          value: _q.status,
          options: invoiceStatuses,
          onChanged: (v) => setState(
            () => _q = (q: _q.q, status: v, overdue: _q.overdue, page: 1),
          ),
        ),
        FilterChip(
          label: const Text('Overdue only'),
          selected: _q.overdue,
          onSelected: (v) => setState(
            () => _q = (q: _q.q, status: _q.status, overdue: v, page: 1),
          ),
        ),
      ],
    ),
    child: AsyncView(
      value: ref.watch(invoicesProvider(_q)),
      onRetry: () => ref.invalidate(invoicesProvider),
      builder: (page) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (page.items.isEmpty)
            const EmptyState(
              icon: Icons.receipt_long_outlined,
              title: 'No invoices',
              message: 'Invoices appear when subscriptions are billed.',
            ),
          for (final i in page.items)
            ResourceTile(
              title: '${i.number} · ${i.customerName}',
              subtitle:
                  '${period(i)} · due ${Fmt.date(i.dueDate)} · total ${Fmt.inr(i.total)} · due ${Fmt.inr(i.amountDue)}'
                  '${i.isOverdue ? ' · ${i.daysOverdue} days late' : ''}',
              pills: [invoicePills(i)],
              onTap: () => showInvoiceDetail(context, ref, i.id),
            ),
          Pager(
            page: page.page,
            pageSize: page.pageSize,
            total: page.total,
            onPage: (n) => setState(
              () => _q = (
                q: _q.q,
                status: _q.status,
                overdue: _q.overdue,
                page: n,
              ),
            ),
          ),
        ],
      ),
    ),
  );
}

Future<void> showInvoiceDetail(
  BuildContext context,
  WidgetRef ref,
  int id,
) => showDialog<void>(
  context: context,
  builder: (dialogContext) => Consumer(
    builder: (context, ref, _) {
      final value = ref.watch(invoiceDetailProvider(id));
      final inv = value.value;
      Future<void> act(String action, String ok) async {
        if (await runAction(
          dialogContext,
          () => ref.read(salesRepositoryProvider).invoiceAction(id, action),
          success: ok,
        )) {
          invalidateSales(ref);
        }
      }

      return AlertDialog(
        title: Text(inv == null ? 'Invoice' : 'Invoice ${inv.number}'),
        content: SizedBox(
          width: 640,
          child: AsyncView(
            value: value,
            builder: (inv) => SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  invoicePills(inv),
                  const SizedBox(height: 8),
                  InfoRow(
                    'Bill to',
                    [
                      inv.customerName,
                      inv.billingAddress,
                      if (inv.gstin != null) 'GSTIN ${inv.gstin}',
                    ].whereType<String>().join('\n'),
                  ),
                  if (inv.subscriptionNumber != null)
                    InfoRow('Subscription', inv.subscriptionNumber!),
                  InfoRow('Period', period(inv)),
                  InfoRow(
                    'Invoice / due date',
                    '${Fmt.date(inv.invoiceDate)} / ${Fmt.date(inv.dueDate)}',
                  ),
                  if (inv.sentAt != null)
                    InfoRow(
                      'Sent',
                      inv.sentAt!.toLocal().toString().substring(0, 16),
                    ),
                  const Divider(),
                  for (final l in inv.lines) ...[
                    Text(
                      '${l.quantity} × ${l.description} @ ${Fmt.inr(l.unitPrice)}',
                      style: const TextStyle(fontWeight: FontWeight.w500),
                    ),
                    moneyRow(context, '  Subtotal', Fmt.inr(l.lineSubtotal)),
                    if (Fmt.money(l.discountAmount) > Fmt.money('0'))
                      moneyRow(
                        context,
                        '  Discount',
                        '−${Fmt.inr(l.discountAmount)}',
                      ),
                    moneyRow(
                      context,
                      '  GST ${l.taxRate}%',
                      Fmt.inr(l.taxAmount),
                    ),
                  ],
                  const Divider(),
                  moneyRow(context, 'Subtotal', Fmt.inr(inv.subtotal)),
                  moneyRow(
                    context,
                    'Discounts',
                    '−${Fmt.inr(inv.discountTotal)}',
                  ),
                  moneyRow(context, 'Tax', Fmt.inr(inv.taxTotal)),
                  moneyRow(context, 'Total', Fmt.inr(inv.total), bold: true),
                  moneyRow(context, 'Paid', Fmt.inr(inv.amountPaid)),
                  moneyRow(
                    context,
                    'Amount due',
                    Fmt.inr(inv.amountDue),
                    bold: true,
                  ),
                  if (inv.payments.isNotEmpty) ...[
                    const Divider(),
                    Text(
                      'Payments',
                      style: Theme.of(context).textTheme.titleSmall,
                    ),
                    for (final p in inv.payments)
                      moneyRow(
                        context,
                        '${Fmt.date(p.paymentDate)} · ${paymentMethods[p.method] ?? p.method}${p.reference == null ? '' : ' · ${p.reference}'}',
                        Fmt.inr(p.amount),
                      ),
                  ],
                ],
              ),
            ),
          ),
        ),
        actions: [
          if (inv?.status == 'draft')
            FilledButton(
              onPressed: () => act('confirm', 'Invoice confirmed'),
              child: const Text('Confirm'),
            ),
          if (inv != null &&
              inv.status == 'confirmed' &&
              Fmt.money(inv.amountDue) > Fmt.money('0'))
            FilledButton(
              onPressed: () => recordPaymentDialog(dialogContext, ref, inv),
              child: const Text('Record payment'),
            ),
          if (inv != null &&
              (inv.status == 'confirmed' || inv.status == 'paid'))
            TextButton(
              onPressed: () => act(
                'send',
                'Invoice emailed to ${inv.customerEmail ?? 'the customer'}',
              ),
              child: const Text('Send'),
            ),
          if (inv != null && inv.status != 'cancelled')
            TextButton(
              onPressed: () => _print(dialogContext, inv),
              child: const Text('Print'),
            ),
          if (inv != null &&
              (inv.status == 'draft' || inv.status == 'confirmed') &&
              Fmt.money(inv.amountPaid) == Fmt.money('0'))
            TextButton(
              onPressed: () async {
                if (await confirmAction(
                  dialogContext,
                  title: 'Cancel ${inv.number}?',
                  message:
                      'The invoice is voided and any discount use is given back.',
                  confirmLabel: 'Cancel invoice',
                  danger: true,
                )) {
                  await act('cancel', 'Invoice cancelled');
                }
              },
              child: const Text(
                'Cancel',
                style: TextStyle(color: Color(0xFFB3261E)),
              ),
            ),
          TextButton(
            onPressed: () => Navigator.pop(dialogContext),
            child: const Text('Done'),
          ),
        ],
      );
    },
  ),
);

/// A clean printable view; use the browser's print (Ctrl+P) on it.
void _print(BuildContext context, Invoice inv) => showDialog<void>(
  context: context,
  builder: (c) => Dialog.fullscreen(
    child: Scaffold(
      appBar: AppBar(
        title: Text('${inv.number} · press Ctrl+P to print'),
        actions: [
          IconButton(
            onPressed: () => Navigator.pop(c),
            icon: const Icon(Icons.close),
          ),
        ],
      ),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 760),
          child: ListView(
            padding: const EdgeInsets.all(32),
            children: [
              const Text(
                'Brightloop Software · Subly',
                style: TextStyle(fontSize: 22, fontWeight: FontWeight.w600),
              ),
              Text(
                'TAX INVOICE ${inv.number}',
                style: const TextStyle(fontSize: 16),
              ),
              const SizedBox(height: 16),
              Text(
                'Bill to: ${inv.customerName}${inv.billingAddress == null ? '' : '\n${inv.billingAddress}'}${inv.gstin == null ? '' : '\nGSTIN ${inv.gstin}'}',
              ),
              Text(
                'Invoice date ${Fmt.date(inv.invoiceDate)} · Due ${Fmt.date(inv.dueDate)} · Period ${period(inv)}',
              ),
              const Divider(height: 32),
              for (final l in inv.lines)
                moneyRow(
                  c,
                  '${l.quantity} × ${l.description} @ ${Fmt.inr(l.unitPrice)} · disc ${Fmt.inr(l.discountAmount)} · GST ${l.taxRate}% ${Fmt.inr(l.taxAmount)}',
                  Fmt.inr(l.lineTotal),
                ),
              const Divider(height: 32),
              moneyRow(c, 'Subtotal', Fmt.inr(inv.subtotal)),
              moneyRow(c, 'Discounts', '−${Fmt.inr(inv.discountTotal)}'),
              moneyRow(c, 'GST', Fmt.inr(inv.taxTotal)),
              moneyRow(c, 'Total', Fmt.inr(inv.total), bold: true),
              moneyRow(c, 'Paid', Fmt.inr(inv.amountPaid)),
              moneyRow(c, 'Balance due', Fmt.inr(inv.amountDue), bold: true),
            ],
          ),
        ),
      ),
    ),
  ),
);

Future<void> recordPaymentDialog(
  BuildContext context,
  WidgetRef ref,
  Invoice inv,
) async {
  final amount = TextEditingController(text: inv.amountDue);
  final date = TextEditingController(text: Fmt.apiDate(DateTime.now()));
  final reference = TextEditingController();
  final notes = TextEditingController();
  var method = 'upi';
  // One key per dialog: a double-click can't record the payment twice.
  final key =
      '${DateTime.now().microsecondsSinceEpoch}-${Random().nextInt(1 << 30)}';
  final saved = await showApiForm<bool>(
    context,
    title: 'Record payment for ${inv.number}',
    saveLabel: 'Record payment',
    fields: (_, setState, e) => [
      Text(
        'Amount due ${Fmt.inr(inv.amountDue)}',
        style: Theme.of(context).textTheme.bodySmall,
      ),
      DropdownButtonFormField<String>(
        initialValue: method,
        decoration: const InputDecoration(labelText: 'Method'),
        items: [
          for (final m in paymentMethods.entries)
            DropdownMenuItem(value: m.key, child: Text(m.value)),
        ],
        onChanged: (v) => method = v ?? method,
      ),
      formField(
        amount,
        'Amount (₹)',
        errors: e,
        errorKey: 'amount',
        validator: moneyValidator(),
      ),
      DateField(
        controller: date,
        label: 'Payment date',
        required: true,
        errorText: fieldError(e, 'payment_date'),
      ),
      formField(
        reference,
        'Reference (UTR / transaction id)',
        errors: e,
        errorKey: 'reference',
      ),
      formField(notes, 'Note', errors: e, errorKey: 'notes', maxLines: 2),
    ],
    onSave: () async {
      await ref.read(salesRepositoryProvider).recordPayment(inv.id, {
        'method': method,
        'amount': amount.text.trim(),
        'payment_date': date.text.trim(),
        'reference': reference.text.trim(),
        'notes': notes.text.trim(),
      }, key);
      return true;
    },
  );
  for (final c in [amount, date, reference, notes]) {
    c.dispose();
  }
  if (saved == true && context.mounted) {
    invalidateSales(ref);
    showMessage(context, 'Payment recorded');
  }
}
