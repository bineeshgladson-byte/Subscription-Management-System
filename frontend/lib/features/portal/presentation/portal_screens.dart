import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../sales/data/sales_api.dart';
import '../../sales/presentation/sales_widgets.dart';

/// Customer portal: "see only their own subscriptions and invoices" and
/// "manage their own subscriptions without having to email anyone".

class MySubscriptionsScreen extends ConsumerStatefulWidget {
  const MySubscriptionsScreen({super.key});
  @override
  ConsumerState<MySubscriptionsScreen> createState() => _MySubscriptionsState();
}

class _MySubscriptionsState extends ConsumerState<MySubscriptionsScreen> {
  int _page = 1;
  @override
  Widget build(BuildContext context) => PageFrame(
    title: 'My subscriptions',
    subtitle:
        'Pause, resume, renew or close a subscription yourself, where your plan allows it.',
    child: AsyncView(
      value: ref.watch(mySubscriptionsProvider(_page)),
      onRetry: () => ref.invalidate(mySubscriptionsProvider),
      builder: (page) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (page.items.isEmpty)
            const EmptyState(
              icon: Icons.autorenew,
              title: 'No subscriptions yet',
              message: 'Your subscriptions appear here once confirmed.',
            ),
          for (final s in page.items)
            ResourceTile(
              title: '${s.planName} · ${s.number}',
              subtitle:
                  '${Fmt.inr(s.total)} per period (${s.billingLabel})'
                  '${s.nextBillingDate != null && s.status == 'active' ? ' · next invoice ${Fmt.date(s.nextBillingDate!)}' : ''}',
              pills: [statusPill(s.status)],
              onTap: () => _detail(context, ref, s.id),
            ),
          Pager(
            page: page.page,
            pageSize: page.pageSize,
            total: page.total,
            onPage: (n) => setState(() => _page = n),
          ),
        ],
      ),
    ),
  );
}

