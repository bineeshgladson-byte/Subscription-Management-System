import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';

class Tax {
  Tax.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      name = str(j['name']),
      rate = str(j['rate'], '0'),
      isActive = toBool(j['is_active']),
      productsUsing = toInt(j['products_using']);
  final int id;
  final String name, rate;
  final bool isActive;
  final int productsUsing;
}

class TaxesRepository {
  TaxesRepository(this._api);
  final ApiClient _api;

  Future<Paged<Tax>> list({bool? active}) => _api.getPaged(
    '/taxes',
    Tax.fromJson,
    query: {'is_active': active?.toString(), 'page_size': 100, 'sort': 'name'},
  );
  Future<void> create(String name, String rate) =>
      _api.post('/taxes', body: {'name': name, 'rate': rate});
  Future<void> update(int id, Map<String, dynamic> body) =>
      _api.put('/taxes/$id', body: body);
  Future<void> deactivate(int id) => _api.delete('/taxes/$id');
}

final taxesRepositoryProvider = Provider(
  (ref) => TaxesRepository(ref.watch(apiClientProvider)),
);

/// All taxes (for the taxes screen). [activeTaxesProvider] feeds product forms.
final taxesProvider = FutureProvider.autoDispose<Paged<Tax>>(
  (ref) => ref.watch(taxesRepositoryProvider).list(),
);
final activeTaxesProvider = FutureProvider.autoDispose<List<Tax>>(
  (ref) async =>
      (await ref.watch(taxesRepositoryProvider).list(active: true)).items,
);
