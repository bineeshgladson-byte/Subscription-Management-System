import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/workspace_theme.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../auth/data/auth_user.dart';
import '../../plans/data/plans_api.dart';

/// Customer overview using the same catalogue the customer can browse.
class CustomerDashboard extends StatelessWidget {
  const CustomerDashboard({super.key, required this.user, required this.plans});
  final AuthUser user;
  final List<Plan> plans;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final wide = constraints.maxWidth >= 1000;
      final first = user.fullName.trim().split(' ').first;
      final welcome = Container(
        constraints: const BoxConstraints(minHeight: 274),
        padding: const EdgeInsets.all(28),
        decoration: BoxDecoration(
          gradient: const LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [WorkspaceTheme.lavender, Color(0xFFF7F4FF)],
          ),
          borderRadius: BorderRadius.circular(24),
        ),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const _Eyebrow('YOUR PERSONAL WORKSPACE'),
                  const SizedBox(height: 16),
                  Text(
                    'Welcome back${first.isEmpty ? '' : ', $first'}.',
                    style: const TextStyle(
                      fontSize: 30,
                      height: 1.25,
                      letterSpacing: -1,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  const SizedBox(height: 12),
                  const Text(
                    'More time for what matters.\nKeep your plans, invoices and payments in order.',
                    style: TextStyle(color: WorkspaceTheme.muted, fontSize: 14),
                  ),
                  const SizedBox(height: 24),
                  FilledButton.icon(
                    onPressed: () => context.go('/plans'),
                    label: const Text('Explore plans'),
                    icon: const Icon(Icons.arrow_forward_rounded, size: 18),
                    iconAlignment: IconAlignment.end,
                  ),
                ],
              ),
            ),
            if (constraints.maxWidth >= 1200 &&
                MediaQuery.textScalerOf(context).scale(1) < 1.5) ...[
              const SizedBox(width: 24),
              const _WorkspaceIllustration(),
            ],
          ],
        ),
      );
      final account = _Panel(
        color: WorkspaceTheme.soft,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                CircleAvatar(
                  radius: 23,
                  backgroundColor: WorkspaceTheme.selected,
                  foregroundColor: WorkspaceTheme.ink,
                  child: Text(first.isEmpty ? 'S' : first[0].toUpperCase()),
                ),
                const SizedBox(width: 12),
                const Expanded(
                  child: Text(
                    'Your account',
                    style: TextStyle(fontWeight: FontWeight.w500),
                  ),
                ),
                const Icon(
                  Icons.verified_user_outlined,
                  size: 20,
                  color: WorkspaceTheme.muted,
                ),
              ],
            ),
            const SizedBox(height: 20),
            Text(
              user.fullName,
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 4),
            Text(
              user.email,
              style: const TextStyle(color: WorkspaceTheme.muted),
              overflow: TextOverflow.ellipsis,
            ),
            const SizedBox(height: 16),
            const Pill('Customer account', tone: 'brand'),
            const SizedBox(height: 16),
            TextButton.icon(
              onPressed: () => context.go('/profile'),
              icon: const Icon(Icons.tune_rounded, size: 18),
              label: const Text('Manage profile'),
              style: TextButton.styleFrom(
                foregroundColor: WorkspaceTheme.ink,
                minimumSize: const Size(0, 44),
              ),
            ),
          ],
        ),
      );
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (wide)
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(flex: 7, child: welcome),
                const SizedBox(width: 20),
                Expanded(flex: 3, child: account),
              ],
            )
          else ...[
            welcome,
            const SizedBox(height: 20),
            account,
          ],
          const SizedBox(height: 32),
          if (wide)
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(flex: 7, child: _catalogue(context)),
                const SizedBox(width: 20),
                Expanded(flex: 3, child: const _QuickActions()),
              ],
            )
          else ...[
            _catalogue(context),
            const SizedBox(height: 28),
            const _QuickActions(),
          ],
        ],
      );
    },
  );

  Widget _catalogue(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      Wrap(
        alignment: WrapAlignment.spaceBetween,
        crossAxisAlignment: WrapCrossAlignment.center,
        spacing: 12,
        children: [
          const Text(
            'Discover your next plan',
            style: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.w500,
              letterSpacing: -0.5,
            ),
          ),
          TextButton.icon(
            onPressed: () => context.go('/plans'),
            label: const Text('View all plans'),
            icon: const Icon(Icons.arrow_forward_rounded, size: 16),
            iconAlignment: IconAlignment.end,
            style: TextButton.styleFrom(foregroundColor: WorkspaceTheme.ink),
          ),
        ],
      ),
      const Text(
        'Find the right fit. See what’s included before you choose.',
        style: TextStyle(color: WorkspaceTheme.muted),
      ),
      const SizedBox(height: 20),
      if (plans.isEmpty)
        const _Panel(
          child: EmptyState(
            icon: Icons.layers_outlined,
            title: 'New possibilities are on the way',
            message: 'Available plans will appear here. Check back soon.',
          ),
        )
      else
        LayoutBuilder(
          builder: (context, c) {
            final columns = c.maxWidth >= 650 && plans.length > 1 ? 2 : 1;
            return Wrap(
              spacing: 16,
              runSpacing: 16,
              children: [
                for (final plan in plans.take(4))
                  SizedBox(
                    width: (c.maxWidth - (columns - 1) * 16) / columns,
                    child: _PlanPreview(plan: plan),
                  ),
              ],
            );
          },
        ),
    ],
  );
}

