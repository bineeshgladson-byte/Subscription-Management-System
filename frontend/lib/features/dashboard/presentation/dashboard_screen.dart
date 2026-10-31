import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/models/role.dart';
import '../../../core/network/api_client.dart';
import '../../../core/theme/workspace_theme.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../../core/widgets/workspace_widgets.dart';
import '../../auth/application/auth_controller.dart';
import '../../customers/data/customers_api.dart';
import '../../discounts/data/discounts_api.dart';
import '../../plans/data/plans_api.dart';
import '../../products/data/products_api.dart';
import '../../staff/data/staff_api.dart';

/// Everything on this page comes from live API calls. Revenue, MRR and
/// overdue invoices join it when the billing modules (A9–A11) are merged.
class DashboardData {
  DashboardData({
    required this.customers,
    required this.products,
    required this.plans,
    this.activeDiscounts,
    this.staff,
  });
  final Paged<Customer> customers; // newest 5 + total
  final Paged<Product> products;
  final List<Plan> plans;
  final int? activeDiscounts; // admin only
  final int? staff; // admin only
}

final dashboardProvider = FutureProvider.autoDispose<DashboardData>((
  ref,
) async {
  final role = ref.watch(authControllerProvider).value?.role;
  final isAdmin = role == Role.admin;
  final customers = ref.watch(customersRepositoryProvider).list((
    q: '',
    hasLogin: null,
    page: 1,
  ));
  final products = ref.watch(productsRepositoryProvider).list((
    q: '',
    status: 'active',
    type: null,
    page: 1,
  ));
  final plans = ref.watch(plansRepositoryProvider).list((
    q: '',
    status: 'active',
    period: null,
  ));
  final discounts = isAdmin
      ? ref.watch(discountsRepositoryProvider).list((
          q: '',
          state: 'active',
          page: 1,
        ))
      : null;
  final staff = isAdmin
      ? ref.watch(staffRepositoryProvider).list((q: '', active: true, page: 1))
      : null;
  return DashboardData(
    customers: await customers,
    products: await products,
    plans: (await plans).items,
    activeDiscounts: discounts == null ? null : (await discounts).total,
    staff: staff == null ? null : (await staff).total,
  );
});

/// Customers see the plans they can buy.
final customerDashboardProvider = FutureProvider.autoDispose<List<Plan>>(
  (ref) async => (await ref.watch(plansRepositoryProvider).list((
    q: '',
    status: 'active',
    period: null,
  ))).items,
);

