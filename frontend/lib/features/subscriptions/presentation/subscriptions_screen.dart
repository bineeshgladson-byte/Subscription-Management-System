import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../invoices/presentation/invoices_screen.dart';
import '../../sales/data/sales_api.dart';
import '../../sales/presentation/sales_widgets.dart';

class SubscriptionsScreen extends ConsumerStatefulWidget {
  const SubscriptionsScreen({super.key});
  @override
  ConsumerState<SubscriptionsScreen> createState() =>
      _SubscriptionsScreenState();
}

class _SubscriptionsScreenState extends ConsumerState<SubscriptionsScreen> {
  SubQuery _q = (q: '', status: null, page: 1);

  @override
  Widget build(BuildContext context) => PageFrame(
    title: 'Subscriptions',
    subtitle:
        'Confirmed and running contracts. Invoices are created automatically on each billing date.',
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
          options: subStatuses,
          onChanged: (v) => setState(() => _q = (q: _q.q, status: v, page: 1)),
        ),
      ],
    ),
    child: AsyncView(
      value: ref.watch(subscriptionsProvider(_q)),
      onRetry: () => ref.invalidate(subscriptionsProvider),
      builder: (page) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (page.items.isEmpty)
            const EmptyState(
              icon: Icons.autorenew,
              title: 'No subscriptions yet',
              message: 'Confirm a quotation to start one.',
            ),
          for (final s in page.items)
            ResourceTile(
              title: '${s.number} · ${s.customerName}',
              subtitle:
                  '${s.planName} (${s.billingLabel}) · ${Fmt.inr(s.total)} per period · '
                  '${s.nextBillingDate == null ? 'no further billing' : 'next invoice ${Fmt.date(s.nextBillingDate!)}'} · ${s.cyclesBilled} billed',
              pills: [
                statusPill(s.status),
                if (s.closeReason != null)
                  Pill(s.closeReason!.replaceAll('_', ' ')),
              ],
              onTap: () => showSubscriptionDetail(context, ref, s.id),
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
}

/// Detail dialog with lifecycle actions (used by quotations and subscriptions).
Future<void> showSubscriptionDetail(
  BuildContext context,
  WidgetRef ref,
  int id,
) => showDialog<void>(
  context: context,
  builder: (dialogContext) => Consumer(
    builder: (context, ref, _) {
      final value = ref.watch(subscriptionDetailProvider(id));
      final s = value.value;
      Future<void> act(String action, String ok) async {
        if (await runAction(
          dialogContext,
          () =>
              ref.read(salesRepositoryProvider).subscriptionAction(id, action),
          success: ok,
        )) {
          invalidateSales(ref);
        }
      }

      return AlertDialog(
        title: Text(s == null ? 'Subscription' : s.number),
        content: SizedBox(
          width: 620,
          child: AsyncView(
            value: value,
            builder: (s) => SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Wrap(
                    spacing: 6,
                    children: [statusPill(s.status), Pill(s.quotationNumber)],
                  ),
                  const SizedBox(height: 8),
                  InfoRow('Customer', s.customerName),
                  InfoRow('Plan', '${s.planName} · ${s.billingLabel}'),
                  InfoRow(
                    'Start / expiry',
                    '${Fmt.date(s.startDate)} / ${s.expiryDate == null ? 'open-ended' : Fmt.date(s.expiryDate!)}',
                  ),
                  InfoRow('Payment terms', '${s.paymentTermsDays} days'),
                  if (s.nextBillingDate != null)
                    InfoRow('Next invoice', Fmt.date(s.nextBillingDate!)),
                  const Divider(),
                  for (final it in s.items)
                    moneyRow(
                      context,
                      '${it.quantity} × ${it.description} @ ${Fmt.inr(it.unitPrice)} (GST ${it.taxRate}%)',
                      Fmt.inr(it.lineSubtotal),
                    ),
                  moneyRow(
                    context,
                    'Discounts',
                    '−${Fmt.inr(s.discountTotal)}',
                  ),
                  moneyRow(context, 'Tax', Fmt.inr(s.taxTotal)),
                  moneyRow(
                    context,
                    'Per billing period',
                    Fmt.inr(s.total),
                    bold: true,
                  ),
                  if (s.invoices.isNotEmpty) ...[
                    const Divider(),
                    Text(
                      'Invoices',
                      style: Theme.of(context).textTheme.titleSmall,
                    ),
                    for (final i in s.invoices)
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        title: Text('${i.number} · ${Fmt.inr(i.total)}'),
                        subtitle: Text(period(i)),
                        trailing: invoicePills(i),
                        onTap: () =>
                            showInvoiceDetail(dialogContext, ref, i.id),
                      ),
                  ],
                ],
              ),
            ),
          ),
        ),
        actions: [
          if (s != null && s.status == 'active') ...[
            TextButton(
              onPressed: () => act('bill-now', 'Billed'),
              child: const Text('Bill now'),
            ),
            if (s.isPausable)
              TextButton(
                onPressed: () => act('pause', 'Paused'),
                child: const Text('Pause'),
              ),
          ],
          if (s != null && s.status == 'paused')
            TextButton(
              onPressed: () => act('resume', 'Resumed'),
              child: const Text('Resume'),
            ),
          if (s != null &&
              s.isRenewable &&
              {'active', 'paused', 'closed'}.contains(s.status))
            TextButton(
              onPressed: () => act('renew', 'Renewal quotation created'),
              child: const Text('Renew'),
            ),
          if (s != null &&
              s.isClosable &&
              {'confirmed', 'active', 'paused'}.contains(s.status))
            TextButton(
              onPressed: () async {
                if (await confirmAction(
                  dialogContext,
                  title: 'Close ${s.number}?',
                  message:
                      'No more invoices will be created. Unpaid invoices stay due.',
                  confirmLabel: 'Close',
                  danger: true,
                )) {
                  await act('close', 'Subscription closed');
                }
              },
              child: const Text(
                'Close',
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