class _PlanPreview extends StatelessWidget {
  const _PlanPreview({required this.plan});
  final Plan plan;

  @override
  Widget build(BuildContext context) => _Panel(
    child: LayoutBuilder(
      builder: (context, constraints) {
        final details = Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const _IconBadge(Icons.layers_outlined),
                const SizedBox(width: 12),
                Flexible(child: Pill(plan.billingLabel, tone: 'brand')),
              ],
            ),
            const SizedBox(height: 20),
            Text(
              plan.name,
              style: const TextStyle(fontSize: 19, fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 8),
            Text(
              plan.description?.trim().isNotEmpty == true
                  ? plan.description!
                  : 'Explore the products and options included in this plan.',
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(color: WorkspaceTheme.muted),
            ),
            const SizedBox(height: 20),
            Text(
              Fmt.inr(plan.price),
              style: const TextStyle(
                fontSize: 28,
                fontWeight: FontWeight.w500,
                letterSpacing: -1,
              ),
            ),
            Text(
              'Billed ${plan.billingLabel.toLowerCase()}',
              style: const TextStyle(color: WorkspaceTheme.muted, fontSize: 12),
            ),
          ],
        );
        final included = Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (plan.lines.isNotEmpty) ...[
              const _Eyebrow('INCLUDED IN THIS PLAN'),
              const SizedBox(height: 16),
            ],
            for (final line in plan.lines.take(2))
              Padding(
                padding: const EdgeInsets.only(bottom: 10),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Icon(
                      Icons.check_circle_outline_rounded,
                      size: 18,
                      color: WorkspaceTheme.muted,
                    ),
                    const SizedBox(width: 8),
                    Expanded(child: Text(line.label)),
                  ],
                ),
              ),
            const SizedBox(height: 10),
            OutlinedButton(
              onPressed: () => context.go(
                plan.lines.isEmpty
                    ? '/plans'
                    : Uri(
                        path: '/plans',
                        queryParameters: {
                          'product_id': '${plan.lines.first.productId}',
                          'product_name': plan.lines.first.productName,
                        },
                      ).toString(),
              ),
              style: OutlinedButton.styleFrom(
                side: const BorderSide(color: WorkspaceTheme.border),
              ),
              child: Text('Explore ${plan.name}', textAlign: TextAlign.center),
            ),
          ],
        );
        if (constraints.maxWidth >= 600) {
          return Row(
            children: [
              Expanded(child: details),
              const SizedBox(width: 40),
              Expanded(child: included),
            ],
          );
        }
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [details, const Divider(height: 32), included],
        );
      },
    ),
  );
}