class DashboardScreen extends ConsumerWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authControllerProvider).value;
    if (user == null) return const SizedBox.shrink();
    final first = user.fullName.split(' ').first;
    if (user.role == Role.customer) {
      return PageFrame(
        title: 'Overview',
        subtitle: 'Welcome back, $first.',
        child: AsyncView(
          value: ref.watch(customerDashboardProvider),
          onRetry: () => ref.invalidate(customerDashboardProvider),
          builder: (plans) => Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Wrap(
                spacing: 12,
                runSpacing: 12,
                children: [
                  SizedBox(
                    width: 220,
                    child: MetricTile(
                      label: 'Plans you can buy',
                      value: '${plans.length}',
                    ),
                  ),
                  SizedBox(
                    width: 220,
                    child: MetricTile(
                      label: 'Account',
                      value: user.email.split('@').first,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 20),
              Text(
                'Available plans',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 10),
              if (plans.isEmpty)
                const EmptyState(
                  icon: Icons.layers_outlined,
                  title: 'No plans right now',
                  message: 'Please check back soon.',
                ),
              for (final p in plans.take(5))
                ResourceTile(
                  title: p.name,
                  subtitle: '${Fmt.inr(p.price)} · ${p.billingLabel}',
                  onTap: () => context.go('/plans'),
                ),
              const SizedBox(height: 8),
              Text(
                'Your subscriptions and invoices appear here once those modules are live.',
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ],
          ),
        ),
      );
    }

    final isAdmin = user.role == Role.admin;
    return PageFrame(
      title: 'Overview',
      subtitle: 'Hello $first · live figures from Subly.',
      child: AsyncView(
        value: ref.watch(dashboardProvider),
        onRetry: () => ref.invalidate(dashboardProvider),
        builder: (d) {
          final subscribers = d.plans.fold<int>(
            0,
            (s, p) => s + (p.activeSubscriptions ?? 0),
          );
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Wrap(
                spacing: 12,
                runSpacing: 12,
                children: [
                  _tile(
                    context,
                    'Customers',
                    '${d.customers.total}',
                    '/customers',
                  ),
                  _tile(
                    context,
                    'Active products',
                    '${d.products.total}',
                    '/products',
                  ),
                  _tile(context, 'Active plans', '${d.plans.length}', '/plans'),
                  _tile(
                    context,
                    'Active subscribers',
                    '$subscribers',
                    '/plans',
                  ),
                  if (isAdmin && d.activeDiscounts != null)
                    _tile(
                      context,
                      'Active discounts',
                      '${d.activeDiscounts}',
                      '/discounts',
                    ),
                  if (isAdmin && d.staff != null)
                    _tile(context, 'Active staff', '${d.staff}', '/staff'),
                ],
              ),
              const SizedBox(height: 24),
              LayoutBuilder(
                builder: (context, c) {
                  final wide = c.maxWidth >= 900;
                  final plans = _section(context, 'Plans', [
                    if (d.plans.isEmpty)
                      Text(
                        'No active plans yet.',
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    for (final p in d.plans)
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        title: Text(p.name),
                        subtitle: Text(
                          '${p.billingLabel} · ${p.activeSubscriptions ?? 0} subscriber${p.activeSubscriptions == 1 ? '' : 's'}',
                        ),
                        trailing: Text(
                          Fmt.inr(p.price),
                          style: const TextStyle(fontWeight: FontWeight.w500),
                        ),
                      ),
                  ]);
                  final customers = _section(context, 'Newest customers', [
                    if (d.customers.items.isEmpty)
                      Text(
                        'No customers yet.',
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    for (final cu in d.customers.items.take(5))
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: CircleAvatar(
                          child: Text(
                            cu.name.isEmpty ? '?' : cu.name[0].toUpperCase(),
                          ),
                        ),
                        title: Text(cu.name),
                        subtitle: Text(cu.companyName ?? cu.email),
                        trailing: Pill(
                          cu.hasLogin ? 'Portal login' : 'No login',
                          tone: cu.hasLogin ? 'good' : 'neutral',
                        ),
                      ),
                  ]);
                  return wide
                      ? Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Expanded(child: plans),
                            const SizedBox(width: 16),
                            Expanded(child: customers),
                          ],
                        )
                      : Column(
                          children: [
                            plans,
                            const SizedBox(height: 16),
                            customers,
                          ],
                        );
                },
              ),
              const SizedBox(height: 14),
              const Row(
                children: [
                  Icon(
                    Icons.info_outline,
                    size: 13,
                    color: WorkspaceTheme.muted,
                  ),
                  SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      'Revenue, MRR and overdue invoices appear here once billing and payments are live.',
                      style: TextStyle(
                        fontSize: 11,
                        color: WorkspaceTheme.muted,
                      ),
                    ),
                  ),
                ],
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _tile(
    BuildContext context,
    String label,
    String value,
    String route,
  ) => SizedBox(
    width: 200,
    child: InkWell(
      borderRadius: BorderRadius.circular(17),
      onTap: () => context.go(route),
      child: MetricTile(label: label, value: value),
    ),
  );

  Widget _section(BuildContext context, String title, List<Widget> children) =>
      Card(
        child: Padding(
          padding: const EdgeInsets.all(18),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(title, style: Theme.of(context).textTheme.titleLarge),
              const SizedBox(height: 8),
              ...children,
            ],
          ),
        ),
      );
}
