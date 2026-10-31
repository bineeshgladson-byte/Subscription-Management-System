import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../auth/application/auth_controller.dart';
import '../data/subscriptions_api.dart';

class SubscriptionsScreen extends ConsumerStatefulWidget {
  const SubscriptionsScreen({super.key});

  @override
  ConsumerState<SubscriptionsScreen> createState() => _SubscriptionsScreenState();
}

class _SubscriptionsScreenState extends ConsumerState<SubscriptionsScreen> {
  SubscriptionQuery _q = (q: '', status: null, customerId: null, page: 1);

  @override
  Widget build(BuildContext context) {
    final role = ref.watch(authControllerProvider).value?.role ?? Role.customer;
    final isCustomer = role == Role.customer;
    final data = ref.watch(subscriptionsProvider(_q));

    return PageFrame(
      title: 'Subscriptions',
      subtitle: isCustomer
          ? 'Your active and past subscriptions.'
          : 'Track subscription lifecycles, manage renewals, pauses and trigger manual billing.',
      toolbar: Toolbar(
        children: [
          SearchBox(
            hint: 'Search subscription number or customer',
            onChanged: (v) => setState(
              () => _q = (q: v, status: _q.status, customerId: _q.customerId, page: 1),
            ),
          ),
          FilterDropdown<String>(
            label: 'Status',
            value: _q.status,
            options: const {
              'active': 'Active',
              'confirmed': 'Confirmed',
              'paused': 'Paused',
              'closed': 'Closed',
              'renewed': 'Renewed',
            },
            onChanged: (v) => setState(
              () => _q = (q: _q.q, status: v, customerId: _q.customerId, page: 1),
            ),
          ),
        ],
      ),
      child: AsyncView(
        value: data,
        onRetry: () => ref.invalidate(subscriptionsProvider),
        builder: (page) => Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (page.items.isEmpty)
              const EmptyState(
                icon: Icons.sync_alt_outlined,
                title: 'No subscriptions found',
                message: 'No subscriptions match your query. Confirm a quotation to create one.',
              ),
            for (final sub in page.items)
              ResourceTile(
                leading: CircleAvatar(
                  backgroundColor: _statusTone(sub.status) == 'good'
                      ? const Color(0xFFE3F4EA)
                      : const Color(0xFFF1F5F9),
                  child: Icon(
                    Icons.autorenew_outlined,
                    color: _statusTone(sub.status) == 'good'
                        ? const Color(0xFF0F8A52)
                        : const Color(0xFF64748B),
                  ),
                ),
                title: '${sub.subscriptionNumber} · ${sub.customerName}',
                subtitle: '${sub.planName} (${sub.billingPeriod}) · Started: ${sub.startDate} · Next Billing: ${sub.nextBillingDate ?? "N/A"} · Cycles Billed: ${sub.cyclesBilled}',
                pills: [
                  Pill(sub.status.toUpperCase(), tone: _statusTone(sub.status)),
                  Pill(Fmt.inr(sub.total), tone: 'brand'),
                ],
                onTap: () => _viewDetails(sub),
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (!isCustomer) ...[
                      if (sub.isConfirmed)
                        FilledButton.tonal(
                          onPressed: () => _activate(sub.id),
                          child: const Text('Activate'),
                        ),
                      if (sub.isActive) ...[
                        IconButton(
                          tooltip: 'Pause Subscription',
                          icon: const Icon(Icons.pause_circle_outline, color: Colors.orange),
                          onPressed: () => _pause(sub.id),
                        ),
                        IconButton(
                          tooltip: 'Bill Now (Generate Invoice)',
                          icon: const Icon(Icons.receipt_long_outlined, color: Colors.indigo),
                          onPressed: () => _billNow(sub.id),
                        ),
                        IconButton(
                          tooltip: 'Close Early',
                          icon: const Icon(Icons.cancel_outlined, color: Colors.red),
                          onPressed: () => _close(sub.id),
                        ),
                      ],
                      if (sub.isPaused)
                        IconButton(
                          tooltip: 'Resume Subscription',
                          icon: const Icon(Icons.play_circle_outline, color: Colors.green),
                          onPressed: () => _resume(sub.id),
                        ),
                      if (sub.isActive || sub.isClosed)
                        IconButton(
                          tooltip: 'Renew Subscription',
                          icon: const Icon(Icons.refresh_outlined, color: Colors.blue),
                          onPressed: () => _renew(sub.id),
                        ),
                    ],
                    IconButton(
                      tooltip: 'View Details',
                      icon: const Icon(Icons.visibility_outlined),
                      onPressed: () => _viewDetails(sub),
                    ),
                  ],
                ),
              ),
            Pager(
              page: page.page,
              pageSize: page.pageSize,
              total: page.total,
              onPage: (p) => setState(
                () => _q = (q: _q.q, status: _q.status, customerId: _q.customerId, page: p),
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _statusTone(String status) => switch (status) {
        'active' => 'good',
        'confirmed' => 'brand',
        'paused' => 'warn',
        'closed' => 'bad',
        'renewed' => 'good',
        _ => 'neutral',
      };

  Future<void> _activate(int id) async {
    try {
      await ref.read(subscriptionsRepositoryProvider).activate(id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Subscription activated!')),
        );
        ref.invalidate(subscriptionsProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Failed: $e')));
      }
    }
  }

  Future<void> _pause(int id) async {
    try {
      await ref.read(subscriptionsRepositoryProvider).pause(id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Subscription paused')),
        );
        ref.invalidate(subscriptionsProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Failed: $e')));
      }
    }
  }

  Future<void> _resume(int id) async {
    try {
      await ref.read(subscriptionsRepositoryProvider).resume(id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Subscription resumed to active!')),
        );
        ref.invalidate(subscriptionsProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Failed: $e')));
      }
    }
  }

  Future<void> _close(int id) async {
    final reasonCtrl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Close Subscription'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text('Are you sure you want to close this subscription early?'),
            const SizedBox(height: 12),
            TextField(
              controller: reasonCtrl,
              decoration: const InputDecoration(labelText: 'Close Reason (Optional)'),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Back')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Close Subscription'),
          ),
        ],
      ),
    );
    if (ok != true) return;

