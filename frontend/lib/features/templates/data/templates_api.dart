import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';

class TemplateLine {
  TemplateLine.fromJson(Map<String, dynamic> j)
    : productId = toInt(j['product_id']),
      productName = str(j['product_name']),
      variantId = intOrNull(j['variant_id']),
      variantName = strOrNull(j['variant_name']),
      quantity = toInt(j['quantity'], 1),
      unitPrice = str(j['unit_price'], '0'),
      lineTotal = str(j['line_total'], '0');
  final int productId, quantity;
  final int? variantId;
  final String productName, unitPrice, lineTotal;
  final String? variantName;
  String get label =>
      variantName == null ? productName : '$productName · $variantName';
}

class QuotationTemplate {
  QuotationTemplate.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      name = str(j['name']),
      planId = j['plan'] is Map ? toInt((j['plan'] as Map)['id']) : null,
      planName = j['plan'] is Map ? '${(j['plan'] as Map)['name']}' : null,
      planLabel = j['plan'] is Map
          ? '${(j['plan'] as Map)['billing_label']}'
          : null,
      validityDays = toInt(j['validity_days'], 15),
      paymentTermsDays = toInt(j['payment_terms_days'], 15),
      terms = strOrNull(j['terms']),
      isActive = toBool(j['is_active']),
      lines = listOf(j['lines']).map(TemplateLine.fromJson).toList(),
      estimatedTotal = str(j['estimated_total'], '0');
  final int id, validityDays, paymentTermsDays;
  final int? planId;
  final String name, estimatedTotal;
  final String? planName, planLabel, terms;
  final bool isActive;
  final List<TemplateLine> lines;
}

typedef TemplateQuery = ({String q, bool? active, int page});

class TemplatesRepository {
  TemplatesRepository(this._api);
  final ApiClient _api;

  Future<Paged<QuotationTemplate>> list(TemplateQuery q) => _api.getPaged(
    '/quotation-templates',
    QuotationTemplate.fromJson,
    query: {
      'q': q.q,
      'is_active': q.active?.toString(),
      'page': q.page,
      'page_size': 20,
      'sort': 'name',
    },
  );
  Future<void> create(Map<String, dynamic> body) =>
      _api.post('/quotation-templates', body: body);
  Future<void> update(int id, Map<String, dynamic> body) =>
      _api.put('/quotation-templates/$id', body: body);
  Future<void> deactivate(int id) => _api.delete('/quotation-templates/$id');
}

final templatesRepositoryProvider = Provider(
  (ref) => TemplatesRepository(ref.watch(apiClientProvider)),
);

final templatesProvider = FutureProvider.autoDispose
    .family<Paged<QuotationTemplate>, TemplateQuery>(
      (ref, q) => ref.watch(templatesRepositoryProvider).list(q),
    );
