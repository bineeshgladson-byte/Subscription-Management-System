import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';

class Payment {
  Payment.fromJson(Map<String, dynamic> j)
      : id = toInt(j['id']),
        paymentNumber = str(j['payment_number']),
        invoiceId = toInt(j['invoice_id']),
        invoiceNumber = str(j['invoice_number']),
        customerId = toInt(j['customer_id']),
        customerName = str(j['customer_name']),
        method = str(j['method']),
        amount = str(j['amount'], '0'),
        paymentDate = str(j['payment_date']),
        reference = strOrNull(j['reference']),
        notes = strOrNull(j['notes']),
        status = str(j['status']),
        createdAt = dateOrNull(j['created_at']);

  final int id, invoiceId, customerId;
  final String paymentNumber, invoiceNumber, customerName, method, amount, paymentDate, status;
  final String? reference, notes;
  final DateTime? createdAt;

  bool get isCompleted => status == 'completed';
  bool get isRefunded => status == 'refunded';
}

typedef PaymentQuery = ({String? method, String? status, int? invoiceId, int? customerId, int page});

class PaymentsRepository {
  PaymentsRepository(this._api);
  final ApiClient _api;

  Future<Paged<Payment>> list(PaymentQuery q) => _api.getPaged(
        '/payments',
        Payment.fromJson,
        query: {
          'method': q.method,
          'status': q.status,
          'invoice_id': q.invoiceId,
          'customer_id': q.customerId,
          'page': q.page,
          'page_size': 20,
        },
      );

  Future<Payment> get(int id) async => Payment.fromJson(
        await _api.get('/payments/$id') as Map<String, dynamic>,
      );

  Future<void> refund(int id) => _api.post('/payments/$id/refund');
}

final paymentsRepositoryProvider = Provider(
  (ref) => PaymentsRepository(ref.watch(apiClientProvider)),
);

final paymentsProvider =
    FutureProvider.autoDispose.family<Paged<Payment>, PaymentQuery>(
  (ref, q) => ref.watch(paymentsRepositoryProvider).list(q),
);