Future<void> _detail(
  BuildContext context,
  WidgetRef ref,
  int id,
) => showDialog<void>(
  context: context,
  builder: (dialogContext) => Consumer(
    builder: (context, ref, _) {
      final value = ref.watch(mySubscriptionProvider(id));
      final s = value.value;
      Future<void> act(String action, String ok) async {
        if (await runAction(
          dialogContext,
          () => ref.read(salesRepositoryProvider).myAction(id, action),
          success: ok,
        )) {
          ref.invalidate(mySubscriptionsProvider);
          ref.invalidate(mySubscriptionProvider);
          ref.invalidate(myInvoicesProvider);
        }
      }

      return AlertDialog(
        title: Text(s == null ? 'Subscription' : s.planName),
        content: SizedBox(
          width: 560,
          child: AsyncView(
            value: value,
            builder: (s) => SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Wrap(
                    spacing: 6,
                    children: [statusPill(s.status), Pill(s.number)],
                  ),
                  const SizedBox(height: 8),
                  InfoRow('Billing', s.billingLabel),
                  InfoRow('Started', Fmt.date(s.startDate)),
                  if (s.expiryDate != null)
                    InfoRow('Ends', Fmt.date(s.expiryDate!)),
                  if (s.nextBillingDate != null && s.status == 'active')
                    InfoRow('Next invoice', Fmt.date(s.nextBillingDate!)),
                  const Divider(),
                  for (final it in s.items)
                    moneyRow(
                      context,
                      '${it.quantity} × ${it.description}',
                      Fmt.inr(it.lineSubtotal),
                    ),
                  moneyRow(
                    context,
                    'Per period incl. tax',
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
                      ),
                  ],
                ],
              ),
            ),
          ),
        ),
        actions: [
          if (s != null && s.status == 'active' && s.isPausable)
            TextButton(
              onPressed: () => act('pause', 'Subscription paused'),
              child: const Text('Pause'),
            ),
          if (s != null && s.status == 'paused')
            TextButton(
              onPressed: () => act('resume', 'Subscription resumed'),
              child: const Text('Resume'),
            ),
          if (s != null &&
              s.isRenewable &&
              {'active', 'paused', 'closed'}.contains(s.status))
            TextButton(
              onPressed: () =>
                  act('renew', 'Renewal requested · our team will confirm it'),
              child: const Text('Renew'),
            ),
          if (s != null &&
              s.isClosable &&
              {'confirmed', 'active', 'paused'}.contains(s.status))
            TextButton(
              onPressed: () async {
                if (await confirmAction(
                  dialogContext,
                  title: 'Close this subscription?',
                  message:
                      'No further invoices will be created. Unpaid invoices stay due.',
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

class MyInvoicesScreen extends ConsumerStatefulWidget {
  const MyInvoicesScreen({super.key});
  @override
  ConsumerState<MyInvoicesScreen> createState() => _MyInvoicesState();
}

class _MyInvoicesState extends ConsumerState<MyInvoicesScreen> {
  int _page = 1;
  @override
  Widget build(BuildContext context) => PageFrame(
    title: 'My invoices',
    subtitle:
        'Every invoice issued to you, with what is paid and what is still due.',
    child: AsyncView(
      value: ref.watch(myInvoicesProvider(_page)),
      onRetry: () => ref.invalidate(myInvoicesProvider),
      builder: (page) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (page.items.isEmpty)
            const EmptyState(
              icon: Icons.receipt_long_outlined,
              title: 'No invoices yet',
              message: 'Invoices appear here when they are issued.',
            ),
          for (final i in page.items)
            ResourceTile(
              title: '${i.number} · ${Fmt.inr(i.total)}',
              subtitle:
                  '${period(i)} · due ${Fmt.date(i.dueDate)} · still due ${Fmt.inr(i.amountDue)}',
              pills: [invoicePills(i)],
              onTap: () => _invoice(context, ref, i.id),
            ),
          Pager(
            page: page.page,
            pageSize: page.pageSize,
            total: page.total,
            onPage: (n) => setState(() => _page = n),
          ),
        ],
      ),
    ),
  );
}

Future<void> _invoice(BuildContext context, WidgetRef ref, int id) =>
    showDialog<void>(
      context: context,
      builder: (c) => Consumer(
        builder: (context, ref, _) => AlertDialog(
          title: const Text('Invoice'),
          content: SizedBox(
            width: 560,
            child: AsyncView(
              value: ref.watch(myInvoiceProvider(id)),
              builder: (inv) => SingleChildScrollView(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      inv.number,
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    invoicePills(inv),
                    const SizedBox(height: 8),
                    InfoRow('Period', period(inv)),
                    InfoRow(
                      'Invoice / due date',
                      '${Fmt.date(inv.invoiceDate)} / ${Fmt.date(inv.dueDate)}',
                    ),
                    const Divider(),
                    for (final l in inv.lines)
                      moneyRow(
                        context,
                        '${l.quantity} × ${l.description} (GST ${l.taxRate}%)',
                        Fmt.inr(l.lineTotal),
                      ),
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
                  ],
                ),
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(c),
              child: const Text('Close'),
            ),
          ],
        ),
      ),
    );

class MyPaymentsScreen extends ConsumerStatefulWidget {
  const MyPaymentsScreen({super.key});
  @override
  ConsumerState<MyPaymentsScreen> createState() => _MyPaymentsState();
}

class _MyPaymentsState extends ConsumerState<MyPaymentsScreen> {
  int _page = 1;
  @override
  Widget build(BuildContext context) => PageFrame(
    title: 'Payment history',
    subtitle: 'Payments we have received from you.',
    child: AsyncView(
      value: ref.watch(myPaymentsProvider(_page)),
      onRetry: () => ref.invalidate(myPaymentsProvider),
      builder: (page) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (page.items.isEmpty)
            const EmptyState(
              icon: Icons.payments_outlined,
              title: 'No payments yet',
              message: 'Your payments appear here once recorded.',
            ),
          for (final p in page.items)
            ResourceTile(
              title: Fmt.inr(p.amount),
              subtitle:
                  '${Fmt.date(p.paymentDate)} · ${paymentMethods[p.method] ?? p.method} · invoice ${p.invoiceNumber ?? p.invoiceId}${p.reference == null ? '' : ' · ${p.reference}'}',
              pills: [statusPill(p.status)],
            ),
          Pager(
            page: page.page,
            pageSize: page.pageSize,
            total: page.total,
            onPage: (n) => setState(() => _page = n),
          ),
        ],
      ),
    ),
  );
}
