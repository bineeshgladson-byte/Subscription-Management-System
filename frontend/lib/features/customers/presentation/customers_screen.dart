import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/utils/validators.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../auth/application/auth_controller.dart';
import '../data/customers_api.dart';

/// Customers (admin + staff). Staff create and edit; only the admin archives.
class CustomersScreen extends ConsumerStatefulWidget {
  const CustomersScreen({super.key});
  @override
  ConsumerState<CustomersScreen> createState() => _CustomersScreenState();
}

class _CustomersScreenState extends ConsumerState<CustomersScreen> {
  CustomerQuery _q = (q: '', hasLogin: null, page: 1);

  @override
  Widget build(BuildContext context) {
    final isAdmin = ref.watch(authControllerProvider).value?.role == Role.admin;
    final data = ref.watch(customersProvider(_q));
    return PageFrame(
      title: 'Customers',
      subtitle:
          'Businesses and people you bill. Customers who sign up with the same email are linked automatically.',
      action: FilledButton.icon(
        onPressed: () => _edit(null),
        icon: const Icon(Icons.add),
        label: const Text('New customer'),
      ),
      toolbar: Toolbar(
        children: [
          SearchBox(
            hint: 'Search name, email, company or phone',
            onChanged: (v) =>
                setState(() => _q = (q: v, hasLogin: _q.hasLogin, page: 1)),
          ),
          FilterDropdown<bool>(
            label: 'Portal login',
            value: _q.hasLogin,
            options: const {true: 'Has signed up', false: 'Added by staff'},
            onChanged: (v) =>
                setState(() => _q = (q: _q.q, hasLogin: v, page: 1)),
          ),
        ],
      ),
      child: AsyncView(
        value: data,
        onRetry: () => ref.invalidate(customersProvider),
        builder: (page) => Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (page.items.isEmpty)
              const EmptyState(
                icon: Icons.people_outline,
                title: 'No customers found',
                message: 'Add a customer, or change the search.',
              ),
            for (final c in page.items)
              ResourceTile(
                leading: CircleAvatar(
                  child: Text(c.name.isEmpty ? '?' : c.name[0].toUpperCase()),
                ),
                title: c.name,
                subtitle: [
                  c.email,
                  c.companyName,
                  c.phone,
                ].whereType<String>().join(' · '),
                pills: [
                  Pill(
                    c.hasLogin ? 'Portal login' : 'No login',
                    tone: c.hasLogin ? 'good' : 'neutral',
                  ),
                  if (c.gstin != null) Pill('GSTIN ${c.gstin}'),
                ],
                onTap: () => _detail(c.id, isAdmin),
                trailing: IconButton(
                  tooltip: 'Edit ${c.name}',
                  icon: const Icon(Icons.edit_outlined),
                  onPressed: () => _edit(c),
                ),
              ),
            Pager(
              page: page.page,
              pageSize: page.pageSize,
              total: page.total,
              onPage: (p) => setState(
                () => _q = (q: _q.q, hasLogin: _q.hasLogin, page: p),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _edit(Customer? c) async {
    final name = TextEditingController(text: c?.name);
    final email = TextEditingController(text: c?.email);
    final phone = TextEditingController(text: c?.phone);
    final company = TextEditingController(text: c?.companyName);
    final address = TextEditingController(text: c?.billingAddress);
    final gstin = TextEditingController(text: c?.gstin);
    final repo = ref.read(customersRepositoryProvider);
    final saved = await showApiForm<bool>(
      context,
      title: c == null ? 'New customer' : 'Edit ${c.name}',
      fields: (_, _, e) => [
        formField(
          name,
          'Name',
          errors: e,
          errorKey: 'name',
          validator: requiredValidator,
        ),
        formField(
          email,
          'Email',
          errors: e,
          errorKey: 'email',
          validator: Validators.email,
          keyboard: TextInputType.emailAddress,
        ),
        if (c?.hasLogin == true)
          const Text(
            'This customer logs in with this email, so it can\'t be changed here.',
            style: TextStyle(fontSize: 11),
          ),
        formField(
          phone,
          'Phone',
          errors: e,
          errorKey: 'phone',
          keyboard: TextInputType.phone,
        ),
        formField(company, 'Company', errors: e, errorKey: 'company_name'),
        formField(
          address,
          'Billing address',
          errors: e,
          errorKey: 'billing_address',
          maxLines: 3,
        ),
        formField(
          gstin,
          'GSTIN',
          errors: e,
          errorKey: 'gstin',
          hint: '29ABCDE1234F1Z5',
        ),
      ],
      onSave: () async {
        final body = {
          'name': name.text.trim(),
          'email': email.text.trim(),
          'phone': phone.text.trim(),
          'company_name': company.text.trim(),
          'billing_address': address.text.trim(),
          'gstin': gstin.text.trim(),
        };
        if (c == null) {
          body.removeWhere((_, v) => v.isEmpty);
          await repo.create(body);
        } else {
          if (c.hasLogin) body.remove('email');
          await repo.update(c.id, body);
        }
        return true;
      },
    );
    for (final t in [name, email, phone, company, address, gstin]) {
      t.dispose();
    }
    if (saved == true && mounted) {
      ref.invalidate(customersProvider);
      showMessage(context, c == null ? 'Customer added' : 'Customer updated');
    }
  }

  Future<void> _detail(int id, bool isAdmin) => showDialog<void>(
    context: context,
    builder: (dialogContext) => Consumer(
      builder: (context, ref, _) => AlertDialog(
        title: const Text('Customer'),
        content: SizedBox(
          width: 480,
          child: AsyncView(
            value: ref.watch(customerDetailProvider(id)),
            builder: (c) => Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(c.name, style: Theme.of(context).textTheme.titleLarge),
                const SizedBox(height: 8),
                InfoRow('Email', c.email),
                InfoRow('Phone', c.phone ?? '—'),
                InfoRow('Company', c.companyName ?? '—'),
                InfoRow('Billing address', c.billingAddress ?? '—'),
                InfoRow('GSTIN', c.gstin ?? '—'),
                InfoRow('Portal login', c.hasLogin ? 'Yes' : 'No'),
                const Divider(),
                if (c.stats != null) ...[
                  InfoRow(
                    'Active subscriptions',
                    '${c.stats!.activeSubscriptions} of ${c.stats!.totalSubscriptions}',
                  ),
                  InfoRow(
                    'Open invoices',
                    '${c.stats!.openInvoices} (${c.stats!.overdueInvoices} overdue)',
                  ),
                  InfoRow('Amount due', Fmt.inr(c.stats!.amountDue)),
                ],
              ],
            ),
          ),
        ),
        actions: [
          if (isAdmin)
            TextButton(
              onPressed: () async {
                if (!await confirmAction(
                  dialogContext,
                  title: 'Archive customer?',
                  message:
                      'They disappear from lists and their portal login is disabled. Invoices are kept.',
                  confirmLabel: 'Archive',
                  danger: true,
                )) {
                  return;
                }
                if (!dialogContext.mounted) return;
                final ok = await runAction(
                  dialogContext,
                  () => ref.read(customersRepositoryProvider).archive(id),
                  success: 'Customer archived',
                );
                if (ok && dialogContext.mounted) {
                  ref.invalidate(customersProvider);
                  Navigator.pop(dialogContext);
                }
              },
              child: const Text(
                'Archive',
                style: TextStyle(color: Color(0xFFB3261E)),
              ),
            ),
          TextButton(
            onPressed: () => Navigator.pop(dialogContext),
            child: const Text('Close'),
          ),
        ],
      ),
    ),
  );
}