    try {
      await ref.read(subscriptionsRepositoryProvider).close(id, reason: reasonCtrl.text.trim());
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Subscription closed')),
        );
        ref.invalidate(subscriptionsProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Failed: $e')));
      }
    }
  }

  Future<void> _renew(int id) async {
    try {
      await ref.read(subscriptionsRepositoryProvider).renew(id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Subscription renewed successfully!')),
        );
        ref.invalidate(subscriptionsProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Failed: $e')));
      }
    }
  }

  Future<void> _billNow(int id) async {
    try {
      final res = await ref.read(subscriptionsRepositoryProvider).billNow(id);
      if (mounted) {
        final invNum = res is Map ? res['invoice_number'] ?? '' : '';
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Invoice $invNum generated successfully for subscription!')),
        );
        ref.invalidate(subscriptionsProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Failed to bill: $e')));
      }
    }
  }

  Future<void> _viewDetails(Subscription sub) async {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('${sub.subscriptionNumber} Details'),
        content: SizedBox(
          width: 550,
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                _row('Quotation Ref', sub.quotationNumber),
                _row('Customer', sub.customerName),
                _row('Email', sub.customerEmail),
                _row('Plan', '${sub.planName} (${sub.billingPeriod})'),
                _row('Status', sub.status.toUpperCase()),
                _row('Start Date', sub.startDate),
                _row('Expiry Date', sub.expiryDate ?? 'Ongoing'),
                _row('Next Billing Date', sub.nextBillingDate ?? 'N/A'),
                _row('Billing Anchor Day', 'Day ${sub.billingAnchorDay}'),
                _row('Cycles Billed', '${sub.cyclesBilled}'),
                if (sub.closeReason != null && sub.closeReason!.isNotEmpty)
                  _row('Close Reason', sub.closeReason!),
                const Divider(height: 24),
                const Text('Line Items', style: TextStyle(fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                for (final line in sub.items)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 4),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text('${line.productName} × ${line.quantity}'),
                        Text(Fmt.inr(line.lineSubtotal)),
                      ],
                    ),
                  ),
                const Divider(height: 24),
                _row('Subtotal', Fmt.inr(sub.subtotal)),
                if (sub.discountTotal != '0.00' && sub.discountTotal != '0')
                  _row('Discount', '- ${Fmt.inr(sub.discountTotal)}'),
                _row('Tax', Fmt.inr(sub.taxTotal)),
                const SizedBox(height: 6),
                _row('Total per cycle', Fmt.inr(sub.total), isBold: true),
              ],
            ),
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Close')),
        ],
      ),
    );
  }

  Widget _row(String label, String value, {bool isBold = false}) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 3),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label, style: TextStyle(color: Colors.grey[700], fontWeight: isBold ? FontWeight.bold : FontWeight.normal)),
            Text(value, style: TextStyle(fontWeight: isBold ? FontWeight.bold : FontWeight.normal)),
          ],
        ),
      );
}
