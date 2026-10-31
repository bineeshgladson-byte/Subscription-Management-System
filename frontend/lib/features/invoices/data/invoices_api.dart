import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';

class InvoiceItem {
  InvoiceItem.fromJson(Map<String, dynamic> j)
      : id = toInt(j['id']),
        productId = intOrNull(j['product_id']),
        productName = str(j['product_name']),
        variantId = intOrNull(j['variant_id']),
        variantName = strOrNull(j['variant_name']),
        description = str(j['description']),
        quantity = toInt(j['quantity'], 1),
        unitPrice = str(j['unit_price'], '0'),
        lineSubtotal = str(j['line_subtotal'], '0'),
        discountAmount = str(j['discount_amount'], '0'),
        taxRate = str(j['tax_rate'], '0'),
        taxAmount = str(j['tax_amount'], '0'),
        lineTotal = str(j['line_total'], '0');

  final int id, quantity;
  final int? productId, variantId;
  final String productName,
      description,
      unitPrice,
      lineSubtotal,
      discountAmount,
      taxRate,
      taxAmount,
      lineTotal;
  final String? variantName;
}

class Invoice {
  Invoice.fromJson(Map<String, dynamic> j)
      : id = toInt(j['id']),
        invoiceNumber = str(j['invoice_number']),
        customerId = toInt(j['customer_id']),
        customerName = str(j['customer_name']),
        billingAddress = strOrNull(j['billing_address']),
        customerGstin = strOrNull(j['customer_gstin']),
        subscriptionId = intOrNull(j['subscription_id']),
        subscriptionNumber = strOrNull(j['subscription_number']),
        periodStart = strOrNull(j['period_start']),
        periodEnd = strOrNull(j['period_end']),
        invoiceDate = str(j['invoice_date']),
        dueDate = str(j['due_date']),
        status = str(j['status']),
        subtotal = str(j['subtotal'], '0'),
        discountTotal = str(j['discount_total'], '0'),
        taxTotal = str(j['tax_total'], '0'),
        total = str(j['total'], '0'),
        amountPaid = str(j['amount_paid'], '0'),
        amountDue = str(j['amount_due'], '0'),
        isOverdue = toBool(j['is_overdue']),
        sentAt = dateOrNull(j['sent_at']),
        confirmedAt = dateOrNull(j['confirmed_at']),
        cancelledAt = dateOrNull(j['cancelled_at']),
        cancelReason = strOrNull(j['cancel_reason']),
        items = listOf(j['items']).map(InvoiceItem.fromJson).toList(),
        createdAt = dateOrNull(j['created_at']);

  final int id, customerId;
  final int? subscriptionId;
  final String invoiceNumber,
      customerName,
      invoiceDate,
      dueDate,
      status,
      subtotal,
      discountTotal,
      taxTotal,
      total,
      amountPaid,
      amountDue;
  final String? billingAddress,
      customerGstin,
      subscriptionNumber,
      periodStart,
      periodEnd,
      cancelReason;
  final bool isOverdue;
  final DateTime? sentAt, confirmedAt, cancelledAt, createdAt;
  final List<InvoiceItem> items;

  bool get isDraft => status == 'draft';
  bool get isConfirmed => status == 'confirmed';
  bool get isPaid => status == 'paid';
  bool get isCancelled => status == 'cancelled';
}

class RazorpayOrder {
  RazorpayOrder.fromJson(Map<String, dynamic> j)
      : orderId = str(j['order_id']),
        amount = toInt(j['amount']),
        currency = str(j['currency'], 'INR'),
        keyId = str(j['key_id']),
        invoiceNumber = str(j['invoice_number']),
        customerName = str(j['customer_name']),
        customerEmail = str(j['customer_email']);

  final String orderId, currency, keyId, invoiceNumber, customerName, customerEmail;
  final int amount; // in paise
}

typedef InvoiceQuery = ({String q, String? status, bool? overdue, int? customerId, int page});

class InvoicesRepository {
  InvoicesRepository(this._api);
  final ApiClient _api;

  Future<Paged<Invoice>> list(InvoiceQuery q) => _api.getPaged(
        '/invoices',
        Invoice.fromJson,
        query: {
          'q': q.q,
          'status': q.status,
          'overdue': q.overdue?.toString(),
          'customer_id': q.customerId,
          'page': q.page,
          'page_size': 20,
        },
      );

  Future<Invoice> get(int id) async => Invoice.fromJson(
        await _api.get('/invoices/$id') as Map<String, dynamic>,
      );

  Future<void> confirm(int id) => _api.post('/invoices/$id/confirm');
  Future<void> cancel(int id, {String? reason}) =>
      _api.post('/invoices/$id/cancel', body: {'reason': reason ?? ''});
  Future<void> send(int id) => _api.post('/invoices/$id/send');

  Future<void> recordPayment(int id, Map<String, dynamic> body) =>
      _api.post('/invoices/$id/payments', body: body);

  Future<RazorpayOrder> createRazorpayOrder(int id) async =>
      RazorpayOrder.fromJson(
        await _api.post('/invoices/$id/razorpay/order') as Map<String, dynamic>,
      );

  Future<void> verifyRazorpayPayment(int id, Map<String, dynamic> body) =>
      _api.post('/invoices/$id/razorpay/verify', body: body);
}

final invoicesRepositoryProvider = Provider(
  (ref) => InvoicesRepository(ref.watch(apiClientProvider)),
);

final invoicesProvider =
    FutureProvider.autoDispose.family<Paged<Invoice>, InvoiceQuery>(
  (ref, q) => ref.watch(invoicesRepositoryProvider).list(q),
);
