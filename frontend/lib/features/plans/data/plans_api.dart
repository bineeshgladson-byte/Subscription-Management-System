import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';
import '../../../core/network/navigation_cache.dart';

class PlanLine {
  PlanLine.fromJson(Map<String, dynamic> j)
    : productId = toInt(j['product_id']),
      productName = str(j['product_name']),
      variantId = intOrNull(j['variant_id']),
      variantName = strOrNull(j['variant_name']),
      quantity = toInt(j['quantity'], 1),
      priceOverride = strOrNull(j['price_override']),
      unitPrice = str(j['unit_price'], '0'),
      lineTotal = str(j['line_total'], '0');
  final int productId, quantity;
  final int? variantId;
  final String productName, unitPrice, lineTotal;
  final String? variantName, priceOverride;
  String get label =>
      variantName == null ? productName : '$productName · $variantName';
}

class Plan {
  Plan.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      name = str(j['name']),
      description = strOrNull(j['description']),
      price = str(j['price'], '0'),
      billingPeriod = str(j['billing_period']),
      billingInterval = toInt(j['billing_interval'], 1),
      billingLabel = str(j['billing_label']),
      minQuantity = toInt(j['min_quantity'], 1),
      availableFrom = str(j['available_from']),
      availableUntil = strOrNull(j['available_until']),
      isAvailableNow = toBool(j['is_available_now']),
      autoCloseEnabled = toBool(j['auto_close_enabled']),
      autoCloseAfterCycles = intOrNull(j['auto_close_after_cycles']),
      isPausable = toBool(j['is_pausable']),
      isClosable = toBool(j['is_closable']),
      isRenewable = toBool(j['is_renewable']),
      status = str(j['status']),
      lines = listOf(j['products']).map(PlanLine.fromJson).toList(),
      activeSubscriptions = intOrNull(j['active_subscriptions']);
  final int id, billingInterval, minQuantity;
  final String name, price, billingPeriod, billingLabel, availableFrom, status;
  final String? description, availableUntil;
  final bool isAvailableNow,
      autoCloseEnabled,
      isPausable,
      isClosable,
      isRenewable;
  final int? autoCloseAfterCycles;
  final List<PlanLine> lines;
  final int? activeSubscriptions; // null for customers
  bool get isActive => status == 'active';
}

typedef PlanQuery = ({String q, String status, String? period});

const billingPeriods = {
  'daily': 'Daily',
  'weekly': 'Weekly',
  'monthly': 'Monthly',
  'yearly': 'Yearly',
};

class PlansRepository {
  PlansRepository(this._api);
  final ApiClient _api;

  Future<Paged<Plan>> list(PlanQuery q, {int? productId}) => _api.getPaged(
    '/plans',
    Plan.fromJson,
    query: {
      'q': q.q,
      'product_id': productId,
      'status': q.status,
      'billing_period': q.period,
      'page_size': 100,
      'sort': 'price',
    },
  );
  Future<void> create(Map<String, dynamic> body) =>
      _api.post('/plans', body: body);
  Future<void> update(int id, Map<String, dynamic> body) =>
      _api.put('/plans/$id', body: body);
  Future<void> setStatus(int id, String status) =>
      _api.patch('/plans/$id/status', body: {'status': status});
}

final plansRepositoryProvider = Provider(
  (ref) => PlansRepository(ref.watch(apiClientProvider)),
);

final plansProvider = FutureProvider.autoDispose.family<Paged<Plan>, PlanQuery>(
  (ref, q) =>
      cacheForNavigation(ref, () => ref.watch(plansRepositoryProvider).list(q)),
);

final activePlansProvider = FutureProvider.autoDispose<List<Plan>>(
  (ref) async => (await ref.watch(plansRepositoryProvider).list((
    q: '',
    status: 'active',
    period: null,
  ))).items,
);

final productPlansProvider = FutureProvider.autoDispose
    .family<Paged<Plan>, ({PlanQuery query, int productId})>(
      (ref, selection) => cacheForNavigation(
        ref,
        () => ref
            .watch(plansRepositoryProvider)
            .list(selection.query, productId: selection.productId),
      ),
    );
