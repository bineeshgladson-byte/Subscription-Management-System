import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/customer_widgets.dart';
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
    subtitle: 'Your plans, renewals and quotations, neatly organised.',
    action: OutlinedButton.icon(
      onPressed: () => ref.invalidate(mySubscriptionsProvider),
      icon: const Icon(Icons.refresh_rounded, size: 18),
      label: const Text('Refresh'),
    ),
    toolbar: const CustomerIntro(
      icon: Icons.autorenew_rounded,
      eyebrow: 'Your subscriptions',
      title: 'Stay on top of every plan.',
      message:
          'Check your next billing date, review a quotation or manage an existing subscription.',
    ),
    child: AsyncView(
      value: ref.watch(mySubscriptionsProvider(_page)),
      onRetry: () => ref.invalidate(mySubscriptionsProvider),
      builder: (page) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          CustomerSection(
            title: 'Your plan collection',
            caption:
                '${page.total} record${page.total == 1 ? '' : 's'} · subscriptions and quotation requests',
            child: Column(
              children: [
                if (page.items.isEmpty)
                  const CustomerEmpty(
                    icon: Icons.layers_outlined,
                    title: 'Your next chapter starts with a plan',
                    message:
                        'Explore the catalogue and request a quotation. Your plans will appear here as they move forward.',
                    browse: true,
                  ),
                for (final s in page.items)
                  CustomerRecord(
                    icon: Icons.autorenew_rounded,
                    title: s.planName,
                    reference: s.number,
                    status: statusPill(s.status),
                    facts: [
                      ('Per billing period · incl. tax', Fmt.inr(s.total)),
                      ('Billing cycle', s.billingLabel),
                      (
                        'Next invoice',
                        s.nextBillingDate != null && s.status == 'active'
                            ? Fmt.date(s.nextBillingDate!)
                            : '—',
                      ),
                    ],
                    action: OutlinedButton.icon(
                      onPressed: () => _detail(context, ref, s.id),
                      icon: const Icon(Icons.arrow_forward_rounded, size: 18),
                      iconAlignment: IconAlignment.end,
                      label: const Text('View subscription'),
                    ),
                  ),
              ],
            ),
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
    subtitle: 'A clear view of your bills and what’s due.',
    action: OutlinedButton.icon(
      onPressed: () => ref.invalidate(myInvoicesProvider),
      icon: const Icon(Icons.refresh_rounded, size: 18),
      label: const Text('Refresh'),
    ),
    toolbar: const CustomerIntro(
      icon: Icons.receipt_long_outlined,
      eyebrow: 'Billing, made clear',
      title: 'Every invoice. Every detail.',
      message:
          'Review your billing periods, payment status and outstanding amounts in one place.',
    ),
    child: AsyncView(
      value: ref.watch(myInvoicesProvider(_page)),
      onRetry: () => ref.invalidate(myInvoicesProvider),
      builder: (page) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          CustomerSection(
            title: 'Your invoices',
            caption:
                '${page.total} invoice${page.total == 1 ? '' : 's'} in your account · open an invoice for the full breakdown',
            child: Column(
              children: [
                if (page.items.isEmpty)
                  const CustomerEmpty(
                    icon: Icons.receipt_long_outlined,
                    title: 'A fresh start for your billing',
                    message:
                        'No invoices have been issued yet. When your subscription is billed, you’ll find the full breakdown here.',
                  ),
                for (final i in page.items)
                  CustomerRecord(
                    icon: Icons.receipt_long_outlined,
                    title: i.number,
                    reference: period(i),
                    status: invoicePills(i),
                    facts: [
                      ('Invoice total', Fmt.inr(i.total)),
                      ('Amount due', Fmt.inr(i.amountDue)),
                      ('Due date', Fmt.date(i.dueDate)),
                    ],
                    action: OutlinedButton.icon(
                      onPressed: () => _invoice(context, ref, i.id),
                      icon: const Icon(Icons.arrow_forward_rounded, size: 18),
                      iconAlignment: IconAlignment.end,
                      label: const Text('View invoice'),
                    ),
                  ),
              ],
            ),
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
    subtitle: 'Your payment records, without the paperwork.',
    action: OutlinedButton.icon(
      onPressed: () => ref.invalidate(myPaymentsProvider),
      icon: const Icon(Icons.refresh_rounded, size: 18),
      label: const Text('Refresh'),
    ),
    toolbar: const CustomerIntro(
      icon: Icons.account_balance_wallet_outlined,
      eyebrow: 'Your payment record',
      title: 'Keep every payment in view.',
      message:
          'See when you paid, how you paid and which invoice each payment belongs to.',
    ),
    child: AsyncView(
      value: ref.watch(myPaymentsProvider(_page)),
      onRetry: () => ref.invalidate(myPaymentsProvider),
      builder: (page) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          CustomerSection(
            title: 'Your transactions',
            caption:
                '${page.total} payment${page.total == 1 ? '' : 's'} recorded',
            child: Column(
              children: [
                if (page.items.isEmpty)
                  const CustomerEmpty(
                    icon: Icons.account_balance_wallet_outlined,
                    title: 'Your payment story starts here',
                    message:
                        'Once a payment is recorded, you’ll see the amount, method and invoice reference here.',
                  ),
                for (final p in page.items)
                  CustomerRecord(
                    icon: Icons.account_balance_wallet_outlined,
                    title: Fmt.inr(p.amount),
                    reference: p.number,
                    status: statusPill(p.status),
                    facts: [
                      ('Payment date', Fmt.date(p.paymentDate)),
                      ('Payment method', paymentMethods[p.method] ?? p.method),
                      ('Invoice', p.invoiceNumber ?? '${p.invoiceId}'),
                      if (p.reference != null) ('Reference', p.reference!),
                    ],
                  ),
              ],
            ),
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
