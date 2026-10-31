import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/utils/validators.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../data/staff_api.dart';

/// Staff accounts (admin only: "Only the admin can add a new member of staff").
class StaffScreen extends ConsumerStatefulWidget {
  const StaffScreen({super.key});
  @override
  ConsumerState<StaffScreen> createState() => _StaffScreenState();
}

class _StaffScreenState extends ConsumerState<StaffScreen> {
  StaffQuery _q = (q: '', active: null, page: 1);

  @override
  Widget build(BuildContext context) {
    final data = ref.watch(staffProvider(_q));
    return PageFrame(
      title: 'Staff',
      subtitle:
          'Your internal team. Deactivating someone signs them out immediately.',
      action: FilledButton.icon(
        onPressed: () => _edit(null),
        icon: const Icon(Icons.person_add_alt),
        label: const Text('Add staff'),
      ),
      toolbar: Toolbar(
        children: [
          SearchBox(
            hint: 'Search name or email',
            onChanged: (v) =>
                setState(() => _q = (q: v, active: _q.active, page: 1)),
          ),
          FilterDropdown<bool>(
            label: 'Status',
            value: _q.active,
            options: const {true: 'Active', false: 'Deactivated'},
            onChanged: (v) =>
                setState(() => _q = (q: _q.q, active: v, page: 1)),
          ),
        ],
      ),
      child: AsyncView(
        value: data,
        onRetry: () => ref.invalidate(staffProvider),
        builder: (page) => Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (page.items.isEmpty)
              const EmptyState(
                icon: Icons.badge_outlined,
                title: 'No staff yet',
                message:
                    'Add a team member; they get an email to set their password.',
              ),
            for (final s in page.items)
              ResourceTile(
                leading: CircleAvatar(
                  child: Text(
                    s.fullName.isEmpty ? '?' : s.fullName[0].toUpperCase(),
                  ),
                ),
                title: s.fullName,
                subtitle: s.email,
                pills: [
                  Pill(
                    s.isActive ? 'Active' : 'Deactivated',
                    tone: s.isActive ? 'good' : 'bad',
                  ),
                  Pill(
                    s.lastLoginAt == null
                        ? 'Never logged in'
                        : 'Last login ${s.lastLoginAt!.toLocal().toString().substring(0, 16)}',
                  ),
                ],
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    IconButton(
                      tooltip: 'Edit ${s.fullName}',
                      icon: const Icon(Icons.edit_outlined),
                      onPressed: () => _edit(s),
                    ),
                    Switch(
                      value: s.isActive,
                      onChanged: (v) async {
                        if (!v &&
                            !await confirmAction(
                              context,
                              title: 'Deactivate ${s.fullName}?',
                              message:
                                  'They are signed out at once and can\'t log in until reactivated.',
                              confirmLabel: 'Deactivate',
                              danger: true,
                            )) {
                          return;
                        }
                        if (!context.mounted) return;
                        if (await runAction(
                          context,
                          () => ref
                              .read(staffRepositoryProvider)
                              .setActive(s.id, v),
                          success: v ? 'Reactivated' : 'Deactivated',
                        )) {
                          ref.invalidate(staffProvider);
                        }
                      },
                    ),
                  ],
                ),
              ),
            Pager(
              page: page.page,
              pageSize: page.pageSize,
              total: page.total,
              onPage: (p) =>
                  setState(() => _q = (q: _q.q, active: _q.active, page: p)),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _edit(StaffMember? s) async {
    final name = TextEditingController(text: s?.fullName);
    final email = TextEditingController(text: s?.email);
    final password = TextEditingController();
    final repo = ref.read(staffRepositoryProvider);
    var invited = false;
    final saved = await showApiForm<bool>(
      context,
      title: s == null ? 'Add staff member' : 'Edit ${s.fullName}',
      fields: (_, _, e) => [
        formField(
          name,
          'Full name',
          errors: e,
          errorKey: 'full_name',
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
        if (s == null) ...[
          formField(
            password,
            'Password (optional)',
            errors: e,
            errorKey: 'password',
            obscure: true,
            validator: (v) => (v ?? '').isEmpty ? null : Validators.password(v),
          ),
          const Text(
            'Leave the password empty to email them a link to set their own (recommended).',
            style: TextStyle(fontSize: 11),
          ),
        ],
      ],
      onSave: () async {
        if (s == null) {
          invited = await repo.create({
            'full_name': name.text.trim(),
            'email': email.text.trim(),
            if (password.text.isNotEmpty) 'password': password.text,
          });
        } else {
          await repo.update(s.id, {
            'full_name': name.text.trim(),
            'email': email.text.trim(),
          });
        }
        return true;
      },
    );
    for (final t in [name, email, password]) {
      t.dispose();
    }
    if (saved == true && mounted) {
      ref.invalidate(staffProvider);
      showMessage(
        context,
        s != null
            ? 'Staff member updated'
            : invited
            ? 'Staff member added · invite email sent'
            : 'Staff member added',
      );
    }
  }
}
