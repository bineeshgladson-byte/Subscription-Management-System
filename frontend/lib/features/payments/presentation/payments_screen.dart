import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../auth/application/auth_controller.dart';
import '../data/payments_api.dart';

class PaymentsScreen extends ConsumerStatefulWidget {
  const PaymentsScreen({super.key});
  @override
  ConsumerState<PaymentsScreen> createState() => _PaymentsScreenState();
}

class _PaymentsScreenState extends ConsumerState<PaymentsScreen> {
  PaymentQuery _q = (q: '', method: null, page: 1);

  @override
  ConsumerState<PaymentsScreen> createState() => _PaymentsScreenState();
}

class _PaymentsScreenState extends ConsumerState<PaymentsScreen> {
  PaymentQuery _q = (method: null, status: null, invoiceId: null, customerId: null, page: 1);

  @override
  Widget build(BuildContext context) {
    final role = ref.watch(authControllerProvider).value?.role ?? Role.customer;
    final isCustomer = role == Role.customer;
    final data = ref.watch(paymentsProvider(_q));

    return PageFrame(
      title: 'Payments',
      subtitle: isCustomer
          ? 'History of payments received for your account.'
          : 'Record payments (UPI, bank transfer, card, cash), view transactions, track status and process refunds.',
      toolbar: Toolbar(
        children: [
          FilterDropdown<String>(
            label: 'Method',
            value: _q.method,
            options: const {
              'upi': 'UPI',
              'bank_transfer': 'Bank Transfer',
              'card': 'Credit / Debit Card',
              'cash': 'Cash',
            },
            onChanged: (v) => setState(
              () => _q = (method: v, status: _q.status, invoiceId: _q.invoiceId, customerId: _q.customerId, page: 1),
            ),
          ),
          FilterDropdown<String>(
            label: 'Status',
            value: _q.status,
            options: const {
              'completed': 'Completed',
              'refunded': 'Refunded',
              'pending': 'Pending',
              'failed': 'Failed',
            },
            onChanged: (v) => setState(
              () => _q = (method: _q.method, status: v, invoiceId: _q.invoiceId, customerId: _q.customerId, page: 1),
            ),
          ),
        ],
      ),
      child: AsyncView(
        value: data,
        onRetry: () => ref.invalidate(paymentsProvider),
        builder: (page) => Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (page.items.isEmpty)
              const EmptyState(
                icon: Icons.payments_outlined,
                title: 'No payments found',
                message: 'No payment records match the current filters.',
              ),
            for (final p in page.items)
              ResourceTile(
                leading: CircleAvatar(
                  backgroundColor: p.isCompleted
                      ? const Color(0xFFE3F4EA)
                      : const Color(0xFFF1F5F9),
                  child: Icon(
                    Icons.payment_outlined,
                    color: p.isCompleted ? const Color(0xFF0F8A52) : const Color(0xFF64748B),
                  ),
                ),
                title: '${p.paymentNumber} · Invoice ${p.invoiceNumber}',
                subtitle: '${p.customerName} · ${p.method.toUpperCase()} · Date: ${p.paymentDate}${p.reference != null ? " · Ref: ${p.reference}" : ""}',
                pills: [
                  Pill(p.status.toUpperCase(), tone: p.isCompleted ? 'good' : 'warn'),
                  Pill(Fmt.inr(p.amount), tone: 'brand'),
                ],
                onTap: () => _viewDetails(p),
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (!isCustomer && p.isCompleted)
                      IconButton(
                        tooltip: 'Refund Payment',
                        icon: const Icon(Icons.replay_outlined, color: Colors.deepOrange),
                        onPressed: () => _refund(p.id),
                      ),
                    IconButton(
                      tooltip: 'View Details',
                      icon: const Icon(Icons.visibility_outlined),
                      onPressed: () => _viewDetails(p),
                    ),
                  ],
                ),
              ),
            Pager(
              page: page.page,
              pageSize: page.pageSize,
              total: page.total,
              onPage: (pg) => setState(
                () => _q = (method: _q.method, status: _q.status, invoiceId: _q.invoiceId, customerId: _q.customerId, page: pg),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _refund(int id) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Refund Payment'),
        content: const Text('Are you sure you want to mark this payment as refunded?'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.deepOrange),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Refund Payment'),
          ),
        ],
      ),
    );
    if (ok != true) return;

    try {
      await ref.read(paymentsRepositoryProvider).refund(id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Payment refunded successfully')),
        );
        ref.invalidate(paymentsProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Failed: $e')));
      }
    }
  }

  Future<void> _viewDetails(Payment p) async {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('${p.paymentNumber} Details'),
        content: SizedBox(
          width: 500,
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                _row('Invoice Number', p.invoiceNumber),
                _row('Customer', p.customerName),
                _row('Payment Method', p.method.toUpperCase()),
                _row('Amount Paid', Fmt.inr(p.amount), isBold: true),
                _row('Payment Date', p.paymentDate),
                _row('Status', p.status.toUpperCase()),
                if (p.reference != null && p.reference!.isNotEmpty)
                  _row('Reference / Txn ID', p.reference!),
                if (p.notes != null && p.notes!.isNotEmpty)
                  _row('Notes', p.notes!),
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
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label, style: TextStyle(color: Colors.grey[700], fontWeight: isBold ? FontWeight.bold : FontWeight.normal)),
            Text(value, style: TextStyle(fontWeight: isBold ? FontWeight.bold : FontWeight.normal)),
          ],
        ),
      );
}
