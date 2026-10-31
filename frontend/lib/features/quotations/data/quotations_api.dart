import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';

class QuotationLine {
  QuotationLine.fromJson(Map<String, dynamic> j)
      : id = toInt(j['id']),
        productId = toInt(j['product_id']),
        productName = str(j['product_name']),
        variantId = intOrNull(j['variant_id']),
        variantName = strOrNull(j['variant_name']),
        description = str(j['description']),
        quantity = toInt(j['quantity'], 1),
        unitPrice = str(j['unit_price'], '0'),
        taxId = intOrNull(j['tax_id']),
        taxRate = str(j['tax_rate'], '0'),
        lineSubtotal = str(j['line_subtotal'], '0');

  final int id, productId, quantity;
  final int? variantId, taxId;
  final String productName, description, unitPrice, taxRate, lineSubtotal;
  final String? variantName;
}

class Quotation {
  Quotation.fromJson(Map<String, dynamic> j)
      : id = toInt(j['id']),
        quotationNumber = str(j['quotation_number']),
        subscriptionNumber = strOrNull(j['subscription_number']),
        customerId = toInt(j['customer_id']),
        customerName = str(j['customer_name']),
        customerEmail = str(j['customer_email']),
        planId = toInt(j['plan_id']),
        planName = str(j['plan_name']),
        billingPeriod = str(j['billing_period']),
        templateId = intOrNull(j['template_id']),
        status = str(j['status']),
        startDate = str(j['start_date']),
        expiryDate = strOrNull(j['expiry_date']),
        quotationValidUntil = strOrNull(j['quotation_valid_until']),
        paymentTermsDays = toInt(j['payment_terms_days'], 15),
        nextBillingDate = strOrNull(j['next_billing_date']),
        subtotal = str(j['subtotal'], '0'),
        discountTotal = str(j['discount_total'], '0'),
        taxTotal = str(j['tax_total'], '0'),
        total = str(j['total'], '0'),
        items = listOf(j['items']).map(QuotationLine.fromJson).toList(),
        createdAt = dateOrNull(j['created_at']);

  final int id, customerId, planId, paymentTermsDays;
  final int? templateId;
  final String quotationNumber,
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
  final String? subscriptionNumber,
      expiryDate,
      quotationValidUntil,
      nextBillingDate;
  final List<QuotationLine> items;
  final DateTime? createdAt;

  bool get isDraft => status == 'draft';
  bool get isQuotation => status == 'quotation';
  bool get isConfirmed => status == 'confirmed';
}

typedef QuotationQuery = ({String q, String? status, int? customerId, int page});

class QuotationsRepository {
  QuotationsRepository(this._api);
  final ApiClient _api;

  Future<Paged<Quotation>> list(QuotationQuery q) => _api.getPaged(
        '/quotations',
        Quotation.fromJson,
        query: {
          'q': q.q,
          'status': q.status,
          'customer_id': q.customerId,
          'page': q.page,
          'page_size': 20,
        },
      );

  Future<Quotation> get(int id) async => Quotation.fromJson(
        await _api.get('/quotations/$id') as Map<String, dynamic>,
      );

  Future<void> create(Map<String, dynamic> body) =>
      _api.post('/quotations', body: body);

  Future<void> send(int id) => _api.post('/quotations/$id/send');
  Future<void> confirm(int id) => _api.post('/quotations/$id/confirm');
  Future<void> cancel(int id) => _api.post('/quotations/$id/cancel');
}

final quotationsRepositoryProvider = Provider(
  (ref) => QuotationsRepository(ref.watch(apiClientProvider)),
);

final quotationsProvider =
    FutureProvider.autoDispose.family<Paged<Quotation>, QuotationQuery>(
  (ref, q) => ref.watch(quotationsRepositoryProvider).list(q),
);
