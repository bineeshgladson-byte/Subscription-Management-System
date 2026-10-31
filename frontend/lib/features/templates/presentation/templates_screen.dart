import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../auth/application/auth_controller.dart';
import '../../plans/data/plans_api.dart';
import '../../products/data/products_api.dart';
import '../../products/presentation/line_editor.dart';
import '../data/templates_api.dart';

/// Quotation templates: "start from a quotation template that already has the
/// right plan and product lines, and stays valid for a set number of days".
class TemplatesScreen extends ConsumerStatefulWidget {
  const TemplatesScreen({super.key});
  @override
  ConsumerState<TemplatesScreen> createState() => _TemplatesScreenState();
}

class _TemplatesScreenState extends ConsumerState<TemplatesScreen> {
  TemplateQuery _q = (q: '', active: true, page: 1);

  @override
  Widget build(BuildContext context) {
    final isAdmin = ref.watch(authControllerProvider).value?.role == Role.admin;
    return PageFrame(
      title: 'Quotation templates',
      subtitle:
          'Ready-made offers: a plan, product lines, validity and payment terms.',
      action: isAdmin
          ? FilledButton.icon(
              onPressed: () => _edit(null),
              icon: const Icon(Icons.add),
              label: const Text('New template'),
            )
          : null,
      toolbar: Toolbar(
        children: [
          SearchBox(
            hint: 'Search templates',
            onChanged: (v) =>
                setState(() => _q = (q: v, active: _q.active, page: 1)),
          ),
          FilterDropdown<bool>(
            label: 'Status',
            value: _q.active,
            options: const {true: 'Active', false: 'Inactive'},
            onChanged: (v) =>
                setState(() => _q = (q: _q.q, active: v, page: 1)),
          ),
        ],
      ),
      child: AsyncView(
        value: ref.watch(templatesProvider(_q)),
        onRetry: () => ref.invalidate(templatesProvider),
        builder: (page) => Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (page.items.isEmpty)
              const EmptyState(
                icon: Icons.description_outlined,
                title: 'No templates',
                message: 'Create one from a plan to speed up quotations.',
              ),
            for (final t in page.items)
              ResourceTile(
                title: t.name,
                subtitle: [
                  if (t.planName != null) '${t.planName} (${t.planLabel})',
                  'valid ${t.validityDays} days',
                  'payment in ${t.paymentTermsDays} days',
                  'estimated ${Fmt.inr(t.estimatedTotal)} before tax',
                ].join(' · '),
                pills: [
                  if (!t.isActive) const Pill('Inactive'),
                  for (final l in t.lines)
                    Pill('${l.quantity} × ${l.label}', tone: 'brand'),
                ],
                trailing: isAdmin
                    ? Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          IconButton(
                            tooltip: 'Edit ${t.name}',
                            icon: const Icon(Icons.edit_outlined),
                            onPressed: () => _edit(t),
                          ),
                          Switch(
                            value: t.isActive,
                            onChanged: (v) async {
                              final repo = ref.read(
                                templatesRepositoryProvider,
                              );
                              if (await runAction(
                                context,
                                () => v
                                    ? repo.update(t.id, {'is_active': true})
                                    : repo.deactivate(t.id),
                                success: v
                                    ? 'Template reactivated'
                                    : 'Template deactivated',
                              )) {
                                ref.invalidate(templatesProvider);
                              }
                            },
                          ),
                        ],
                      )
                    : null,
              ),
            Pager(
              page: page.page,
              pageSize: page.pageSize,
              total: page.total,
              onPage: (n) =>
                  setState(() => _q = (q: _q.q, active: _q.active, page: n)),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _edit(QuotationTemplate? t) async {
    final products = await ref
        .read(activeProductsProvider.future)
        .catchError((_) => <Product>[]);
    final plans = await ref
        .read(activePlansProvider.future)
        .catchError((_) => <Plan>[]);
    if (!mounted) return;
    final name = TextEditingController(text: t?.name);
    final validity = TextEditingController(text: '${t?.validityDays ?? 15}');
    final payment = TextEditingController(text: '${t?.paymentTermsDays ?? 15}');
    final terms = TextEditingController(text: t?.terms);
    int? planId = t?.planId;
    var useLines = t != null;
    final lines = t == null || t.lines.isEmpty
        ? [LineDraft()]
        : [
            for (final l in t.lines)
              LineDraft(
                productId: l.productId,
                variantId: l.variantId,
                quantity: l.quantity,
              ),
          ];
    final repo = ref.read(templatesRepositoryProvider);
    final saved = await showApiForm<bool>(
      context,
      title: t == null ? 'New quotation template' : 'Edit ${t.name}',
      width: 640,
      fields: (_, setState, e) => [
        formField(
          name,
          'Name',
          errors: e,
          errorKey: 'name',
          validator: requiredValidator,
        ),
        DropdownButtonFormField<int?>(
          initialValue: plans.any((p) => p.id == planId) ? planId : null,
          decoration: InputDecoration(
            labelText: 'Plan',
            errorText: fieldError(e, 'plan_id'),
          ),
          items: [
            const DropdownMenuItem<int?>(value: null, child: Text('No plan')),
            for (final p in plans)
              DropdownMenuItem<int?>(
                value: p.id,
                child: Text(
                  '${p.name} · ${Fmt.inr(p.price)} ${p.billingLabel}',
                ),
              ),
          ],
          onChanged: (v) => setState(() => planId = v),
        ),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            SizedBox(
              width: 180,
              child: formField(
                validity,
                'Valid for (days)',
                errors: e,
                errorKey: 'validity_days',
                keyboard: TextInputType.number,
                validator: (v) => intValidator(v),
              ),
            ),
            SizedBox(
              width: 200,
              child: formField(
                payment,
                'Payment terms (days)',
                errors: e,
                errorKey: 'payment_terms_days',
                keyboard: TextInputType.number,
                validator: (v) => intValidator(v, min: 0),
              ),
            ),
          ],
        ),
        formField(terms, 'Terms', errors: e, errorKey: 'terms', maxLines: 3),
        if (t == null)
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('Choose product lines myself'),
            subtitle: const Text(
              'Off: the plan\'s products are copied into the template.',
            ),
            value: useLines,
            onChanged: (v) => setState(() => useLines = v),
          ),
        if (useLines)
          LineEditor(
            products: products,
            lines: lines,
            onChanged: () => setState(() {}),
            errors: e,
            errorKey: 'lines',
          ),
        if (fieldError(e, 'lines') != null && !useLines)
          Text(
            fieldError(e, 'lines')!,
            style: const TextStyle(color: Color(0xFFB3261E), fontSize: 12),
          ),
      ],
      onSave: () async {
        final body = <String, dynamic>{
          'name': name.text.trim(),
          'validity_days': int.parse(validity.text.trim()),
          'payment_terms_days': int.parse(payment.text.trim()),
          'terms': terms.text.trim(),
          if (useLines) 'lines': [for (final l in lines) ?l.toJson()],
        };
        if (t == null) {
          if (planId != null) body['plan_id'] = planId;
          await repo.create(body);
        } else {
          body['plan_id'] = planId ?? 0;
          await repo.update(t.id, body);
        }
        return true;
      },
    );
    for (final c in [name, validity, payment, terms]) {
      c.dispose();
    }
    for (final l in lines) {
      l.dispose();
    }
    if (saved == true && mounted) {
      ref.invalidate(templatesProvider);
      showMessage(context, t == null ? 'Template created' : 'Template updated');
    }
  }
}
