import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/utils/razorpay_checkout.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../auth/application/auth_controller.dart';
import '../data/invoices_api.dart';

class InvoicesScreen extends ConsumerStatefulWidget {
  const InvoicesScreen({super.key});
  @override
  ConsumerState<InvoicesScreen> createState() => _InvoicesScreenState();
}


class _InvoicesScreenState extends ConsumerState<InvoicesScreen> {
  InvoiceQuery _q = (q: '', status: null, overdue: null, customerId: null, page: 1);

  @override
  Widget build(BuildContext context) {
    final role = ref.watch(authControllerProvider).value?.role ?? Role.customer;
    final isCustomer = role == Role.customer;
    final data = ref.watch(invoicesProvider(_q));

    return PageFrame(
      title: 'Invoices',
      subtitle: isCustomer
          ? 'View and pay your subscription invoices online.'
          : 'Filter by status, track overdue accounts, send reminders, record payments or collect via Razorpay.',
      toolbar: Toolbar(
        children: [
          SearchBox(
            hint: 'Search invoice or customer',
            onChanged: (v) => setState(
              () => _q = (q: v, status: _q.status, overdue: _q.overdue, customerId: _q.customerId, page: 1),
            ),
          ),
          FilterDropdown<String>(
            label: 'Status',
            value: _q.status,
            options: const {
              'draft': 'Draft',
              'confirmed': 'Confirmed',
              'paid': 'Paid',
              'cancelled': 'Cancelled',
            },
            onChanged: (v) => setState(
              () => _q = (q: _q.q, status: v, overdue: _q.overdue, customerId: _q.customerId, page: 1),
            ),
          ),
          FilterDropdown<bool>(
            label: 'Overdue',
            value: _q.overdue,
            options: const {
              true: 'Overdue Only',
              false: 'Not Overdue',
            },
            onChanged: (v) => setState(
              () => _q = (q: _q.q, status: _q.status, overdue: v, customerId: _q.customerId, page: 1),
            ),
          ),
        ],
      ),
      child: AsyncView(
        value: data,
        onRetry: () => ref.invalidate(invoicesProvider),
        builder: (page) => Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (page.items.isEmpty)
              const EmptyState(
                icon: Icons.receipt_long_outlined,
                title: 'No invoices found',
                message: 'No invoices match the current filters.',
              ),
            for (final inv in page.items)
              ResourceTile(
                leading: CircleAvatar(
                  backgroundColor: _statusTone(inv.status, inv.isOverdue) == 'good'
                      ? const Color(0xFFE3F4EA)
                      : _statusTone(inv.status, inv.isOverdue) == 'bad'
                          ? const Color(0xFFFBE4E2)
                          : const Color(0xFFF1F5F9),
                  child: Icon(
                    Icons.receipt_outlined,
                    color: _statusTone(inv.status, inv.isOverdue) == 'good'
                        ? const Color(0xFF0F8A52)
                        : _statusTone(inv.status, inv.isOverdue) == 'bad'
                            ? const Color(0xFFB3261E)
                            : const Color(0xFF64748B),
                  ),
                ),
                title: '${inv.invoiceNumber} · ${inv.customerName}',
                subtitle: 'Issue Date: ${inv.invoiceDate} · Due: ${inv.dueDate} · Amount Due: ${Fmt.inr(inv.amountDue)}',
                pills: [
                  Pill(inv.status.toUpperCase(), tone: _statusTone(inv.status, inv.isOverdue)),
                  if (inv.isOverdue) const Pill('OVERDUE', tone: 'bad'),
                  Pill(Fmt.inr(inv.total), tone: 'brand'),
                ],
                onTap: () => _viewDetails(inv),
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (!isCustomer && inv.isDraft) ...[
                      IconButton(
                        tooltip: 'Confirm Invoice',
                        icon: const Icon(Icons.check_circle_outline, color: Colors.green),
                        onPressed: () => _confirm(inv.id),
                      ),
                      IconButton(
                        tooltip: 'Cancel Invoice',
                        icon: const Icon(Icons.cancel_outlined, color: Colors.red),
                        onPressed: () => _cancel(inv.id),
                      ),
                    ],
                    if (!isCustomer && inv.isConfirmed) ...[
                      IconButton(
                        tooltip: 'Send Invoice',
                        icon: const Icon(Icons.send_outlined),
                        onPressed: () => _send(inv.id),
                      ),
                      FilledButton.tonalIcon(
                        icon: const Icon(Icons.payments_outlined, size: 16),
                        label: const Text('Record Pay'),
                        onPressed: () => _recordPaymentDialog(inv),
                      ),
                      const SizedBox(width: 8),
                    ],
                    if (inv.isConfirmed && (inv.amountDue != '0.00' && inv.amountDue != '0'))
                      FilledButton.icon(
                        style: FilledButton.styleFrom(
                          backgroundColor: const Color(0xFF0B64F4), // Razorpay blue
                        ),
                        icon: const Icon(Icons.credit_card, size: 16),
                        label: const Text('Pay with Razorpay'),
                        onPressed: () => _payWithRazorpay(inv),
                      ),
                    IconButton(
                      tooltip: 'View Details',
                      icon: const Icon(Icons.visibility_outlined),
                      onPressed: () => _viewDetails(inv),
                    ),
                  ],
                ),
              ),
            Pager(
              page: page.page,
              pageSize: page.pageSize,
              total: page.total,
              onPage: (p) => setState(
                () => _q = (q: _q.q, status: _q.status, overdue: _q.overdue, customerId: _q.customerId, page: p),
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _statusTone(String status, bool isOverdue) {
    if (isOverdue && status != 'paid' && status != 'cancelled') return 'bad';
    return switch (status) {
      'paid' => 'good',
      'confirmed' => 'brand',
      'draft' => 'neutral',
      'cancelled' => 'bad',
      _ => 'neutral',
    };
  }

  Future<void> _confirm(int id) async {
    try {
      await ref.read(invoicesRepositoryProvider).confirm(id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Invoice confirmed successfully')),
        );
        ref.invalidate(invoicesProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Failed: $e')));
      }
    }
  }

  Future<void> _send(int id) async {
    try {
      await ref.read(invoicesRepositoryProvider).send(id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Invoice sent to customer email')),
        );
        ref.invalidate(invoicesProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Failed: $e')));
      }
    }
  }

  Future<void> _cancel(int id) async {
    final reasonCtrl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Cancel Invoice'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text('Are you sure you want to cancel this invoice?'),
            const SizedBox(height: 12),
            TextField(
              controller: reasonCtrl,
              decoration: const InputDecoration(labelText: 'Cancel Reason (Optional)'),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Back')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Cancel Invoice'),
          ),
        ],
      ),
    );
    if (ok != true) return;

