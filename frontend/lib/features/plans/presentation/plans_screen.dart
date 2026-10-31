import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/theme/workspace_theme.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/customer_widgets.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../../core/widgets/workspace_widgets.dart';
import '../../auth/application/auth_controller.dart';
import '../../products/data/products_api.dart';
import '../../products/presentation/line_editor.dart';
import '../../sales/data/sales_api.dart';
import '../data/plans_api.dart';

/// Subscription plans from GET /plans.
///
/// Customer flow: a browse-by-product grid first; selecting a product shows
/// the tier-style pricing cards for every plan that includes that product.
/// Admin / staff flow: the same tier-style pricing cards for every plan,
/// with Edit / Archive controls (admins) and the per-plan subscriber stats
/// (staff + admins).
class PlansScreen extends ConsumerStatefulWidget {
  const PlansScreen({
    super.key,
    this.initialProductId,
    this.initialProductName,
  });
  final int? initialProductId;
  final String? initialProductName;
  @override
  ConsumerState<PlansScreen> createState() => _PlansScreenState();
}

class _PlansScreenState extends ConsumerState<PlansScreen> {
  PlanQuery _q = (q: '', status: 'active', period: null);
  // Customer only: the product the shopper picked from the browse grid.
  _ProductBucket? _picked;
  // Customer only: free-text filter against product name.
  String _productSearch = '';

  @override
  void initState() {
    super.initState();
    if (widget.initialProductId != null) {
      _picked = _ProductBucket(
        productId: widget.initialProductId!,
        name: widget.initialProductName ?? 'Product plans',
      );
    }
  }

  void _refreshPlans() {
    ref.invalidate(plansProvider);
    ref.invalidate(productPlansProvider);
    ref.invalidate(activePlansProvider);
  }

  void _allProducts() => setState(() {
    _picked = null;
    _q = (q: '', status: _q.status, period: _q.period);
  });

