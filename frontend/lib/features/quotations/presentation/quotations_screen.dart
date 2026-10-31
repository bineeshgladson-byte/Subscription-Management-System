import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

<<<<<<< HEAD
import '../../../core/models/role.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../auth/application/auth_controller.dart';
import '../../customers/data/customers_api.dart';
import '../../plans/data/plans_api.dart';
import '../data/quotations_api.dart';

=======
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
>>>>>>> origin/main
class QuotationsScreen extends ConsumerStatefulWidget {
  const QuotationsScreen({super.key});
  @override
  ConsumerState<QuotationsScreen> createState() => _QuotationsScreenState();
}

class _QuotationsScreenState extends ConsumerState<QuotationsScreen> {
  SubQuery _q = (q: '', status: null, page: 1);

  @override
  ConsumerState<QuotationsScreen> createState() => _QuotationsScreenState();
}

class _QuotationsScreenState extends ConsumerState<QuotationsScreen> {
  QuotationQuery _q = (q: '', status: null, customerId: null, page: 1);

  @override
  Widget build(BuildContext context) {
    final role = ref.watch(authControllerProvider).value?.role ?? Role.customer;
    final isCustomer = role == Role.customer;
    final data = ref.watch(quotationsProvider(_q));

    return PageFrame(
      title: 'Quotations',
      subtitle: isCustomer
          ? 'Quotations offered to you. Confirm to activate your subscription.'
          : 'Create from a template or plan, send to customers, and confirm into active subscriptions.',
      action: !isCustomer
          ? FilledButton.icon(
              onPressed: _newQuotation,
              icon: const Icon(Icons.add),
              label: const Text('New Quotation'),
            )
          : null,
      toolbar: Toolbar(
        children: [
          SearchBox(
            hint: 'Search number or customer',
            onChanged: (v) => setState(
              () => _q = (q: v, status: _q.status, customerId: _q.customerId, page: 1),
            ),
          ),
          FilterDropdown<String>(
            label: 'Status',
            value: _q.status,
            options: const {
              'draft': 'Draft',
              'quotation': 'Quotation',
              'confirmed': 'Confirmed',
              'cancelled': 'Cancelled',
            },
            onChanged: (v) => setState(
              () => _q = (q: _q.q, status: v, customerId: _q.customerId, page: 1),
            ),
          ),
        ],
      ),
      child: AsyncView(
        value: data,
        onRetry: () => ref.invalidate(quotationsProvider),
        builder: (page) => Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (page.items.isEmpty)
              const EmptyState(
                icon: Icons.request_quote_outlined,
                title: 'No quotations found',
                message: 'No quotations match the selected filters.',
              ),
            for (final item in page.items)
              ResourceTile(
                leading: CircleAvatar(
                  backgroundColor: _statusTone(item.status) == 'good'
                      ? const Color(0xFFE3F4EA)
                      : const Color(0xFFF1F5F9),
                  child: Icon(
                    Icons.description_outlined,
                    color: _statusTone(item.status) == 'good'
                        ? const Color(0xFF0F8A52)
                        : const Color(0xFF64748B),
                  ),
                ),
                title: '${item.quotationNumber} · ${item.customerName}',
                subtitle: '${item.planName} (${item.billingPeriod}) · Start: ${item.startDate} · Valid until: ${item.quotationValidUntil ?? 'N/A'}',
                pills: [
                  Pill(
                    item.status.toUpperCase(),
                    tone: _statusTone(item.status),
                  ),
                  Pill(Fmt.inr(item.total), tone: 'brand'),
                ],
                onTap: () => _viewDetails(item),
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (!isCustomer && (item.isDraft || item.isQuotation)) ...[
                      IconButton(
                        tooltip: 'Send Quotation',
                        icon: const Icon(Icons.send_outlined),
                        onPressed: () => _send(item.id),
                      ),
                      IconButton(
                        tooltip: 'Confirm to Subscription',
                        icon: const Icon(Icons.check_circle_outline, color: Colors.green),
                        onPressed: () => _confirm(item.id),
                      ),
                      IconButton(
                        tooltip: 'Cancel',
                        icon: const Icon(Icons.cancel_outlined, color: Colors.red),
                        onPressed: () => _cancel(item.id),
                      ),
                    ],
                    if (isCustomer && item.isQuotation)
                      FilledButton.tonal(
                        onPressed: () => _confirm(item.id),
                        child: const Text('Confirm'),
                      ),
                    IconButton(
                      tooltip: 'View Details',
                      icon: const Icon(Icons.visibility_outlined),
                      onPressed: () => _viewDetails(item),
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
<<<<<<< HEAD
      ),
=======
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
>>>>>>> origin/main
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

  String _statusTone(String status) => switch (status) {
        'confirmed' => 'good',
        'quotation' => 'warn',
        'draft' => 'neutral',
        'cancelled' => 'bad',
        _ => 'neutral',
      };

  Future<void> _send(int id) async {
    try {
      await ref.read(quotationsRepositoryProvider).send(id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Quotation sent successfully')),
        );
        ref.invalidate(quotationsProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to send: $e')),
        );
      }
    }
  }

  Future<void> _confirm(int id) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Confirm Quotation'),
        content: const Text(
          'Confirming this quotation will turn it into an active subscription and schedule recurring billing. Proceed?',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Confirm')),
        ],
      ),
    );
    if (ok != true) return;

    try {
      await ref.read(quotationsRepositoryProvider).confirm(id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Quotation confirmed into Subscription!')),
        );
        ref.invalidate(quotationsProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to confirm: $e')),
        );
      }
    }
  }

  Future<void> _cancel(int id) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Cancel Quotation'),
        content: const Text('Are you sure you want to cancel this quotation?'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Back')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Cancel Quotation'),
          ),
        ],
      ),
    );
    if (ok != true) return;

    try {
      await ref.read(quotationsRepositoryProvider).cancel(id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Quotation cancelled')),
        );
        ref.invalidate(quotationsProvider);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to cancel: $e')),
        );
      }
    }
  }

  Future<void> _viewDetails(Quotation item) async {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('${item.quotationNumber} Details'),
        content: SizedBox(
          width: 550,
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                _row('Customer', item.customerName),
                _row('Email', item.customerEmail),
                _row('Plan', '${item.planName} (${item.billingPeriod})'),
                _row('Status', item.status.toUpperCase()),
                _row('Start Date', item.startDate),
                _row('Valid Until', item.quotationValidUntil ?? 'N/A'),
                _row('Payment Terms', '${item.paymentTermsDays} days'),
                const Divider(height: 24),
                const Text('Line Items', style: TextStyle(fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                for (final line in item.items)
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
                _row('Subtotal', Fmt.inr(item.subtotal)),
                if (item.discountTotal != '0.00' && item.discountTotal != '0')
                  _row('Discount', '- ${Fmt.inr(item.discountTotal)}'),
                _row('Tax', Fmt.inr(item.taxTotal)),
                const SizedBox(height: 6),
                _row('Total', Fmt.inr(item.total), isBold: true),
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

  Future<void> _newQuotation() async {
    final customersRes = await ref.read(customersRepositoryProvider).list((q: '', hasLogin: null, page: 1));
    final plansRes = await ref.read(activePlansProvider.future);

    if (!mounted) return;
    if (customersRes.items.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Please create at least one customer first')),
      );
      return;
    }
    if (plansRes.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Please create at least one active plan first')),
      );
      return;
    }

    int selectedCustomer = customersRes.items.first.id;
    int selectedPlan = plansRes.first.id;
    final startDateCtrl = TextEditingController(text: Fmt.apiDate(DateTime.now()));
    final termsCtrl = TextEditingController(text: '15');

    final created = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) => AlertDialog(
          title: const Text('Create New Quotation'),
          content: SizedBox(
            width: 480,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text('Customer', style: TextStyle(fontWeight: FontWeight.w500)),
                  const SizedBox(height: 4),
                  DropdownButtonFormField<int>(
                    initialValue: selectedCustomer,
                    isExpanded: true,
                    items: customersRes.items
                        .map((c) => DropdownMenuItem(value: c.id, child: Text('${c.name} (${c.email})')))
                        .toList(),
                    onChanged: (v) => setDialogState(() => selectedCustomer = v!),
                  ),
                  const SizedBox(height: 16),
                  const Text('Subscription Plan', style: TextStyle(fontWeight: FontWeight.w500)),
                  const SizedBox(height: 4),
                  DropdownButtonFormField<int>(
                    initialValue: selectedPlan,
                    isExpanded: true,
                    items: plansRes
                        .map((p) => DropdownMenuItem(value: p.id, child: Text('${p.name} · ₹${p.price} (${p.billingPeriod})')))
                        .toList(),
                    onChanged: (v) => setDialogState(() => selectedPlan = v!),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: startDateCtrl,
                    decoration: const InputDecoration(
                      labelText: 'Start Date (YYYY-MM-DD)',
                      hintText: '2026-11-01',
                    ),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: termsCtrl,
                    keyboardType: TextInputType.number,
                    decoration: const InputDecoration(
                      labelText: 'Payment Terms (Days)',
                      hintText: '15',
                    ),
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
                  await ref.read(quotationsRepositoryProvider).create({
                    'customer_id': selectedCustomer,
                    'plan_id': selectedPlan,
                    'start_date': startDateCtrl.text.trim(),
                    'payment_terms_days': int.tryParse(termsCtrl.text.trim()) ?? 15,
                  });
                  if (ctx.mounted) Navigator.pop(ctx, true);
                } catch (e) {
                  if (ctx.mounted) {
                    ScaffoldMessenger.of(ctx).showSnackBar(
                      SnackBar(content: Text('Failed to create quotation: $e')),
                    );
                  }
                }
              },
              child: const Text('Create Quotation'),
            ),
          ],
        ),
      ),
    );

    if (created == true && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Quotation created successfully')),
      );
      ref.invalidate(quotationsProvider);
    }
  }
}
