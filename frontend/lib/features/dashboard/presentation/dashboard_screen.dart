import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/models/role.dart';
import '../../../core/theme/workspace_theme.dart';
import '../../../core/widgets/workspace_widgets.dart';
import '../../auth/application/auth_controller.dart';

class DashboardScreen extends ConsumerWidget {
  const DashboardScreen({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authControllerProvider).value;
    final admin = user?.role == Role.admin;
    final customer = user?.role == Role.customer;
    return LayoutBuilder(
      builder: (context, constraints) => ListView(
        padding: EdgeInsets.fromLTRB(
          constraints.maxWidth < 600 ? 20 : 24,
          20,
          constraints.maxWidth < 600 ? 20 : 24,
          32,
        ),
        children: [
          WorkspaceHeading(
            title: 'Overview',
            subtitle: 'A little clarity for your everyday business.',
            action: OutlinedButton.icon(
              onPressed: () => context.go('/plans'),
              icon: const Icon(Icons.add, size: 18),
              label: const Text('Explore Plans'),
            ),
          ),
          const SizedBox(height: 30),
          LayoutBuilder(
            builder: (context, c) {
              final metrics = [
                MetricTile(
                  label: customer
                      ? 'Your Subscriptions'
                      : 'Active Subscriptions',
                  value: customer ? '3' : '324',
                ),
                MetricTile(
                  label: customer ? 'Upcoming Invoices' : 'Total Customers',
                  value: customer ? '1' : '286',
                ),
                if (admin)
                  const MetricTile(
                    label: 'Monthly Recurring Revenue',
                    value: r'$48,546',
                  ),
                if (!admin) const MetricTile(label: 'Active Plans', value: '4'),
              ];
              return Wrap(
                spacing: 12,
                runSpacing: 12,
                children: [
                  for (final metric in metrics)
                    SizedBox(
                      width: c.maxWidth < 600
                          ? c.maxWidth
                          : (c.maxWidth - 24) / 3,
                      child: metric,
                    ),
                ],
              );
            },
          ),
          const SizedBox(height: 32),
          LayoutBuilder(
            builder: (context, c) {
              final chart = _ActivityChart(financial: admin);
              final plans = Container(
                padding: const EdgeInsets.all(22),
                decoration: BoxDecoration(
                  border: Border.all(color: WorkspaceTheme.border),
                  borderRadius: BorderRadius.circular(18),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      'A plan for every possibility',
                      style: TextStyle(
                        fontSize: 17,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 10),
                    const Text(
                      'Simple beginnings. Room to grow.\nExplore the plans that power your business.',
                      style: TextStyle(
                        fontSize: 12,
                        height: 1.8,
                        color: WorkspaceTheme.muted,
                      ),
                    ),
                    const SizedBox(height: 26),
                    for (final (name, count, progress) in [
                      ('Starter', '145 subscribers', .87),
                      ('Professional', '89 subscribers', .62),
                      ('Business', '52 subscribers', .42),
                      ('Enterprise', '38 subscribers', .28),
                    ]) ...[
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              name,
                              style: const TextStyle(fontSize: 12),
                            ),
                          ),
                          if (!customer)
                            Text(
                              count,
                              style: const TextStyle(
                                fontSize: 10,
                                color: WorkspaceTheme.muted,
                              ),
                            ),
                        ],
                      ),
                      const SizedBox(height: 8),
                      LinearProgressIndicator(
                        value: progress,
                        minHeight: 5,
                        color: WorkspaceTheme.purple,
                        backgroundColor: WorkspaceTheme.lavender,
                        borderRadius: BorderRadius.circular(4),
                      ),
                      const SizedBox(height: 18),
                    ],
                    TextButton(
                      onPressed: () => context.go('/plans'),
                      child: const Text('View all plans  →'),
                    ),
                  ],
                ),
              );
              if (c.maxWidth < 800) {
                return Column(
                  children: [chart, const SizedBox(height: 16), plans],
                );
              }
              return Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(flex: 7, child: chart),
                  const SizedBox(width: 16),
                  Expanded(flex: 4, child: plans),
                ],
              );
            },
          ),
          const SizedBox(height: 28),
          Container(
            padding: const EdgeInsets.all(22),
            decoration: BoxDecoration(
              border: Border.all(color: WorkspaceTheme.border),
              borderRadius: BorderRadius.circular(18),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    const Expanded(
                      child: Text(
                        'Recent activity',
                        style: TextStyle(
                          fontSize: 17,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ),
                    TextButton(
                      onPressed: () => context.go('/subscriptions'),
                      child: const Text('View all →'),
                    ),
                  ],
                ),
                const Divider(),
                for (final (icon, title, detail, time) in [
                  (
                    Icons.check_circle_outline,
                    'Subscription renewed',
                    'Professional · Monthly billing',
                    '2 hours ago',
                  ),
                  (
                    Icons.receipt_long_outlined,
                    'Invoice created',
                    'Business · New billing period',
                    '4 hours ago',
                  ),
                  (
                    Icons.person_add_alt_outlined,
                    'Welcome to Subly',
                    'Everything you need, in one workspace',
                    'Today',
                  ),
                ])
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 10),
                    child: Row(
                      children: [
                        CircleAvatar(
                          radius: 20,
                          backgroundColor: WorkspaceTheme.lavender,
                          child: Icon(
                            icon,
                            size: 18,
                            color: WorkspaceTheme.purple,
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                title,
                                style: const TextStyle(
                                  fontSize: 12,
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                              Text(
                                detail,
                                style: const TextStyle(
                                  fontSize: 10,
                                  color: WorkspaceTheme.muted,
                                ),
                              ),
                            ],
                          ),
                        ),
                        if (constraints.maxWidth >= 600)
                          Text(
                            time,
                            style: const TextStyle(
                              fontSize: 10,
                              color: WorkspaceTheme.muted,
                            ),
                          ),
                      ],
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(height: 22),
          const PreviewNotice(),
        ],
      ),
    );
  }
}

