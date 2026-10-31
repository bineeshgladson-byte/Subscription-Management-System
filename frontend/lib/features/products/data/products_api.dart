import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';

class Variant {
  Variant.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      attribute = str(j['attribute']),
      name = str(j['name']),
      extraPrice = str(j['extra_price'], '0'),
      price = str(j['price'], '0'),
      sku = strOrNull(j['sku']),
      isActive = toBool(j['is_active']);
  final int id;
  final String attribute, name, extraPrice, price;
  final String? sku;
  final bool isActive;
}

class Product {
  Product.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      name = str(j['name']),
      productType = str(j['product_type']),
      salesPrice = str(j['sales_price'], '0'),
      costPrice = strOrNull(j['cost_price']),
      margin = strOrNull(j['margin']),
      description = strOrNull(j['description']),
      taxId = j['tax'] is Map ? toInt((j['tax'] as Map)['id']) : null,
      taxLabel = j['tax'] is Map ? '${(j['tax'] as Map)['name']}' : null,
      status = str(j['status']),
      variants = listOf(j['variants']).map(Variant.fromJson).toList();
  final int id;
  final String name, productType, salesPrice, status;
  final String? costPrice, margin, description, taxLabel;
  final int? taxId;
  final List<Variant> variants;
  bool get isActive => status == 'active';
  List<Variant> get activeVariants =>
      variants.where((v) => v.isActive).toList();
}

typedef ProductQuery = ({String q, String status, String? type, int page});

const productTypes = {
  'software': 'Software',
  'service': 'Service',
  'addon': 'Add-on',
};

class ProductsRepository {
  ProductsRepository(this._api);
  final ApiClient _api;

  Future<Paged<Product>> list(ProductQuery q) => _api.getPaged(
    '/products',
    Product.fromJson,
    query: {
      'q': q.q,
      'status': q.status,
      'product_type': q.type,
      'page': q.page,
      'page_size': 20,
      'sort': 'name',
    },
  );

  Future<Product> get(int id) async =>
      Product.fromJson(await _api.get('/products/$id') as Map<String, dynamic>);
  Future<Product> create(Map<String, dynamic> body) async => Product.fromJson(
    await _api.post('/products', body: body) as Map<String, dynamic>,
  );
  Future<void> update(int id, Map<String, dynamic> body) =>
      _api.put('/products/$id', body: body);
  Future<void> archive(int id) => _api.delete('/products/$id');
  Future<void> addVariant(int id, Map<String, dynamic> body) =>
      _api.post('/products/$id/variants', body: body);
  Future<void> updateVariant(int id, int vid, Map<String, dynamic> body) =>
      _api.put('/products/$id/variants/$vid', body: body);
}

final productsRepositoryProvider = Provider(
  (ref) => ProductsRepository(ref.watch(apiClientProvider)),
);

final productsProvider = FutureProvider.autoDispose
    .family<Paged<Product>, ProductQuery>(
      (ref, q) => ref.watch(productsRepositoryProvider).list(q),
    );

final productDetailProvider = FutureProvider.autoDispose.family<Product, int>(
  (ref, id) => ref.watch(productsRepositoryProvider).get(id),
);

/// Active products with variants, for line pickers (plans, templates, pricing, discounts).
final activeProductsProvider = FutureProvider.autoDispose<List<Product>>((
  ref,
) async {
  final page = await ref.watch(productsRepositoryProvider).list((
    q: '',
    status: 'active',
    type: null,
    page: 1,
  ));
  return page.items;
});
