import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../core/models/role.dart';
import '../../../core/theme/workspace_theme.dart';
import '../../../core/widgets/workspace_widgets.dart';
import '../../auth/application/auth_controller.dart';

final _currency = NumberFormat.currency(symbol: r'$', decimalDigits: 0);

class PlanPreview {
  const PlanPreview(
    this.name,
    this.description,
    this.price,
    this.subscribers,
    this.mrr,
    this.features, {
    this.available = 5,
    this.popular = false,
    this.icon = Icons.layers_outlined,
  });
  final String name, description;
  final int price, subscribers, mrr, available;
  final List<String> features;
  final bool popular;
  final IconData icon;
}

const samplePlans = [
  PlanPreview(
    'Starter',
    'Perfect for small teams getting well started',
    29,
    145,
    4205,
    [
      'Up to 5 team members',
      'Basic analytics',
      'Email support',
      'API access',
      'Custom integrations',
    ],
    available: 3,
    icon: Icons.auto_awesome_outlined,
  ),
  PlanPreview(
    'Professional',
    'For growing businesses with advanced needs',
    79,
    89,
    7031,
    [
      'Up to 20 team members',
      'Advanced analytics',
      'Priority support',
      'API access',
      'Custom integrations',
    ],
    available: 4,
    popular: true,
    icon: Icons.bolt_outlined,
  ),
  PlanPreview(
    'Business',
    'Scale your operations with powerful tools',
    199,
    52,
    10348,
    [
      'Up to 50 team members',
      'Full analytics suite',
      '24/7 phone support',
      'API access',
      'Custom integrations',
    ],
    icon: Icons.workspace_premium_outlined,
  ),
  PlanPreview(
    'Enterprise',
    'Custom solutions for large organizations',
    499,
    38,
    26962,
    [
      'Unlimited team members',
      'Custom analytics',
      'Dedicated support',
      'Full API access',
      'White-label options',
    ],
    icon: Icons.business_outlined,
  ),
];

class PlansScreen extends ConsumerStatefulWidget {
  const PlansScreen({super.key});
  @override
  ConsumerState<PlansScreen> createState() => _PlansScreenState();
}

class _PlansScreenState extends ConsumerState<PlansScreen> {
  final plans = [...samplePlans];

  Future<void> _edit({PlanPreview? plan}) async {
    final result = await showDialog<PlanPreview>(
      context: context,
      builder: (context) => _PlanDialog(plan: plan),
    );
    if (result == null || !mounted) return;
    setState(() {
      if (plan == null) {
        plans.add(result);
      } else {
        plans[plans.indexOf(plan)] = result;
      }
    });
  }

  Future<void> _archive(PlanPreview plan) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('Archive ${plan.name}?'),
        content: const Text(
          'This removes the plan from this design preview. No live subscriptions will change.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Archive'),
          ),
        ],
      ),
    );
    if (confirmed == true && mounted) setState(() => plans.remove(plan));
  }

  @override
  Widget build(BuildContext context) {
    final role = ref.watch(authControllerProvider).value?.role ?? Role.customer;
    final admin = role == Role.admin;
    final financials = role == Role.admin;
    final mrr = plans.fold<int>(0, (sum, plan) => sum + plan.mrr);
    return LayoutBuilder(
      builder: (context, constraints) {
        final narrow = constraints.maxWidth < 600;
        return ListView(
          padding: EdgeInsets.fromLTRB(
            narrow ? 20 : 24,
            20,
            narrow ? 20 : 24,
            32,
          ),
          children: [
            WorkspaceHeading(
              title: 'Pricing & Plans',
              subtitle: 'Manage your subscription plans and pricing tiers',
              action: admin
                  ? DecoratedBox(
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.circular(16),
                        boxShadow: const [
                          BoxShadow(
                            color: Color(0x22000000),
                            blurRadius: 4,
                            offset: Offset(0, 3),
                          ),
                        ],
                      ),
                      child: OutlinedButton.icon(
                        style: OutlinedButton.styleFrom(
                          backgroundColor: Colors.white,
                        ),
                        onPressed: () => _edit(),
                        icon: const Icon(Icons.add, size: 19),
                        label: const Text('Create Plan'),
                      ),
                    )
                  : null,
            ),
            const SizedBox(height: 30),
            LayoutBuilder(
              builder: (context, c) {
                final tiles = [
                  MetricTile(label: 'Total Plans', value: '${plans.length}'),
                  if (role != Role.customer)
                    MetricTile(
                      label: 'Active Subscribers',
                      value:
                          '${plans.fold<int>(0, (sum, p) => sum + p.subscribers)}',
                    ),
                  if (financials)
                    MetricTile(
                      label: 'Total MRR from Plans',
                      value: _currency.format(mrr),
                    ),
                ];
                return Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    for (final tile in tiles)
                      SizedBox(
                        width: c.maxWidth < 520
                            ? c.maxWidth
                            : (c.maxWidth - 8 * (tiles.length - 1)) /
                                  tiles.length,
                        child: tile,
                      ),
                  ],
                );
              },
            ),
            const SizedBox(height: 42),
            LayoutBuilder(
              builder: (context, c) {
                final columns = c.maxWidth >= 1020
                    ? 4
                    : c.maxWidth >= 540
                    ? 2
                    : 1;
                return Wrap(
                  spacing: 12,
                  runSpacing: 24,
                  children: [
                    for (final plan in plans)
                      SizedBox(
                        width: (c.maxWidth - 12 * (columns - 1)) / columns,
                        child: _PlanCard(
                          plan: plan,
                          totalMrr: mrr,
                          showSubscribers: role != Role.customer,
                          showFinancials: financials,
                          onEdit: admin ? () => _edit(plan: plan) : null,
                          onArchive: admin ? () => _archive(plan) : null,
                        ),
                      ),
                  ],
                );
              },
            ),
            const SizedBox(height: 22),
            const PreviewNotice(),
          ],
        );
      },
    );
  }
}