class _QuickActions extends StatelessWidget {
  const _QuickActions();

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      const Padding(
        padding: EdgeInsets.symmetric(vertical: 10),
        child: Text(
          'Your essentials',
          style: TextStyle(
            fontSize: 20,
            fontWeight: FontWeight.w500,
            letterSpacing: -0.5,
          ),
        ),
      ),
      const SizedBox(height: 16),
      _Panel(
        child: Column(
          children: [
            for (final (index, item) in [
              (
                Icons.autorenew_rounded,
                'Subscriptions',
                'View and manage your plans',
                '/my-subscriptions',
              ),
              (
                Icons.receipt_long_outlined,
                'Invoices',
                'Find your bills and due dates',
                '/my-invoices',
              ),
              (
                Icons.account_balance_wallet_outlined,
                'Payments',
                'Keep track of your payments',
                '/my-payments',
              ),
            ].indexed) ...[
              if (index > 0) const Divider(height: 24),
              Material(
                color: Colors.transparent,
                child: InkWell(
                  borderRadius: BorderRadius.circular(12),
                  onTap: () => context.go(item.$4),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    child: Row(
                      children: [
                        _IconBadge(item.$1),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                item.$2,
                                style: const TextStyle(
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                              const SizedBox(height: 3),
                              Text(
                                item.$3,
                                style: const TextStyle(
                                  fontSize: 12,
                                  color: WorkspaceTheme.muted,
                                ),
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(width: 8),
                        const Icon(
                          Icons.chevron_right_rounded,
                          size: 18,
                          color: WorkspaceTheme.muted,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
      const SizedBox(height: 20),
      const _Panel(
        color: WorkspaceTheme.lavender,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(
              Icons.lightbulb_outline_rounded,
              color: WorkspaceTheme.ink,
              size: 24,
            ),
            SizedBox(height: 12),
            Text(
              'Everything, in one place.',
              style: TextStyle(fontSize: 15, fontWeight: FontWeight.w500),
            ),
            SizedBox(height: 8),
            Text(
              'Your subscriptions show what’s active. Your invoices show what’s due. Your payment history keeps the record.',
              style: TextStyle(color: WorkspaceTheme.muted),
            ),
          ],
        ),
      ),
    ],
  );
}

class _Panel extends StatelessWidget {
  const _Panel({required this.child, this.color = Colors.white});
  final Widget child;
  final Color color;

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(24),
    decoration: BoxDecoration(
      color: color,
      borderRadius: BorderRadius.circular(22),
      border: Border.all(color: WorkspaceTheme.border),
    ),
    child: child,
  );
}

class _IconBadge extends StatelessWidget {
  const _IconBadge(this.icon);
  final IconData icon;
  @override
  Widget build(BuildContext context) => Container(
    width: 44,
    height: 44,
    decoration: BoxDecoration(
      color: WorkspaceTheme.lavender,
      borderRadius: BorderRadius.circular(14),
    ),
    child: Icon(icon, size: 22, color: WorkspaceTheme.ink),
  );
}

class _Eyebrow extends StatelessWidget {
  const _Eyebrow(this.text);
  final String text;
  @override
  Widget build(BuildContext context) => Text(
    text,
    style: const TextStyle(
      fontSize: 11,
      letterSpacing: 1.8,
      fontWeight: FontWeight.w600,
      color: WorkspaceTheme.muted,
    ),
  );
}

/// Decorative subscription cards; excluded from screen-reader navigation.
class _WorkspaceIllustration extends StatelessWidget {
  const _WorkspaceIllustration();
  @override
  Widget build(BuildContext context) => ExcludeSemantics(
    child: SizedBox(
      width: 200,
      height: 190,
      child: Stack(
        alignment: Alignment.center,
        children: [
          Container(
            width: 176,
            height: 176,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              border: Border.all(color: WorkspaceTheme.selected),
            ),
          ),
          Transform.rotate(
            angle: -0.16,
            child: Container(
              width: 140,
              height: 160,
              decoration: BoxDecoration(
                color: WorkspaceTheme.selected,
                borderRadius: BorderRadius.circular(22),
              ),
            ),
          ),
          Transform.rotate(
            angle: 0.08,
            child: Container(
              width: 140,
              height: 160,
              padding: const EdgeInsets.all(22),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(22),
                boxShadow: const [
                  BoxShadow(
                    color: Color(0x148066FF),
                    blurRadius: 24,
                    offset: Offset(0, 12),
                  ),
                ],
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const _IconBadge(Icons.all_inclusive_rounded),
                  const Spacer(),
                  Container(
                    height: 8,
                    width: 75,
                    decoration: BoxDecoration(
                      color: WorkspaceTheme.selected,
                      borderRadius: BorderRadius.circular(4),
                    ),
                  ),
                  const SizedBox(height: 10),
                  Container(
                    height: 6,
                    width: 50,
                    decoration: BoxDecoration(
                      color: WorkspaceTheme.lavender,
                      borderRadius: BorderRadius.circular(4),
                    ),
                  ),
                ],
              ),
            ),
          ),
          const Positioned(
            right: 0,
            top: 12,
            child: CircleAvatar(
              radius: 19,
              backgroundColor: WorkspaceTheme.ink,
              child: Icon(Icons.check_rounded, color: Colors.white, size: 20),
            ),
          ),
        ],
      ),
    ),
  );
}
