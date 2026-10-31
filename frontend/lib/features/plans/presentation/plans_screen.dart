import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/theme/workspace_theme.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../../core/widgets/workspace_widgets.dart';
import '../../auth/application/auth_controller.dart';
import '../../products/data/products_api.dart';
import '../../products/presentation/line_editor.dart';
import '../data/plans_api.dart';

/// Subscription plans from GET /plans. Admin creates / edits / archives;
/// staff see subscriber counts; customers see the plans they can buy today.
class PlansScreen extends ConsumerStatefulWidget {
  const PlansScreen({super.key});
  @override
  ConsumerState<PlansScreen> createState() => _PlansScreenState();
}

class _PlansScreenState extends ConsumerState<PlansScreen> {
  PlanQuery _q = (q: '', status: 'active', period: null);

  @override
  Widget build(BuildContext context) {
    final role = ref.watch(authControllerProvider).value?.role ?? Role.customer;
    final isAdmin = role == Role.admin;
    final isCustomer = role == Role.customer;
    return PageFrame(
      title: isCustomer ? 'Available plans' : 'Plans',
      subtitle: isCustomer
          ? 'Plans you can subscribe to today. Prices exclude tax.'
          : 'How often each plan bills, what it includes and what customers may do with it.',
      action: isAdmin
          ? FilledButton.icon(
              onPressed: () => _edit(null),
              icon: const Icon(Icons.add),
              label: const Text('Create Plan'),
            )
          : null,
      toolbar: Toolbar(
        children: [
          SearchBox(
            hint: 'Search plans',
            onChanged: (v) => setState(
              () => _q = (q: v, status: _q.status, period: _q.period),
            ),
          ),
          FilterDropdown<String>(
            label: 'Billing',
            value: _q.period,
            options: billingPeriods,
            onChanged: (v) =>
                setState(() => _q = (q: _q.q, status: _q.status, period: v)),
          ),
          if (!isCustomer)
            FilterDropdown<String>(
              label: 'Status',
              value: _q.status == 'active' ? null : _q.status,
              allLabel: 'Active',
              options: const {'archived': 'Archived', 'all': 'All'},
              onChanged: (v) => setState(
                () => _q = (q: _q.q, status: v ?? 'active', period: _q.period),
              ),
            ),
        ],
      ),
      child: AsyncView(
        value: ref.watch(plansProvider(_q)),
        onRetry: () => ref.invalidate(plansProvider),
        builder: (page) {
          final plans = page.items;
          if (plans.isEmpty) {
            return EmptyState(
              icon: Icons.layers_outlined,
              title: isCustomer
                  ? 'No plans available right now'
                  : 'No plans yet',
              message: isCustomer
                  ? 'Please check back soon.'
                  : 'Create a plan from your products.',
            );
          }
          final subscribers = plans.fold<int>(
            0,
            (s, p) => s + (p.activeSubscriptions ?? 0),
          );
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (!isCustomer) ...[
                Wrap(
                  spacing: 12,
                  runSpacing: 12,
                  children: [
                    SizedBox(
                      width: 200,
                      child: MetricTile(
                        label: 'Plans shown',
                        value: '${plans.length}',
                      ),
                    ),
                    SizedBox(
                      width: 200,
                      child: MetricTile(
                        label: 'Available now',
                        value: '${plans.where((p) => p.isAvailableNow).length}',
                      ),
                    ),
                    SizedBox(
                      width: 200,
                      child: MetricTile(
                        label: 'Active subscribers',
                        value: '$subscribers',
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 18),
              ],
              LayoutBuilder(
                builder: (context, c) {
                  final cols = c.maxWidth >= 1100
                      ? 3
                      : c.maxWidth >= 700
                      ? 2
                      : 1;
                  final w = (c.maxWidth - (cols - 1) * 16) / cols;
                  return Wrap(
                    spacing: 16,
                    runSpacing: 16,
                    children: [
                      for (final p in plans)
                        SizedBox(
                          width: w,
                          child: _PlanCard(
                            plan: p,
                            isAdmin: isAdmin,
                            isCustomer: isCustomer,
                            onEdit: () => _edit(p),
                            onToggle: () => _toggle(p),
                          ),
                        ),
                    ],
                  );
                },
              ),
            ],
          );
        },
      ),
    );
  }

  Future<void> _toggle(Plan p) async {
    final archive = p.isActive;
    if (archive &&
        !await confirmAction(
          context,
          title: 'Archive ${p.name}?',
          message:
              'Customers stop seeing it. People already subscribed keep being billed.',
          confirmLabel: 'Archive',
          danger: true,
        )) {
      return;
    }
    if (!mounted) return;
    if (await runAction(
      context,
      () => ref
          .read(plansRepositoryProvider)
          .setStatus(p.id, archive ? 'archived' : 'active'),
      success: archive ? 'Plan archived' : 'Plan reactivated',
    )) {
      ref.invalidate(plansProvider);
      ref.invalidate(activePlansProvider);
    }
  }

  Future<void> _edit(Plan? p) async {
    final products = await ref
        .read(activeProductsProvider.future)
        .catchError((_) => <Product>[]);
    if (!mounted) return;
    final name = TextEditingController(text: p?.name);
    final desc = TextEditingController(text: p?.description);
    final interval = TextEditingController(text: '${p?.billingInterval ?? 1}');
    final minQty = TextEditingController(text: '${p?.minQuantity ?? 1}');
    final from = TextEditingController(
      text: p?.availableFrom ?? Fmt.apiDate(DateTime.now()),
    );
    final until = TextEditingController(text: p?.availableUntil ?? '');
    final cycles = TextEditingController(
      text: p?.autoCloseAfterCycles?.toString() ?? '',
    );
    var period = p?.billingPeriod ?? 'monthly';
    var pausable = p?.isPausable ?? false,
        closable = p?.isClosable ?? true,
        renewable = p?.isRenewable ?? true;
    var autoClose = p?.autoCloseEnabled ?? false;
    final lines = p == null || p.lines.isEmpty
        ? [LineDraft()]
        : [
            for (final l in p.lines)
              LineDraft(
                productId: l.productId,
                variantId: l.variantId,
                quantity: l.quantity,
                priceOverride: l.priceOverride,
              ),
          ];
    final repo = ref.read(plansRepositoryProvider);
    final saved = await showApiForm<bool>(
      context,
      title: p == null ? 'Create plan' : 'Edit ${p.name}',
      width: 680,
      fields: (_, setState, e) => [
        formField(
          name,
          'Plan name',
          errors: e,
          errorKey: 'name',
          validator: requiredValidator,
        ),
        formField(
          desc,
          'Description',
          errors: e,
          errorKey: 'description',
          maxLines: 2,
        ),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            SizedBox(
              width: 180,
              child: DropdownButtonFormField<String>(
                initialValue: period,
                decoration: InputDecoration(
                  labelText: 'Bills',
                  errorText:
                      fieldError(e, 'billing_period') ?? fieldError(e, 'rule'),
                ),
                items: [
                  for (final b in billingPeriods.entries)
                    DropdownMenuItem(value: b.key, child: Text(b.value)),
                ],
                onChanged: (v) => period = v ?? period,
              ),
            ),
            SizedBox(
              width: 130,
              child: formField(
                interval,
                'Every N periods',
                errors: e,
                errorKey: 'billing_interval',
                keyboard: TextInputType.number,
                validator: (v) => intValidator(v),
              ),
            ),
            SizedBox(
              width: 130,
              child: formField(
                minQty,
                'Minimum qty',
                errors: e,
                errorKey: 'min_quantity',
                keyboard: TextInputType.number,
                validator: (v) => intValidator(v),
              ),
            ),
          ],
        ),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            SizedBox(
              width: 220,
              child: DateField(
                controller: from,
                label: 'Available from',
                required: true,
                errorText: fieldError(e, 'available_from'),
              ),
            ),
            SizedBox(
              width: 220,
              child: DateField(
                controller: until,
                label: 'Available until (optional)',
                errorText: fieldError(e, 'available_until'),
              ),
            ),
          ],
        ),
        Wrap(
          spacing: 8,
          runSpacing: 4,
          children: [
            FilterChip(
              label: const Text('Can be paused'),
              selected: pausable,
              onSelected: (v) => setState(() => pausable = v),
            ),
            FilterChip(
              label: const Text('Can be closed early'),
              selected: closable,
              onSelected: (v) => setState(() => closable = v),
            ),
            FilterChip(
              label: const Text('Can be renewed'),
              selected: renewable,
              onSelected: (v) => setState(() => renewable = v),
            ),
            FilterChip(
              label: const Text('Closes on its own'),
              selected: autoClose,
              onSelected: (v) => setState(() => autoClose = v),
            ),
          ],
        ),
        if (autoClose)
          SizedBox(
            width: 220,
            child: formField(
              cycles,
              'Close after N billing cycles',
              errors: e,
              errorKey: 'auto_close_after_cycles',
              keyboard: TextInputType.number,
              validator: (v) => intValidator(v),
            ),
          ),
        LineEditor(
          products: products,
          lines: lines,
          onChanged: () => setState(() {}),
          allowOverride: true,
          errors: e,
        ),
        Text(
          'The plan price is worked out from these products.',
          style: Theme.of(context).textTheme.bodySmall,
        ),
      ],
      onSave: () async {
        final body = {
          'name': name.text.trim(),
          'description': desc.text.trim(),
          'billing_period': period,
          'billing_interval': int.parse(interval.text.trim()),
          'min_quantity': int.parse(minQty.text.trim()),
          'available_from': from.text.trim(),
          'available_until': until.text.trim(),
          'is_pausable': pausable,
          'is_closable': closable,
          'is_renewable': renewable,
          'auto_close_enabled': autoClose,
          'auto_close_after_cycles': autoClose
              ? int.tryParse(cycles.text.trim()) ?? 0
              : 0,
          'products': [for (final l in lines) ?l.toJson(withOverride: true)],
        };
        if (p == null) {
          if (until.text.trim().isEmpty) body.remove('available_until');
          if (!autoClose) body.remove('auto_close_after_cycles');
          await repo.create(body);
        } else {
          if (period == p.billingPeriod) body.remove('billing_period');
          if (body['billing_interval'] == p.billingInterval) {
            body.remove('billing_interval');
          }
          await repo.update(p.id, body);
        }
        return true;
      },
    );
    for (final t in [name, desc, interval, minQty, from, until, cycles]) {
      t.dispose();
    }
    for (final l in lines) {
      l.dispose();
    }
    if (saved == true && mounted) {
      ref.invalidate(plansProvider);
      ref.invalidate(activePlansProvider);
      showMessage(context, p == null ? 'Plan created' : 'Plan updated');
    }
  }
}

