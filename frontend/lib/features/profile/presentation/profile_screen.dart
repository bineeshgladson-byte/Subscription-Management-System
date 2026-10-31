import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/utils/validators.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../auth/application/auth_controller.dart';
import '../data/profile_api.dart';

class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) => PageFrame(
    title: 'Profile',
    subtitle: 'Your account details.',
    action: Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        OutlinedButton.icon(
          onPressed: () => _changePassword(context, ref),
          icon: const Icon(Icons.lock_outline),
          label: const Text('Change password'),
        ),
        FilledButton.icon(
          onPressed: () async {
            final p = await ref.read(profileProvider.future);
            if (context.mounted) await _edit(context, ref, p);
          },
          icon: const Icon(Icons.edit_outlined),
          label: const Text('Edit'),
        ),
      ],
    ),
    child: AsyncView(
      value: ref.watch(profileProvider),
      onRetry: () => ref.invalidate(profileProvider),
      builder: (p) => Card(
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(p.fullName, style: Theme.of(context).textTheme.titleLarge),
              const SizedBox(height: 8),
              InfoRow('Email', p.email),
              InfoRow('Role', p.role[0].toUpperCase() + p.role.substring(1)),
              if (p.isCustomer) ...[
                InfoRow('Phone', p.phone ?? '—'),
                InfoRow('Company', p.companyName ?? '—'),
                InfoRow('Billing address', p.billingAddress ?? '—'),
                InfoRow('GSTIN', p.gstin ?? '—'),
              ],
            ],
          ),
        ),
      ),
    ),
  );

  Future<void> _edit(BuildContext context, WidgetRef ref, Profile p) async {
    final name = TextEditingController(text: p.fullName);
    final phone = TextEditingController(text: p.phone);
    final company = TextEditingController(text: p.companyName);
    final address = TextEditingController(text: p.billingAddress);
    final gstin = TextEditingController(text: p.gstin);
    final saved = await showApiForm<bool>(
      context,
      title: 'Edit profile',
      fields: (_, _, e) => [
        formField(
          name,
          'Full name',
          errors: e,
          errorKey: 'full_name',
          validator: requiredValidator,
        ),
        if (p.isCustomer) ...[
          formField(phone, 'Phone', errors: e, errorKey: 'phone'),
          formField(company, 'Company', errors: e, errorKey: 'company_name'),
          formField(
            address,
            'Billing address',
            errors: e,
            errorKey: 'billing_address',
            maxLines: 3,
          ),
          formField(gstin, 'GSTIN', errors: e, errorKey: 'gstin'),
        ],
      ],
      onSave: () async {
        await ref.read(profileRepositoryProvider).update({
          'full_name': name.text.trim(),
          if (p.isCustomer) ...{
            'phone': phone.text.trim(),
            'company_name': company.text.trim(),
            'billing_address': address.text.trim(),
            'gstin': gstin.text.trim(),
          },
        });
        return true;
      },
    );
    for (final c in [name, phone, company, address, gstin]) {
      c.dispose();
    }
    if (saved == true && context.mounted) {
      ref.invalidate(profileProvider);
      await ref.read(authControllerProvider.notifier).refreshUser();
      if (context.mounted) showMessage(context, 'Profile updated');
    }
  }

  Future<void> _changePassword(BuildContext context, WidgetRef ref) async {
    final current = TextEditingController();
    final next = TextEditingController();
    final saved = await showApiForm<bool>(
      context,
      title: 'Change password',
      saveLabel: 'Change password',
      fields: (_, _, e) => [
        formField(
          current,
          'Current password',
          errors: e,
          errorKey: 'current_password',
          obscure: true,
          validator: requiredValidator,
        ),
        formField(
          next,
          'New password',
          errors: e,
          errorKey: 'new_password',
          obscure: true,
          validator: Validators.password,
        ),
        const Text(
          '9+ characters with an uppercase letter, a lowercase letter and a special character. Other devices are signed out.',
          style: TextStyle(fontSize: 11),
        ),
      ],
      onSave: () async {
        final token = await ref
            .read(profileRepositoryProvider)
            .changePassword(current.text, next.text);
        await ref.read(authControllerProvider.notifier).replaceToken(token);
        return true;
      },
    );
    current.dispose();
    next.dispose();
    if (saved == true && context.mounted) {
      showMessage(context, 'Password changed');
    }
  }
}
