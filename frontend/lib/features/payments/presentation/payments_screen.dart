import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../invoices/presentation/invoices_screen.dart';
import '../../sales/data/sales_api.dart';
import '../../sales/presentation/sales_widgets.dart';

class PaymentsScreen extends ConsumerStatefulWidget {
  const PaymentsScreen({super.key});
  @override
  ConsumerState<PaymentsScreen> createState() => _PaymentsScreenState();
}

class _PaymentsScreenState extends ConsumerState<PaymentsScreen> {
  PaymentQuery _q = (q: '', method: null, page: 1);

  @override
  Widget build(BuildContext context) => PageFrame(
    title: 'Payments',
    subtitle:
        'Money received against invoices. Record a payment from the invoice.',
    toolbar: Toolbar(
      children: [
        SearchBox(
          hint: 'Search payment, invoice or customer',
          onChanged: (v) =>
              setState(() => _q = (q: v, method: _q.method, page: 1)),
        ),
        FilterDropdown<String>(
          label: 'Method',
          value: _q.method,
          options: paymentMethods,
          onChanged: (v) => setState(() => _q = (q: _q.q, method: v, page: 1)),
        ),
      ],
    ),
    child: AsyncView(
      value: ref.watch(paymentsProvider(_q)),
      onRetry: () => ref.invalidate(paymentsProvider),
      builder: (page) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (page.items.isEmpty)
            const EmptyState(
              icon: Icons.payments_outlined,
              title: 'No payments yet',
              message: 'Open an invoice and choose “Record payment”.',
            ),
          for (final p in page.items)
            ResourceTile(
              title: '${Fmt.inr(p.amount)} · ${p.customerName ?? ''}',
              subtitle:
                  '${p.number} · ${paymentMethods[p.method] ?? p.method} · ${Fmt.date(p.paymentDate)} · invoice ${p.invoiceNumber ?? p.invoiceId}'
                  '${p.reference == null ? '' : ' · ${p.reference}'}${p.notes == null || p.notes!.isEmpty ? '' : ' · ${p.notes}'}',
              pills: [statusPill(p.status)],
              onTap: () => showInvoiceDetail(context, ref, p.invoiceId),
            ),
          Pager(
            page: page.page,
            pageSize: page.pageSize,
            total: page.total,
            onPage: (n) =>
                setState(() => _q = (q: _q.q, method: _q.method, page: n)),
          ),
        ],
      ),
    ),
  );
}
