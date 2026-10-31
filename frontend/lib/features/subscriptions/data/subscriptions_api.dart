import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';
import '../../quotations/data/quotations_api.dart';

class Subscription {
  Subscription.fromJson(Map<String, dynamic> j)
      : id = toInt(j['id']),
        quotationNumber = str(j['quotation_number']),
        subscriptionNumber = str(j['subscription_number'], 'SUB-PENDING'),
        customerId = toInt(j['customer_id']),
        customerName = str(j['customer_name']),
        customerEmail = str(j['customer_email']),
        planId = toInt(j['plan_id']),
        planName = str(j['plan_name']),
        billingPeriod = str(j['billing_period']),
        status = str(j['status']),
        startDate = str(j['start_date']),
        expiryDate = strOrNull(j['expiry_date']),
        nextBillingDate = strOrNull(j['next_billing_date']),
        paymentTermsDays = toInt(j['payment_terms_days'], 15),
        billingAnchorDay = toInt(j['billing_anchor_day'], 1),
        cyclesBilled = toInt(j['cycles_billed'], 0),
        closeReason = strOrNull(j['close_reason']),
        subtotal = str(j['subtotal'], '0'),
        discountTotal = str(j['discount_total'], '0'),
        taxTotal = str(j['tax_total'], '0'),
        total = str(j['total'], '0'),
        items = listOf(j['items']).map(QuotationLine.fromJson).toList(),
        createdAt = dateOrNull(j['created_at']);

  final int id, customerId, planId, paymentTermsDays, billingAnchorDay, cyclesBilled;
  final String quotationNumber,
      subscriptionNumber,
      customerName,
      customerEmail,
      planName,
      billingPeriod,
      status,
      startDate,
      subtotal,
      discountTotal,
      taxTotal,
      total;
  final String? expiryDate, nextBillingDate, closeReason;
  final List<QuotationLine> items;
  final DateTime? createdAt;

  bool get isActive => status == 'active';
  bool get isPaused => status == 'paused';
  bool get isConfirmed => status == 'confirmed';
  bool get isClosed => status == 'closed';
}

typedef SubscriptionQuery = ({String q, String? status, int? customerId, int page});

class SubscriptionsRepository {
  SubscriptionsRepository(this._api);
  final ApiClient _api;

  Future<Paged<Subscription>> list(SubscriptionQuery q) => _api.getPaged(
        '/subscriptions',
        Subscription.fromJson,
        query: {
          'q': q.q,
          'status': q.status,
          'customer_id': q.customerId,
          'page': q.page,
          'page_size': 20,
        },
      );

  Future<Subscription> get(int id) async => Subscription.fromJson(
        await _api.get('/subscriptions/$id') as Map<String, dynamic>,
      );

  Future<void> activate(int id) => _api.post('/subscriptions/$id/activate');
  Future<void> pause(int id) => _api.post('/subscriptions/$id/pause');
  Future<void> resume(int id) => _api.post('/subscriptions/$id/resume');
  Future<void> close(int id, {String? reason}) =>
      _api.post('/subscriptions/$id/close', body: {'reason': reason ?? ''});
  Future<void> renew(int id, {String? newExpiryDate}) =>
      _api.post('/subscriptions/$id/renew', body: {
        'new_expiry_date': ?newExpiryDate,
      });
  Future<dynamic> billNow(int id) => _api.post('/subscriptions/$id/bill-now');
}

final subscriptionsRepositoryProvider = Provider(
  (ref) => SubscriptionsRepository(ref.watch(apiClientProvider)),
);

final subscriptionsProvider =
    FutureProvider.autoDispose.family<Paged<Subscription>, SubscriptionQuery>(
  (ref, q) => ref.watch(subscriptionsRepositoryProvider).list(q),
);