class _PlanCard extends StatelessWidget {
  const _PlanCard({
    required this.plan,
    required this.isAdmin,
    required this.isCustomer,
    required this.onEdit,
    required this.onToggle,
  });
  final Plan plan;
  final bool isAdmin, isCustomer;
  final VoidCallback onEdit, onToggle;

  @override
  Widget build(BuildContext context) {
    final p = plan;
    final period = switch (p.billingPeriod) {
      'daily' => 'day',
      'weekly' => 'week',
      'yearly' => 'year',
      _ => 'month',
    };
    final per = p.billingInterval > 1
        ? '${p.billingInterval} ${period}s'
        : period;
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(
          color: p.isActive ? WorkspaceTheme.border : const Color(0xFFF3C98B),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  p.name,
                  style: Theme.of(context).textTheme.titleLarge,
                ),
              ),
              if (isAdmin)
                PopupMenuButton<String>(
                  tooltip: 'Manage ${p.name}',
                  onSelected: (v) => v == 'edit' ? onEdit() : onToggle(),
                  itemBuilder: (_) => [
                    const PopupMenuItem(
                      value: 'edit',
                      child: Text('Edit plan'),
                    ),
                    PopupMenuItem(
                      value: 'toggle',
                      child: Text(
                        p.isActive ? 'Archive plan' : 'Reactivate plan',
                      ),
                    ),
                  ],
                ),
            ],
          ),
          if (p.description != null)
            Text(p.description!, style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 12),
          Text.rich(
            TextSpan(
              children: [
                TextSpan(
                  text: Fmt.inr(p.price),
                  style: const TextStyle(
                    fontSize: 28,
                    fontWeight: FontWeight.w600,
                    letterSpacing: -0.8,
                  ),
                ),
                TextSpan(
                  text: '  / $per',
                  style: const TextStyle(
                    color: WorkspaceTheme.muted,
                    fontSize: 12,
                  ),
                ),
              ],
            ),
          ),
          Text(
            '${p.billingLabel} · minimum ${p.minQuantity}',
            style: Theme.of(context).textTheme.bodySmall,
          ),
          const SizedBox(height: 12),
          for (final l in p.lines)
            Padding(
              padding: const EdgeInsets.only(bottom: 4),
              child: Row(
                children: [
                  const Icon(
                    Icons.check_circle,
                    size: 16,
                    color: WorkspaceTheme.green,
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      '${l.quantity} × ${l.label}',
                      style: const TextStyle(fontSize: 13),
                    ),
                  ),
                  Text(
                    Fmt.inr(l.lineTotal),
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                ],
              ),
            ),
          const SizedBox(height: 10),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: [
              if (!p.isActive) const Pill('Archived', tone: 'warn'),
              if (p.isActive && !p.isAvailableNow)
                const Pill('Not available today', tone: 'warn'),
              if (p.isPausable) const Pill('Pausable', tone: 'brand'),
              if (p.isClosable) const Pill('Close early', tone: 'brand'),
              if (p.isRenewable) const Pill('Renewable', tone: 'brand'),
              if (p.autoCloseEnabled)
                Pill(
                  'Auto-closes after ${p.autoCloseAfterCycles} cycles',
                  tone: 'brand',
                ),
            ],
          ),
          const SizedBox(height: 10),
          Text(
            'Available from ${Fmt.date(p.availableFrom)}${p.availableUntil == null ? '' : ' until ${Fmt.date(p.availableUntil!)}'}',
            style: Theme.of(context).textTheme.bodySmall,
          ),
          if (!isCustomer && p.activeSubscriptions != null)
            Text(
              'Subscribers: ${p.activeSubscriptions}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
        ],
      ),
    );
  }
}
