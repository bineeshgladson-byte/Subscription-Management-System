import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';

class Discount {
  Discount.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      name = str(j['name']),
      code = strOrNull(j['code']),
      discountType = str(j['discount_type']),
      value = str(j['value'], '0'),
      label = str(j['label']),
      scope = str(j['scope']),
      productIds = listOf(j['products']).map((p) => toInt(p['id'])).toList(),
      productNames = listOf(j['products']).map((p) => str(p['name'])).toList(),
      minPurchaseAmount = strOrNull(j['min_purchase_amount']),
      minQuantity = intOrNull(j['min_quantity']),
      startsOn = str(j['starts_on']),
      endsOn = strOrNull(j['ends_on']),
      maxUsage = intOrNull(j['max_usage']),
      usageCount = toInt(j['usage_count']),
      remainingUses = intOrNull(j['remaining_uses']),
      isActive = toBool(j['is_active']),
      state = str(j['state']);
  final int id, usageCount;
  final String name, discountType, value, label, scope, startsOn, state;
  final String? code, minPurchaseAmount, endsOn;
  final List<int> productIds;
  final List<String> productNames;
  final int? minQuantity, maxUsage, remainingUses;
  final bool isActive;
}

const discountStates = {
  'active': 'Active',
  'scheduled': 'Scheduled',
  'expired': 'Expired',
  'used_up': 'Used up',
  'inactive': 'Switched off',
};

typedef DiscountQuery = ({String q, String? state, int page});

class PricedLine {
  PricedLine.fromJson(Map<String, dynamic> j)
    : description = str(j['description']),
      quantity = toInt(j['quantity'], 1),
      unitPrice = str(j['unit_price'], '0'),
      subtotal = str(j['subtotal'], '0'),
      productDiscount = str(j['product_discount'], '0'),
      productDiscountName = strOrNull(j['product_discount_name']),
      subscriptionDiscount = str(j['subscription_discount'], '0'),
      taxRate = str(j['tax_rate'], '0'),
      taxAmount = str(j['tax_amount'], '0'),
      total = str(j['total'], '0');
  final String description,
      unitPrice,
      subtotal,
      productDiscount,
      subscriptionDiscount,
      taxRate,
      taxAmount,
      total;
  final String? productDiscountName;
  final int quantity;
}

class PricePreview {
  PricePreview.fromJson(Map<String, dynamic> j)
    : date = str(j['date']),
      subtotal = str(j['subtotal'], '0'),
      discountTotal = str(j['discount_total'], '0'),
      taxTotal = str(j['tax_total'], '0'),
      total = str(j['total'], '0'),
      lines = listOf(j['lines']).map(PricedLine.fromJson).toList(),
      discounts = listOf(j['discounts'])
          .map(
            (d) => (
              name: str(d['name']),
              amount: str(d['amount'], '0'),
              scope: str(d['scope']),
            ),
          )
          .toList();
  final String date, subtotal, discountTotal, taxTotal, total;
  final List<PricedLine> lines;
  final List<({String name, String amount, String scope})> discounts;
}

class DiscountsRepository {
  DiscountsRepository(this._api);
  final ApiClient _api;

  Future<Paged<Discount>> list(DiscountQuery q) => _api.getPaged(
    '/discounts',
    Discount.fromJson,
    query: {
      'q': q.q,
      'state': q.state,
      'page': q.page,
      'page_size': 20,
      'sort': '-created_at',
    },
  );
  Future<void> create(Map<String, dynamic> body) =>
      _api.post('/discounts', body: body);
  Future<void> update(int id, Map<String, dynamic> body) =>
      _api.put('/discounts/$id', body: body);
  Future<void> deactivate(int id) => _api.delete('/discounts/$id');

  Future<PricePreview> preview(
    List<Map<String, dynamic>> lines, {
    String? date,
  }) async => PricePreview.fromJson(
    await _api.post('/pricing/preview', body: {'lines': lines, 'date': ?date})
        as Map<String, dynamic>,
  );
}

final discountsRepositoryProvider = Provider(
  (ref) => DiscountsRepository(ref.watch(apiClientProvider)),
);

final discountsProvider = FutureProvider.autoDispose
    .family<Paged<Discount>, DiscountQuery>(
      (ref, q) => ref.watch(discountsRepositoryProvider).list(q),
    );