class _ActivityChart extends StatelessWidget {
  const _ActivityChart({required this.financial});
  final bool financial;
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(22),
    decoration: BoxDecoration(
      border: Border.all(color: WorkspaceTheme.border),
      borderRadius: BorderRadius.circular(18),
    ),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          financial ? 'Revenue overview' : 'Subscription overview',
          style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w500),
        ),
        const SizedBox(height: 4),
        const Text(
          'A steady rhythm. A brighter outlook.',
          style: TextStyle(fontSize: 11, color: WorkspaceTheme.muted),
        ),
        const SizedBox(height: 20),
        Text(
          financial ? r'$48,546' : '324 active',
          style: const TextStyle(
            fontSize: 30,
            fontWeight: FontWeight.w500,
            letterSpacing: -1,
          ),
        ),
        const Text(
          '+12.8% from last month',
          style: TextStyle(fontSize: 11, color: WorkspaceTheme.green),
        ),
        const SizedBox(height: 28),
        SizedBox(
          height: 174,
          child: Semantics(
            label: 'Sample monthly growth chart, January to June',
            child: LayoutBuilder(
              builder: (context, c) => Row(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  for (final (label, height) in [
                    ('Jan', .38),
                    ('Feb', .51),
                    ('Mar', .45),
                    ('Apr', .7),
                    ('May', .64),
                    ('Jun', .93),
                  ])
                    Expanded(
                      child: Padding(
                        padding: EdgeInsets.symmetric(
                          horizontal: c.maxWidth < 300 ? 6 : 14,
                        ),
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.end,
                          children: [
                            Container(
                              height: height * 144,
                              decoration: BoxDecoration(
                                borderRadius: const BorderRadius.vertical(
                                  top: Radius.circular(7),
                                ),
                                gradient: LinearGradient(
                                  begin: Alignment.bottomCenter,
                                  end: Alignment.topCenter,
                                  colors: [
                                    WorkspaceTheme.purple.withValues(
                                      alpha: .35,
                                    ),
                                    WorkspaceTheme.purple.withValues(
                                      alpha: label == 'Jun' ? 1 : .65,
                                    ),
                                  ],
                                ),
                              ),
                            ),
                            const SizedBox(height: 12),
                            Text(
                              label,
                              style: const TextStyle(
                                fontSize: 10,
                                color: WorkspaceTheme.muted,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                ],
              ),
            ),
          ),
        ),
      ],
    ),
  );
}