class _PlanCard extends StatelessWidget {
  const _PlanCard({
    required this.plan,
    required this.totalMrr,
    required this.showSubscribers,
    required this.showFinancials,
    this.onEdit,
    this.onArchive,
  });
  final PlanPreview plan;
  final int totalMrr;
  final bool showSubscribers, showFinancials;
  final VoidCallback? onEdit, onArchive;

  @override
  Widget build(BuildContext context) {
    final share = totalMrr == 0 ? 0.0 : plan.mrr / totalMrr;
    return Stack(
      clipBehavior: Clip.none,
      children: [
        Container(
          padding: EdgeInsets.all(plan.popular ? 1.5 : 1),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(18),
            gradient: plan.popular
                ? const LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomRight,
                    colors: [
                      WorkspaceTheme.purple,
                      WorkspaceTheme.purple,
                      Color(0xFFEC971E),
                    ],
                  )
                : null,
            color: plan.popular ? null : WorkspaceTheme.border,
          ),
          child: Container(
            padding: const EdgeInsets.all(15),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(16.5),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Icon(plan.icon, size: 30),
                    if (onEdit != null)
                      SizedBox(
                        height: 44,
                        width: 44,
                        child: PopupMenuButton<String>(
                          tooltip: 'Manage ${plan.name}',
                          padding: EdgeInsets.zero,
                          icon: const Icon(
                            Icons.more_vert,
                            size: 17,
                            color: WorkspaceTheme.muted,
                          ),
                          onSelected: (value) =>
                              value == 'edit' ? onEdit!() : onArchive!(),
                          itemBuilder: (_) => const [
                            PopupMenuItem(
                              value: 'edit',
                              child: Text('Edit plan'),
                            ),
                            PopupMenuItem(
                              value: 'archive',
                              child: Text('Archive plan'),
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
                const SizedBox(height: 6),
                Text(
                  plan.name,
                  style: const TextStyle(
                    fontSize: 17,
                    fontWeight: FontWeight.w500,
                    letterSpacing: -0.5,
                  ),
                ),
                const SizedBox(height: 8),
                ConstrainedBox(
                  constraints: const BoxConstraints(minHeight: 46),
                  child: Text(
                    plan.description,
                    style: const TextStyle(
                      fontSize: 11.5,
                      color: WorkspaceTheme.muted,
                      height: 1.7,
                    ),
                  ),
                ),
                const Divider(),
                Text.rich(
                  TextSpan(
                    children: [
                      TextSpan(
                        text: _currency.format(plan.price),
                        style: const TextStyle(
                          fontSize: 34,
                          fontWeight: FontWeight.w500,
                          letterSpacing: -1.5,
                        ),
                      ),
                      const TextSpan(
                        text: '/mo',
                        style: TextStyle(
                          fontSize: 13,
                          color: WorkspaceTheme.muted,
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 12),
                for (final (index, feature) in plan.features.indexed)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 7),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Container(
                          width: 19,
                          height: 19,
                          decoration: const BoxDecoration(
                            color: Color(0xFFEFF5F0),
                            shape: BoxShape.circle,
                          ),
                          child: Icon(
                            Icons.check,
                            size: 13,
                            color: index < plan.available
                                ? WorkspaceTheme.green
                                : const Color(0xFFA4ADA6),
                          ),
                        ),
                        const SizedBox(width: 10),
                        Expanded(
                          child: Text(
                            feature,
                            style: TextStyle(
                              fontSize: 11.5,
                              color: index < plan.available
                                  ? WorkspaceTheme.ink
                                  : const Color(0xFF99999E),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                if (showSubscribers) ...[
                  const Divider(),
                  Container(
                    padding: const EdgeInsets.all(9),
                    decoration: BoxDecoration(
                      color: WorkspaceTheme.soft,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Column(
                      children: [
                        Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Expanded(
                              child: _smallMetric(
                                Icons.people_outline,
                                'Subscribers',
                                '${plan.subscribers}',
                              ),
                            ),
                            if (showFinancials) ...[
                              Container(
                                height: 50,
                                width: 1,
                                color: WorkspaceTheme.border,
                                margin: const EdgeInsets.symmetric(
                                  horizontal: 10,
                                ),
                              ),
                              Expanded(
                                child: _smallMetric(
                                  Icons.trending_up,
                                  'MRR Contribution',
                                  _currency.format(plan.mrr),
                                  green: true,
                                ),
                              ),
                            ],
                          ],
                        ),
                        if (showFinancials) ...[
                          const SizedBox(height: 18),
                          Row(
                            mainAxisAlignment: MainAxisAlignment.spaceBetween,
                            children: [
                              const Text(
                                'Revenue share',
                                style: TextStyle(fontSize: 9),
                              ),
                              Text(
                                '${(share * 100).toStringAsFixed(1)}%',
                                style: const TextStyle(fontSize: 9),
                              ),
                            ],
                          ),
                          const SizedBox(height: 8),
                          Semantics(
                            label:
                                'Revenue share ${(share * 100).toStringAsFixed(1)} percent',
                            child: Container(
                              height: 8,
                              color: const Color(0xFFEAEAEA),
                              alignment: Alignment.centerLeft,
                              child: FractionallySizedBox(
                                widthFactor: share.clamp(0, 1),
                                heightFactor: 1,
                                child: const DecoratedBox(
                                  decoration: BoxDecoration(
                                    gradient: LinearGradient(
                                      colors: [
                                        Color(0xFFE6DFFF),
                                        WorkspaceTheme.purple,
                                      ],
                                    ),
                                  ),
                                ),
                              ),
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
        if (plan.popular)
          Positioned(
            top: -11,
            left: 0,
            right: 0,
            child: Center(
              child: Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 4,
                ),
                decoration: BoxDecoration(
                  color: Colors.black,
                  borderRadius: BorderRadius.circular(20),
                ),
                child: const Text(
                  'Most Popular',
                  style: TextStyle(color: Colors.white, fontSize: 10),
                ),
              ),
            ),
          ),
      ],
    );
  }

  Widget _smallMetric(
    IconData icon,
    String label,
    String value, {
    bool green = false,
  }) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Row(
        children: [
          Icon(
            icon,
            size: 14,
            color: green ? WorkspaceTheme.green : WorkspaceTheme.muted,
          ),
          const SizedBox(width: 5),
          Expanded(
            child: Text(
              label,
              style: const TextStyle(fontSize: 8),
              maxLines: 2,
            ),
          ),
        ],
      ),
      const SizedBox(height: 10),
      Text(value, style: const TextStyle(fontSize: 13)),
    ],
  );
}

class _PlanDialog extends StatefulWidget {
  const _PlanDialog({this.plan});
  final PlanPreview? plan;
  @override
  State<_PlanDialog> createState() => _PlanDialogState();
}

class _PlanDialogState extends State<_PlanDialog> {
  final form = GlobalKey<FormState>();
  late final name = TextEditingController(text: widget.plan?.name);
  late final price = TextEditingController(text: widget.plan?.price.toString());
  late final description = TextEditingController(
    text: widget.plan?.description,
  );
  @override
  void dispose() {
    name.dispose();
    price.dispose();
    description.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
    title: Text(widget.plan == null ? 'Create Plan' : 'Edit Plan'),
    content: SizedBox(
      width: 400,
      child: SingleChildScrollView(
        child: Form(
          key: form,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Text(
                'Design preview only. This plan is not saved to the server.',
                style: TextStyle(fontSize: 12, color: WorkspaceTheme.muted),
              ),
              const SizedBox(height: 20),
              TextFormField(
                controller: name,
                decoration: const InputDecoration(labelText: 'Plan name'),
                validator: (value) => value == null || value.trim().isEmpty
                    ? 'Enter a plan name'
                    : null,
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: price,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(
                  labelText: 'Monthly price (USD)',
                  prefixText: r'$ ',
                ),
                validator: (value) =>
                    int.tryParse(value ?? '') == null || int.parse(value!) < 0
                    ? 'Enter a whole number, 0 or above'
                    : null,
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: description,
                maxLines: 2,
                maxLength: 90,
                decoration: const InputDecoration(labelText: 'Description'),
              ),
            ],
          ),
        ),
      ),
    ),
    actions: [
      TextButton(
        onPressed: () => Navigator.pop(context),
        child: const Text('Cancel'),
      ),
      FilledButton(
        onPressed: () {
          if (!form.currentState!.validate()) return;
          final old = widget.plan;
          Navigator.pop(
            context,
            PlanPreview(
              name.text.trim(),
              description.text.trim(),
              int.parse(price.text),
              old?.subscribers ?? 0,
              old?.mrr ?? 0,
              old?.features ??
                  const [
                    'Flexible billing',
                    'Email support',
                    'Customer portal',
                  ],
              available: old?.available ?? 3,
              popular: old?.popular ?? false,
              icon: old?.icon ?? Icons.layers_outlined,
            ),
          );
        },
        child: const Text('Save preview'),
      ),
    ],
  );
}
