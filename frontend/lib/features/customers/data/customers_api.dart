import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';

class Customer {
  Customer.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      name = str(j['name']),
      email = str(j['email']),
      phone = strOrNull(j['phone']),
      companyName = strOrNull(j['company_name']),
      billingAddress = strOrNull(j['billing_address']),
      gstin = strOrNull(j['gstin']),
      hasLogin = toBool(j['has_login']),
      createdAt = dateOrNull(j['created_at']),
      stats = j['stats'] is Map
          ? CustomerStats.fromJson(j['stats'] as Map<String, dynamic>)
          : null;
  final int id;
  final String name, email;
  final String? phone, companyName, billingAddress, gstin;
  final bool hasLogin;
  final DateTime? createdAt;
  final CustomerStats? stats;
}

class CustomerStats {
  CustomerStats.fromJson(Map<String, dynamic> j)
    : activeSubscriptions = toInt(j['active_subscriptions']),
      totalSubscriptions = toInt(j['total_subscriptions']),
      openInvoices = toInt(j['open_invoices']),
      overdueInvoices = toInt(j['overdue_invoices']),
      amountDue = str(j['amount_due'], '0');
  final int activeSubscriptions,
      totalSubscriptions,
      openInvoices,
      overdueInvoices;
  final String amountDue;
}

typedef CustomerQuery = ({String q, bool? hasLogin, int page});

class CustomersRepository {
  CustomersRepository(this._api);
  final ApiClient _api;

  Future<Paged<Customer>> list(CustomerQuery q) => _api.getPaged(
    '/customers',
    Customer.fromJson,
    query: {
      'q': q.q,
      'has_login': q.hasLogin?.toString(),
      'page': q.page,
      'page_size': 20,
      'sort': '-created_at',
    },
  );

  Future<Customer> get(int id) async => Customer.fromJson(
    await _api.get('/customers/$id') as Map<String, dynamic>,
  );

  Future<void> create(Map<String, dynamic> body) =>
      _api.post('/customers', body: body);
  Future<void> update(int id, Map<String, dynamic> body) =>
      _api.put('/customers/$id', body: body);
  Future<void> archive(int id) => _api.delete('/customers/$id');
}

final customersRepositoryProvider = Provider(
  (ref) => CustomersRepository(ref.watch(apiClientProvider)),
);

final customersProvider = FutureProvider.autoDispose
    .family<Paged<Customer>, CustomerQuery>(
      (ref, q) => ref.watch(customersRepositoryProvider).list(q),
    );

final customerDetailProvider = FutureProvider.autoDispose.family<Customer, int>(
  (ref, id) => ref.watch(customersRepositoryProvider).get(id),
);