    try {
      await ref.read(invoicesRepositoryProvider).cancel(id, reason: reasonCtrl.text.trim());
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Invoice cancelled')),
        );
        ref.invalidate(invoicesProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Failed: $e')));
      }
    }
  }

  Future<void> _recordPaymentDialog(Invoice inv) async {
    String method = 'upi';
    final amtCtrl = TextEditingController(text: inv.amountDue);
    final dateCtrl = TextEditingController(text: Fmt.apiDate(DateTime.now()));
    final refCtrl = TextEditingController();
    final notesCtrl = TextEditingController();

    final saved = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) => AlertDialog(
          title: Text('Record Payment for ${inv.invoiceNumber}'),
          content: SizedBox(
            width: 480,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Remaining Due: ${Fmt.inr(inv.amountDue)}',
                      style: const TextStyle(fontWeight: FontWeight.bold, color: Colors.indigo)),
                  const SizedBox(height: 16),
                  const Text('Payment Method', style: TextStyle(fontWeight: FontWeight.w500)),
                  const SizedBox(height: 4),
                  DropdownButtonFormField<String>(
                    initialValue: method,
                    items: const [
                      DropdownMenuItem(value: 'upi', child: Text('UPI')),
                      DropdownMenuItem(value: 'bank_transfer', child: Text('Bank Transfer')),
                      DropdownMenuItem(value: 'card', child: Text('Credit / Debit Card')),
                      DropdownMenuItem(value: 'cash', child: Text('Cash')),
                    ],
                    onChanged: (v) => setDialogState(() => method = v!),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: amtCtrl,
                    keyboardType: const TextInputType.numberWithOptions(decimal: true),
                    decoration: const InputDecoration(labelText: 'Amount (₹)'),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: dateCtrl,
                    decoration: const InputDecoration(labelText: 'Payment Date (YYYY-MM-DD)'),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: refCtrl,
                    decoration: const InputDecoration(labelText: 'Reference / Transaction ID (Optional)'),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: notesCtrl,
                    decoration: const InputDecoration(labelText: 'Notes (Optional)'),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
            FilledButton(
              onPressed: () async {
                try {
                  await ref.read(invoicesRepositoryProvider).recordPayment(inv.id, {
                    'method': method,
                    'amount': amtCtrl.text.trim(),
                    'payment_date': dateCtrl.text.trim(),
                    'reference': refCtrl.text.trim(),
                    'notes': notesCtrl.text.trim(),
                  });
                  if (ctx.mounted) Navigator.pop(ctx, true);
                } catch (e) {
                  if (ctx.mounted) {
                    ScaffoldMessenger.of(ctx).showSnackBar(
                      SnackBar(content: Text('Failed to record payment: $e')),
                    );
                  }
                }
              },
              child: const Text('Record Payment'),
            ),
          ],
        ),
      ),
    );

    if (saved == true && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Payment recorded successfully!')),
      );
      ref.invalidate(invoicesProvider);
    }
  }

  Future<void> _payWithRazorpay(Invoice inv) async {
    try {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Initializing Razorpay checkout...')),
      );

      final order = await ref.read(invoicesRepositoryProvider).createRazorpayOrder(inv.id);

      openRazorpayCheckout(
        key: order.keyId,
        amount: order.amount,
        currency: order.currency,
        name: 'Subly / Brightloop',
        description: 'Payment for Invoice ${order.invoiceNumber}',
        orderId: order.orderId,
        prefillName: order.customerName,
        prefillEmail: order.customerEmail,
        onSuccess: (paymentId, orderId, signature) async {
          try {
            await ref.read(invoicesRepositoryProvider).verifyRazorpayPayment(inv.id, {
              'razorpay_payment_id': paymentId,
              'razorpay_order_id': orderId,
              'razorpay_signature': signature,
            });
            if (mounted) {
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('Payment successful and verified with Razorpay!'),
                  backgroundColor: Colors.green,
                ),
              );
              ref.invalidate(invoicesProvider);
            }
          } catch (e) {
            if (mounted) {
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text('Payment verification failed: $e')),
              );
            }
          }
        },
        onFailure: (error) {
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text('Payment not completed: $error')),
            );
          }
        },
      );
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Could not start Razorpay: $e')),
        );
      }
    }
  }

  Future<void> _viewDetails(Invoice inv) async {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('${inv.invoiceNumber} Details'),
        content: SizedBox(
          width: 550,
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                _row('Customer', inv.customerName),
                if (inv.billingAddress != null && inv.billingAddress!.isNotEmpty)
                  _row('Billing Address', inv.billingAddress!),
                if (inv.customerGstin != null && inv.customerGstin!.isNotEmpty)
                  _row('GSTIN', inv.customerGstin!),
                if (inv.subscriptionNumber != null)
                  _row('Subscription', inv.subscriptionNumber!),
                _row('Status', inv.status.toUpperCase()),
                _row('Issue Date', inv.invoiceDate),
                _row('Due Date', inv.dueDate),
                if (inv.isOverdue)
                  _row('Overdue Status', 'OVERDUE', isAlert: true),
                const Divider(height: 24),
                const Text('Line Items', style: TextStyle(fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                for (final item in inv.items)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 4),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text('${item.description} × ${item.quantity}'),
                        Text(Fmt.inr(item.lineTotal)),
                      ],
                    ),
                  ),
                const Divider(height: 24),
                _row('Subtotal', Fmt.inr(inv.subtotal)),
                if (inv.discountTotal != '0.00' && inv.discountTotal != '0')
                  _row('Discount', '- ${Fmt.inr(inv.discountTotal)}'),
                _row('Tax Total', Fmt.inr(inv.taxTotal)),
                _row('Total', Fmt.inr(inv.total), isBold: true),
                const Divider(height: 16),
                _row('Amount Paid', Fmt.inr(inv.amountPaid)),
                _row('Amount Due', Fmt.inr(inv.amountDue), isBold: true),
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

  Widget _row(String label, String value, {bool isBold = false, bool isAlert = false}) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 3),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label,
                style: TextStyle(
                    color: isAlert ? Colors.red : Colors.grey[700],
                    fontWeight: isBold ? FontWeight.bold : FontWeight.normal)),
            Text(value,
                style: TextStyle(
                    color: isAlert ? Colors.red : null,
                    fontWeight: isBold || isAlert ? FontWeight.bold : FontWeight.normal)),
          ],
        ),
      );
}