  @override
  Widget build(BuildContext context) {
    final role = ref.watch(authControllerProvider).value?.role ?? Role.customer;
    final isAdmin = role == Role.admin;
    final isCustomer = role == Role.customer;
    return PageFrame(
      title: _picked?.name ?? (isCustomer ? 'Browse products' : 'Plans'),
      subtitle: isCustomer
          ? (_picked == null
                ? 'Pick a product to see every plan it belongs to. Prices exclude tax.'
                : 'Compare the plans for this product. Prices exclude tax.')
          : 'How often each plan bills, what it includes and what customers may do with it.',
      action: Wrap(
        spacing: 8,
        runSpacing: 8,
        children: [
          if (_picked != null)
            OutlinedButton.icon(
              onPressed: _allProducts,
              icon: const Icon(Icons.arrow_back),
              label: const Text('All products'),
            ),
          if (isAdmin)
            FilledButton.icon(
              onPressed: () => _edit(null),
              icon: const Icon(Icons.add),
              label: const Text('Create Plan'),
            )
          else if (_picked == null)
            OutlinedButton.icon(
              onPressed: _refreshPlans,
              icon: const Icon(Icons.refresh_rounded, size: 18),
              label: const Text('Refresh'),
            ),
        ],
      ),
      toolbar: isCustomer
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                CustomerIntro(
                  icon: Icons.layers_outlined,
                  eyebrow: _picked == null ? 'Find your fit' : 'A plan for you',
                  title: _picked == null
                      ? 'Good things start with the right plan.'
                      : 'Choose the way you subscribe.',
                  message: _picked == null
                      ? 'Explore our products, compare billing options and find what works for you.'
                      : 'Compare what’s included and request a quotation when you’re ready. Prices exclude tax.',
                ),
                const SizedBox(height: 24),
                _buildToolbar(isCustomer),
              ],
            )
          : _buildToolbar(isCustomer),
      child: AsyncView(
        value: _picked == null
            ? ref.watch(plansProvider(_q))
            : ref.watch(
                productPlansProvider((
                  query: _q,
                  productId: _picked!.productId,
                )),
              ),
        onRetry: _refreshPlans,
        builder: (page) {
          final plans = page.items;
          if (_picked != null) {
            if (plans.isEmpty) {
              return CustomerEmpty(
                icon: Icons.layers_outlined,
                title: 'No plans for this product yet',
                message: isAdmin
                    ? 'Use Create Plan to add a billing option for this product.'
                    : 'Try a different billing filter or check back soon.',
              );
            }
            return Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (!isCustomer) ...[
                  _StaffMetrics(plans: plans),
                  const SizedBox(height: 20),
                ],
                _buildPricingGrid(
                  context,
                  plans,
                  isAdmin: isAdmin,
                  isCustomer: isCustomer,
                ),
              ],
            );
          }
          if (isCustomer) {
            return CustomerSection(
              title: 'Explore the collection',
              caption: 'Choose a product to discover its available plans.',
              child: _buildProductGrid(context, plans, isCustomer: true),
            );
          }
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _StaffMetrics(plans: plans),
              const SizedBox(height: 24),
              AsyncView(
                value: ref.watch(activeProductsProvider),
                onRetry: () => ref.invalidate(activeProductsProvider),
                builder: (products) => CustomerSection(
                  title: 'Plans by product',
                  caption: isAdmin
                      ? 'Choose a product to manage its plans or add a new billing option.'
                      : 'Choose a product to view its plans and billing options.',
                  child: _buildProductGrid(
                    context,
                    plans,
                    isCustomer: false,
                    catalogue: products,
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }

  // ---- Toolbar ------------------------------------------------------------

  Widget _buildToolbar(bool isCustomer) {
    // Customer, browsing products — search by product, not by plan.
    if (_picked == null) {
      return Toolbar(
        children: [
          SearchBox(
            hint: 'Search products',
            onChanged: (v) => setState(() => _productSearch = v),
          ),
          FilterDropdown<String>(
            label: 'Billing',
            value: _q.period,
            options: billingPeriods,
            onChanged: (v) =>
                setState(() => _q = (q: '', status: _q.status, period: v)),
          ),
          if (!isCustomer)
            FilterDropdown<String>(
              label: 'Status',
              value: _q.status == 'active' ? null : _q.status,
              allLabel: 'Active',
              options: const {'archived': 'Archived', 'all': 'All'},
              onChanged: (value) => setState(
                () =>
                    _q = (q: '', status: value ?? 'active', period: _q.period),
              ),
            ),
        ],
      );
    }
    // Everywhere else — the existing plan-filter toolbar.
    return Toolbar(
      children: [
        SearchBox(
          hint: 'Search plans',
          onChanged: (v) =>
              setState(() => _q = (q: v, status: _q.status, period: _q.period)),
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
    );
  }

  // ---- Customer browse (product grid) -------------------------------------

  Widget _buildProductGrid(
    BuildContext context,
    List<Plan> plans, {
    required bool isCustomer,
    List<Product> catalogue = const [],
  }) {
    var buckets = _bucketProducts(plans, availableOnly: isCustomer);
    if (!isCustomer) {
      for (final product in catalogue) {
        if (!buckets.any((b) => b.productId == product.id)) {
          buckets.add(
            _ProductBucket(productId: product.id, name: product.name),
          );
        }
      }
      buckets.sort(
        (a, b) => a.name.toLowerCase().compareTo(b.name.toLowerCase()),
      );
    }
    final needle = _productSearch.trim().toLowerCase();
    if (needle.isNotEmpty) {
      buckets = buckets
          .where((b) => b.name.toLowerCase().contains(needle))
          .toList();
    }
    if (buckets.isEmpty) {
      return const EmptyState(
        icon: Icons.inventory_2_outlined,
        title: 'No products match',
        message: 'Try a different search or billing filter.',
      );
    }
    return LayoutBuilder(
      builder: (context, c) {
        final cols = c.maxWidth >= 1200
            ? 4
            : c.maxWidth >= 900
            ? 3
            : c.maxWidth >= 580
            ? 2
            : 1;
        final w = (c.maxWidth - (cols - 1) * 16) / cols;
        return Wrap(
          spacing: 16,
          runSpacing: 16,
          children: [
            for (final b in buckets)
              SizedBox(
                width: w,
                child: _ProductCard(
                  bucket: b,
                  isCustomer: isCustomer,
                  onOpen: () => setState(() => _picked = b),
                ),
              ),
          ],
        );
      },
    );
  }

  // ---- Shared tier-card grid ---------------------------------------------

  Widget _buildPricingGrid(
    BuildContext context,
    List<Plan> plans, {
    required bool isAdmin,
    required bool isCustomer,
  }) {
    // The reference layout highlights the middle tier as "Most popular".
    // We pick the second plan when there are at least two.
    final popularIdx = plans.length >= 2 ? 1 : -1;
    return LayoutBuilder(
      builder: (context, c) {
        final cols = c.maxWidth >= 1200
            ? 4
            : c.maxWidth >= 900
            ? 3
            : c.maxWidth >= 580
            ? 2
            : 1;
        final w = (c.maxWidth - (cols - 1) * 16) / cols;
        return Wrap(
          spacing: 16,
          runSpacing: 24,
          children: [
            for (var i = 0; i < plans.length; i++)
              SizedBox(
                width: w,
                child: _PricingCard(
                  plan: plans[i],
                  tierIndex: i,
                  isPopular: i == popularIdx,
                  isAdmin: isAdmin,
                  isCustomer: isCustomer,
                  onEdit: () => _edit(plans[i]),
                  onToggle: () => _toggle(plans[i]),
                  onRequestQuote: () => _requestQuote(plans[i]),
                ),
              ),
          ],
        );
      },
    );
  }

  // ---- Helpers ------------------------------------------------------------

  /// Group the plans the customer can see by the products they include.
  /// A plan with three products shows up under all three. Only plans active
  /// AND available today are counted (that's what the customer could buy now).
  List<_ProductBucket> _bucketProducts(
    List<Plan> plans, {
    bool availableOnly = true,
  }) {
    final byId = <int, _ProductBucket>{};
    for (final p in plans) {
      if (availableOnly && (!p.isActive || !p.isAvailableNow)) continue;
      for (final line in p.lines) {
        final b = byId.putIfAbsent(
          line.productId,
          () =>
              _ProductBucket(productId: line.productId, name: line.productName),
        );
        if (!b.plans.any((pp) => pp.id == p.id)) b.plans.add(p);
      }
    }
    final out = byId.values.toList()
      ..sort((a, b) => a.name.toLowerCase().compareTo(b.name.toLowerCase()));
    return out;
  }

  // ---- Actions ------------------------------------------------------------

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
      _refreshPlans();
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
        ? [LineDraft(productId: _picked?.productId)]
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
      _refreshPlans();
      showMessage(context, p == null ? 'Plan created' : 'Plan updated');
    }
  }

  Future<void> _requestQuote(Plan p) async {
    if (!p.isActive || !p.isAvailableNow) return;
    if (!mounted) return;
    final ok = await runAction(
      context,
      () => ref.read(salesRepositoryProvider).requestMyQuotation({
        'plan_id': p.id,
        'start_date': Fmt.apiDate(DateTime.now()),
      }),
      success: 'Quotation requested. Our team will review it.',
    );
    if (!ok || !mounted) return;
    ref.invalidate(mySubscriptionsProvider);
  }
}

// ============================================================================
// Models / helpers that are private to this screen
// ============================================================================

class _ProductBucket {
  _ProductBucket({required this.productId, required this.name});
  final int productId;
  final String name;
  final List<Plan> plans = [];

  /// Lowest headline price across the bucket's plans. Prices are stored as
  /// strings on the wire; parse defensively.
  double get lowestPrice => plans
      .map((p) => double.tryParse(p.price) ?? double.infinity)
      .reduce((a, b) => a < b ? a : b);

  /// A short preview of billing cadences — "Monthly · Yearly" — so the
  /// browse card hints at what's available without opening the drawer.
  String get cadenceSummary {
    final seen = <String>{for (final p in plans) p.billingLabel};
    return seen.join(' · ');
  }

  /// Pick a representative plan line (first match) to show the variant blurb
  /// on the product card.
  PlanLine? get sampleLine {
    for (final p in plans) {
      for (final l in p.lines) {
        if (l.productId == productId) return l;
      }
    }
    return null;
  }
}

// ============================================================================
// Admin / staff metrics row (matches reference image #1)
// ============================================================================

class _StaffMetrics extends StatelessWidget {
  const _StaffMetrics({required this.plans});
  final List<Plan> plans;
  @override
  Widget build(BuildContext context) {
    final subscribers = plans.fold<int>(
      0,
      (s, p) => s + (p.activeSubscriptions ?? 0),
    );
    final mrr = plans.fold<double>(0, (s, p) {
      final price = double.tryParse(p.price) ?? 0;
      final count = p.activeSubscriptions ?? 0;
      return s + _toMonthly(price, p.billingPeriod, p.billingInterval) * count;
    });
    return Wrap(
      spacing: 12,
      runSpacing: 12,
      children: [
        SizedBox(
          width: 220,
          child: MetricTile(label: 'Total plans', value: '${plans.length}'),
        ),
        SizedBox(
          width: 220,
          child: MetricTile(
            label: 'Available now',
            value: '${plans.where((p) => p.isAvailableNow).length}',
          ),
        ),
        SizedBox(
          width: 220,
          child: MetricTile(label: 'Active subscribers', value: '$subscribers'),
        ),
        SizedBox(
          width: 240,
          child: MetricTile(
            label: 'Total MRR from plans',
            value: Fmt.inr(mrr.toStringAsFixed(2)),
          ),
        ),
      ],
    );
  }

  /// Rough normalisation so MRR is comparable across billing periods.
  /// Weekly ≈ 4.345 months, yearly = 1/12, daily ≈ 30, with the plan's
  /// `billing_interval` factored in (every N periods).
  static double _toMonthly(double price, String period, int interval) {
    final n = interval <= 0 ? 1 : interval;
    final each = price / n;
    return switch (period) {
      'daily' => each * 30,
      'weekly' => each * 4.345,
      'yearly' => each / 12,
      _ => each,
    };
  }
}

// ============================================================================
// Browse card — one per product (customer only)
// ============================================================================

class _ProductCard extends StatelessWidget {
  const _ProductCard({
    required this.bucket,
    required this.onOpen,
    required this.isCustomer,
  });
  final bool isCustomer;
  final _ProductBucket bucket;
  final VoidCallback onOpen;

  @override
  Widget build(BuildContext context) {
    final line = bucket.sampleLine;
    return InkWell(
      onTap: onOpen,
      borderRadius: BorderRadius.circular(18),
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(18),
          border: Border.all(color: WorkspaceTheme.border),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _TierIcon(index: bucket.productId % 4),
            const SizedBox(height: 14),
            Text(bucket.name, style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 4),
            Text(
              line?.variantName == null
                  ? '${bucket.plans.length} plan${bucket.plans.length == 1 ? '' : 's'} available'
                  : 'Includes ${line!.variantName}',
              style: const TextStyle(fontSize: 12, color: WorkspaceTheme.muted),
            ),
            const SizedBox(height: 16),
            if (bucket.plans.isEmpty)
              const Text(
                'No plans yet',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
              )
            else
              Text.rich(
                TextSpan(
                  children: [
                    const TextSpan(
                      text: 'From  ',
                      style: TextStyle(
                        fontSize: 11,
                        color: WorkspaceTheme.muted,
                      ),
                    ),
                    TextSpan(
                      text: Fmt.inr(bucket.lowestPrice.toStringAsFixed(2)),
                      style: const TextStyle(
                        fontSize: 24,
                        fontWeight: FontWeight.w600,
                        letterSpacing: -0.6,
                      ),
                    ),
                  ],
                ),
              ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: [
                Pill(
                  '${bucket.plans.length} plan${bucket.plans.length == 1 ? '' : 's'}',
                  tone: 'brand',
                ),
                if (bucket.cadenceSummary.isNotEmpty)
                  Pill(bucket.cadenceSummary),
              ],
            ),
            const SizedBox(height: 16),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: onOpen,
                icon: const Icon(Icons.arrow_forward, size: 18),
                label: Text(isCustomer ? 'View plans' : 'Manage plans'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ============================================================================
// Pricing / tier card — admin, staff and customer
// ============================================================================

class _PricingCard extends StatelessWidget {
  const _PricingCard({
    required this.plan,
    required this.tierIndex,
    required this.isPopular,
    required this.isAdmin,
    required this.isCustomer,
    required this.onEdit,
    required this.onToggle,
    required this.onRequestQuote,
  });
  final Plan plan;
  final int tierIndex;
  final bool isPopular, isAdmin, isCustomer;
  final VoidCallback onEdit, onToggle, onRequestQuote;

  @override
  Widget build(BuildContext context) {
    final p = plan;
    final period = switch (p.billingPeriod) {
      'daily' => 'day',
      'weekly' => 'week',
      'yearly' => 'year',
      _ => 'mo',
    };
    final per = p.billingInterval > 1
        ? '${p.billingInterval} ${period}s'
        : period;

    final body = Container(
      padding: const EdgeInsets.all(22),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(
          color: isPopular
              ? const Color(0xFFF4B36A)
              : (p.isActive ? WorkspaceTheme.border : const Color(0xFFF3C98B)),
          width: isPopular ? 1.6 : 1,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              _TierIcon(index: tierIndex),
              const Spacer(),
              if (isAdmin)
                PopupMenuButton<String>(
                  tooltip: 'Manage ${p.name}',
                  icon: const Icon(Icons.more_vert, size: 18),
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
          const SizedBox(height: 16),
          Text(p.name, style: Theme.of(context).textTheme.titleLarge),
          if (p.description != null) ...[
            const SizedBox(height: 4),
            Text(
              p.description!,
              style: const TextStyle(
                fontSize: 12,
                color: WorkspaceTheme.muted,
                height: 1.4,
              ),
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
          ],
          const SizedBox(height: 20),
          Text.rich(
            TextSpan(
              children: [
                TextSpan(
                  text: Fmt.inr(p.price),
                  style: const TextStyle(
                    fontSize: 30,
                    fontWeight: FontWeight.w600,
                    letterSpacing: -0.9,
                  ),
                ),
                TextSpan(
                  text: '  /$per',
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
          const SizedBox(height: 16),
          for (final l in p.lines)
            Padding(
              padding: const EdgeInsets.only(bottom: 6),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
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
          const SizedBox(height: 12),
          Text(
            'Available from ${Fmt.date(p.availableFrom)}'
            '${p.availableUntil == null ? '' : ' until ${Fmt.date(p.availableUntil!)}'}',
            style: Theme.of(context).textTheme.bodySmall,
          ),
          if (!isCustomer && p.activeSubscriptions != null) ...[
            const SizedBox(height: 14),
            _SubscribersFooter(plan: p),
          ],
          if (isCustomer) ...[
            const SizedBox(height: 14),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: (p.isActive && p.isAvailableNow)
                    ? onRequestQuote
                    : null,
                icon: const Icon(Icons.request_quote_outlined),
                label: const Text('Request quotation'),
              ),
            ),
          ],
        ],
      ),
    );

    if (!isPopular) return body;
    // "Most popular" dark pill overlays the top edge, like the reference.
    return Stack(
      clipBehavior: Clip.none,
      children: [
        Padding(padding: const EdgeInsets.only(top: 12), child: body),
        Positioned(
          top: 0,
          left: 0,
          right: 0,
          child: Center(
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 5),
              decoration: BoxDecoration(
                color: WorkspaceTheme.ink,
                borderRadius: BorderRadius.circular(99),
              ),
              child: const Text(
                'Most popular',
                style: TextStyle(
                  color: Colors.white,
                  fontSize: 11,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }
}

/// Subscribers / revenue footer, admin + staff only.
///
/// The MRR split and the revenue-share bar imitate the reference image's
/// mini-stat footer. We derive both from `active_subscriptions` × the plan
/// price (normalised to monthly), and skip the bar when nothing is known.
class _SubscribersFooter extends StatelessWidget {
  const _SubscribersFooter({required this.plan});
  final Plan plan;
  @override
  Widget build(BuildContext context) {
    final subs = plan.activeSubscriptions ?? 0;
    final mrr =
        _StaffMetrics._toMonthly(
          double.tryParse(plan.price) ?? 0,
          plan.billingPeriod,
          plan.billingInterval,
        ) *
        subs;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: WorkspaceTheme.soft,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: _FooterStat(label: 'Subscribers', value: '$subs'),
              ),
              Expanded(
                child: _FooterStat(
                  label: 'MRR Contribution',
                  value: Fmt.inr(mrr.toStringAsFixed(2)),
                  alignEnd: true,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _FooterStat extends StatelessWidget {
  const _FooterStat({
    required this.label,
    required this.value,
    this.alignEnd = false,
  });
  final String label, value;
  final bool alignEnd;
  @override
  Widget build(BuildContext context) {
    final align = alignEnd ? CrossAxisAlignment.end : CrossAxisAlignment.start;
    return Column(
      crossAxisAlignment: align,
      children: [
        Text(
          label,
          style: const TextStyle(fontSize: 10, color: WorkspaceTheme.muted),
        ),
        const SizedBox(height: 2),
        Text(
          value,
          style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
        ),
      ],
    );
  }
}

// ============================================================================
// Icon chip — maps a tier index to an icon + background tint. The icons match
// the reference layout (sparkle, lightning, crown, enterprise building).
// ============================================================================

class _TierIcon extends StatelessWidget {
  const _TierIcon({required this.index});
  final int index;
  @override
  Widget build(BuildContext context) {
    final (icon, tint) = _pick(index);
    return Container(
      width: 44,
      height: 44,
      decoration: BoxDecoration(
        color: tint.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(12),
      ),
      alignment: Alignment.center,
      child: Icon(icon, size: 22, color: tint),
    );
  }

  static (IconData, Color) _pick(int i) {
    switch (i % 4) {
      case 0:
        return (Icons.auto_awesome, WorkspaceTheme.purple);
      case 1:
        return (Icons.bolt, const Color(0xFFF4B36A));
      case 2:
        return (Icons.workspace_premium, const Color(0xFFE89B2A));
      default:
        return (Icons.business, WorkspaceTheme.ink);
    }
  }
}
