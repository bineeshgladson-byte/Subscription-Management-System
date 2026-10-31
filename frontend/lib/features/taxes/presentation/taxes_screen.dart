import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../auth/application/auth_controller.dart';
import '../data/taxes_api.dart';

/// Taxes: "calculated automatically from rules the admin has set".
/// Admin manages them; staff can see them.
class TaxesScreen extends ConsumerWidget {
  const TaxesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isAdmin = ref.watch(authControllerProvider).value?.role == Role.admin;
    return PageFrame(
      title: 'Taxes',
      subtitle:
          'Rates applied to invoice lines. Changing a rate only affects new invoices.',
      action: isAdmin
          ? FilledButton.icon(
              onPressed: () => _edit(context, ref, null),
              icon: const Icon(Icons.add),
              label: const Text('New tax'),
            )
          : null,
      child: AsyncView(
        value: ref.watch(taxesProvider),
        onRetry: () => ref.invalidate(taxesProvider),
        builder: (page) => Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (page.items.isEmpty)
              const EmptyState(
                icon: Icons.percent,
                title: 'No taxes yet',
                message: 'Add GST 18% to get started.',
              ),
            for (final t in page.items)
              ResourceTile(
                leading: CircleAvatar(
                  child: Text(
                    '${double.tryParse(t.rate)?.round() ?? 0}%',
                    style: const TextStyle(fontSize: 11),
                  ),
                ),
                title: t.name,
                subtitle:
                    '${t.rate}% · used by ${t.productsUsing} active product${t.productsUsing == 1 ? '' : 's'}',
                pills: [
                  Pill(
                    t.isActive ? 'Active' : 'Inactive',
                    tone: t.isActive ? 'good' : 'neutral',
                  ),
                ],
                trailing: isAdmin
                    ? Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          IconButton(
                            tooltip: 'Edit ${t.name}',
                            icon: const Icon(Icons.edit_outlined),
                            onPressed: () => _edit(context, ref, t),
                          ),
                          Switch(
                            value: t.isActive,
                            onChanged: (v) async {
                              final repo = ref.read(taxesRepositoryProvider);
                              if (await runAction(
                                context,
                                () => v
                                    ? repo.update(t.id, {'is_active': true})
                                    : repo.deactivate(t.id),
                                success: v
                                    ? 'Tax reactivated'
                                    : 'Tax deactivated',
                              )) {
                                ref.invalidate(taxesProvider);
                              }
                            },
                          ),
                        ],
                      )
                    : null,
              ),
          ],
        ),
      ),
    );
  }

  Future<void> _edit(BuildContext context, WidgetRef ref, Tax? t) async {
    final name = TextEditingController(text: t?.name);
    final rate = TextEditingController(text: t?.rate);
    final repo = ref.read(taxesRepositoryProvider);
    final saved = await showApiForm<bool>(
      context,
      title: t == null ? 'New tax' : 'Edit ${t.name}',
      fields: (_, _, e) => [
        formField(
          name,
          'Name',
          errors: e,
          errorKey: 'name',
          hint: 'GST 18%',
          validator: requiredValidator,
        ),
        formField(
          rate,
          'Rate (%)',
          errors: e,
          errorKey: 'rate',
          hint: '18',
          keyboard: const TextInputType.numberWithOptions(decimal: true),
          validator: moneyValidator(),
        ),
      ],
      onSave: () async {
        t == null
            ? await repo.create(name.text.trim(), rate.text.trim())
            : await repo.update(t.id, {
                'name': name.text.trim(),
                'rate': rate.text.trim(),
              });
        return true;
      },
    );
    name.dispose();
    rate.dispose();
    if (saved == true && context.mounted) {
      ref.invalidate(taxesProvider);
      ref.invalidate(activeTaxesProvider);
      showMessage(context, t == null ? 'Tax added' : 'Tax updated');
    }
  }
}
