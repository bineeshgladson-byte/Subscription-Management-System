import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';
import '../../../core/network/navigation_cache.dart';

/// Quotations, subscriptions, invoices, payments, dashboard summary and billing run.

class SubItem {
  SubItem.fromJson(Map<String, dynamic> j)
    : description = str(j['description']),
      quantity = toInt(j['quantity'], 1),
      unitPrice = str(j['unit_price'], '0'),
      taxRate = str(j['tax_rate'], '0'),
      lineSubtotal = str(j['line_subtotal'], '0');
  final String description, unitPrice, taxRate, lineSubtotal;
  final int quantity;
}

class InvoiceBrief {
  InvoiceBrief.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      number = str(j['invoice_number']),
      periodStart = strOrNull(j['period_start']),
      periodEnd = strOrNull(j['period_end']),
      invoiceDate = str(j['invoice_date']),
      dueDate = str(j['due_date']),
      status = str(j['status']),
      paymentStatus = str(j['payment_status']),
      isOverdue = toBool(j['is_overdue']),
      total = str(j['total'], '0'),
      amountDue = str(j['amount_due'], '0');
  final int id;
  final String number,
      invoiceDate,
      dueDate,
      status,
      paymentStatus,
      total,
      amountDue;
  final String? periodStart, periodEnd;
  final bool isOverdue;
}

class Subscription {
  Subscription.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      quotationNumber = str(j['quotation_number']),
      subscriptionNumber = strOrNull(j['subscription_number']),
      customerId = toInt(j['customer_id']),
      customerName = str(j['customer_name']),
      planName = str(j['plan_name']),
      billingLabel = str(j['billing_label']),
      isPausable = toBool(j['is_pausable']),
      isClosable = toBool(j['is_closable']),
      isRenewable = toBool(j['is_renewable']),
      status = str(j['status']),
      startDate = str(j['start_date']),
      expiryDate = strOrNull(j['expiry_date']),
      validUntil = strOrNull(j['quotation_valid_until']),
      nextBillingDate = strOrNull(j['next_billing_date']),
      paymentTermsDays = toInt(j['payment_terms_days']),
      cyclesBilled = toInt(j['cycles_billed']),
      closeReason = strOrNull(j['close_reason']),
      subtotal = str(j['subtotal'], '0'),
      discountTotal = str(j['discount_total'], '0'),
      taxTotal = str(j['tax_total'], '0'),
      total = str(j['total'], '0'),
      items = listOf(j['items']).map(SubItem.fromJson).toList(),
      invoices = listOf(j['invoices']).map(InvoiceBrief.fromJson).toList();
  final int id, customerId, paymentTermsDays, cyclesBilled;
  final String quotationNumber,
      customerName,
      planName,
      billingLabel,
      status,
      startDate,
      subtotal,
      discountTotal,
      taxTotal,
      total;
  final String? subscriptionNumber,
      expiryDate,
      validUntil,
      nextBillingDate,
      closeReason;
  final bool isPausable, isClosable, isRenewable;
  final List<SubItem> items;
  final List<InvoiceBrief> invoices;
  String get number => subscriptionNumber ?? quotationNumber;
}

class InvoiceLine {
  InvoiceLine.fromJson(Map<String, dynamic> j)
    : description = str(j['description']),
      quantity = toInt(j['quantity'], 1),
      unitPrice = str(j['unit_price'], '0'),
      lineSubtotal = str(j['line_subtotal'], '0'),
      discountAmount = str(j['discount_amount'], '0'),
      taxRate = str(j['tax_rate'], '0'),
      taxAmount = str(j['tax_amount'], '0'),
      lineTotal = str(j['line_total'], '0');
  final String description,
      unitPrice,
      lineSubtotal,
      discountAmount,
      taxRate,
      taxAmount,
      lineTotal;
  final int quantity;
}

class PaymentRow {
  PaymentRow.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      number = str(j['payment_number']),
      method = str(j['method']),
      amount = str(j['amount'], '0'),
      paymentDate = str(j['payment_date']),
      status = str(j['status']),
      reference = strOrNull(j['reference']),
      notes = strOrNull(j['notes']),
      invoiceId = toInt(j['invoice_id']),
      invoiceNumber = strOrNull(j['invoice_number']),
      customerName = strOrNull(j['customer_name']);
  final int id, invoiceId;
  final String number, method, amount, paymentDate, status;
  final String? reference, notes, invoiceNumber, customerName;
}

class Invoice extends InvoiceBrief {
  Invoice.fromJson(super.j)
    : customerName = str(j['customer_name']),
      customerEmail = strOrNull(j['customer_email']),
      billingAddress = strOrNull(j['billing_address']),
      gstin = strOrNull(j['customer_gstin']),
      subscriptionNumber = strOrNull(j['subscription_number']),
      daysOverdue = toInt(j['days_overdue']),
      subtotal = str(j['subtotal'], '0'),
      discountTotal = str(j['discount_total'], '0'),
      taxTotal = str(j['tax_total'], '0'),
      amountPaid = str(j['amount_paid'], '0'),
      sentAt = dateOrNull(j['sent_at']),
      lines = listOf(j['items']).map(InvoiceLine.fromJson).toList(),
      payments = listOf(j['payments']).map(PaymentRow.fromJson).toList(),
      super.fromJson();
  final String customerName, subtotal, discountTotal, taxTotal, amountPaid;
  final String? customerEmail, billingAddress, gstin, subscriptionNumber;
  final int daysOverdue;
  final DateTime? sentAt;
  final List<InvoiceLine> lines;
  final List<PaymentRow> payments;
}

class DashboardSummary {
  DashboardSummary.fromJson(Map<String, dynamic> j)
    : totalCustomers = toInt(j['total_customers']),
      activeSubscriptions = toInt(j['active_subscriptions']),
      totalSubscriptions = toInt(j['total_subscriptions']),
      openInvoices = toInt(j['open_invoices']),
      overdueInvoices = toInt(j['overdue_invoices']),
      byStatus = {
        for (final e in ((j['subscriptions_by_status'] as Map?) ?? {}).entries)
          '${e.key}': toInt(e.value),
      },
      mrr = strOrNull(j['mrr']),
      arr = strOrNull(j['arr']),
      revenueReceived = strOrNull(j['revenue_received']),
      outstanding = strOrNull(j['outstanding']),
      overdueAmount = strOrNull(j['overdue_amount']),
      recentPayments = listOf(
        j['recent_payments'],
      ).map(PaymentRow.fromJson).toList(),
      from = str(j['from']),
      to = str(j['to']);
  final int totalCustomers,
      activeSubscriptions,
      totalSubscriptions,
      openInvoices,
      overdueInvoices;
  final Map<String, int> byStatus;
  final String? mrr, arr, revenueReceived, outstanding, overdueAmount;
  final List<PaymentRow> recentPayments;
  final String from, to;
}

typedef SubQuery = ({String q, String? status, int page});
typedef InvoiceQuery = ({String q, String? status, bool overdue, int page});
typedef PaymentQuery = ({String q, String? method, int page});

const subStatuses = {
  'quotation': 'Quotation',
  'confirmed': 'Confirmed',
  'active': 'Active',
  'paused': 'Paused',
  'closed': 'Closed',
  'renewed': 'Renewed',
};
const invoiceStatuses = {
  'draft': 'Draft',
  'confirmed': 'Confirmed',
  'paid': 'Paid',
  'cancelled': 'Cancelled',
};
const paymentMethods = {
  'upi': 'UPI',
  'bank_transfer': 'Bank transfer',
  'card': 'Card',
  'cash': 'Cash',
};

class SalesRepository {
  SalesRepository(this._api);
  final ApiClient _api;

  Future<Paged<Subscription>> quotations(SubQuery q) => _api.getPaged(
    '/quotations',
    Subscription.fromJson,
    query: {'q': q.q, 'status': q.status, 'page': q.page, 'page_size': 20},
  );
  Future<Paged<Subscription>> subscriptions(SubQuery q) => _api.getPaged(
    '/subscriptions',
    Subscription.fromJson,
    query: {'q': q.q, 'status': q.status, 'page': q.page, 'page_size': 20},
  );
  Future<Subscription> subscription(int id) async => Subscription.fromJson(
    await _api.get('/subscriptions/$id') as Map<String, dynamic>,
  );
  Future<Subscription> createQuotation(Map<String, dynamic> body) async =>
      Subscription.fromJson(
        await _api.post('/quotations', body: body) as Map<String, dynamic>,
      );
  Future<void> quotationAction(int id, String action) =>
      _api.post('/quotations/$id/$action');
  Future<void> subscriptionAction(int id, String action) =>
      _api.post('/subscriptions/$id/$action');

  Future<Paged<Invoice>> invoices(InvoiceQuery q) => _api.getPaged(
    '/invoices',
    Invoice.fromJson,
    query: {
      'q': q.q,
      'status': q.status,
      'overdue': q.overdue ? 'true' : null,
      'page': q.page,
      'page_size': 20,
    },
  );
  Future<Invoice> invoice(int id) async =>
      Invoice.fromJson(await _api.get('/invoices/$id') as Map<String, dynamic>);
  Future<void> invoiceAction(int id, String action) =>
      _api.post('/invoices/$id/$action');
  Future<void> recordPayment(
    int invoiceId,
    Map<String, dynamic> body,
    String key,
  ) => _api.post(
    '/invoices/$invoiceId/payments',
    body: body,
    headers: {'Idempotency-Key': key},
  );

  Future<Paged<PaymentRow>> payments(PaymentQuery q) => _api.getPaged(
    '/payments',
    PaymentRow.fromJson,
    query: {'q': q.q, 'method': q.method, 'page': q.page, 'page_size': 20},
  );

  // ---- customer portal (/me) ----
  Future<Paged<Subscription>> mySubscriptions(int page) => _api.getPaged(
    '/me/subscriptions',
    Subscription.fromJson,
    query: {'page': page, 'page_size': 20},
  );
  Future<Subscription> mySubscription(int id) async => Subscription.fromJson(
    await _api.get('/me/subscriptions/$id') as Map<String, dynamic>,
  );
  Future<void> myAction(int id, String action) =>
      _api.post('/me/subscriptions/$id/$action');
  Future<Subscription> requestMyQuotation(Map<String, dynamic> body) async =>
      Subscription.fromJson(
        await _api.post('/me/quotations', body: body) as Map<String, dynamic>,
      );
  Future<Paged<Invoice>> myInvoices(int page) => _api.getPaged(
    '/me/invoices',
    Invoice.fromJson,
    query: {'page': page, 'page_size': 20},
  );
  Future<Invoice> myInvoice(int id) async => Invoice.fromJson(
    await _api.get('/me/invoices/$id') as Map<String, dynamic>,
  );
  Future<Paged<PaymentRow>> myPayments(int page) => _api.getPaged(
    '/me/payments',
    PaymentRow.fromJson,
    query: {'page': page, 'page_size': 20},
  );

  Future<DashboardSummary> dashboard({String? from, String? to}) async =>
      DashboardSummary.fromJson(
        await _api.get('/dashboard/summary', query: {'from': from, 'to': to})
            as Map<String, dynamic>,
      );

  Future<Map<String, dynamic>> runBilling({String? asOf}) async =>
      await _api.post('/admin/billing/run', body: {'as_of': ?asOf})
          as Map<String, dynamic>;
}

final salesRepositoryProvider = Provider(
  (ref) => SalesRepository(ref.watch(apiClientProvider)),
);

final quotationsProvider = FutureProvider.autoDispose
    .family<Paged<Subscription>, SubQuery>(
      (ref, q) => ref.watch(salesRepositoryProvider).quotations(q),
    );
final subscriptionsProvider = FutureProvider.autoDispose
    .family<Paged<Subscription>, SubQuery>(
      (ref, q) => ref.watch(salesRepositoryProvider).subscriptions(q),
    );
final subscriptionDetailProvider = FutureProvider.autoDispose
    .family<Subscription, int>(
      (ref, id) => ref.watch(salesRepositoryProvider).subscription(id),
    );
final invoicesProvider = FutureProvider.autoDispose
    .family<Paged<Invoice>, InvoiceQuery>(
      (ref, q) => ref.watch(salesRepositoryProvider).invoices(q),
    );
final invoiceDetailProvider = FutureProvider.autoDispose.family<Invoice, int>(
  (ref, id) => ref.watch(salesRepositoryProvider).invoice(id),
);
final paymentsProvider = FutureProvider.autoDispose
    .family<Paged<PaymentRow>, PaymentQuery>(
      (ref, q) => ref.watch(salesRepositoryProvider).payments(q),
    );
final dashboardSummaryProvider = FutureProvider.autoDispose
    .family<DashboardSummary, ({String? from, String? to})>(
      (ref, r) =>
          ref.watch(salesRepositoryProvider).dashboard(from: r.from, to: r.to),
    );

final mySubscriptionsProvider = FutureProvider.autoDispose
    .family<Paged<Subscription>, int>(
      (ref, p) => cacheForNavigation(
        ref,
        () => ref.watch(salesRepositoryProvider).mySubscriptions(p),
      ),
    );
final mySubscriptionProvider = FutureProvider.autoDispose
    .family<Subscription, int>(
      (ref, id) => cacheForNavigation(
        ref,
        () => ref.watch(salesRepositoryProvider).mySubscription(id),
      ),
    );
final myInvoicesProvider = FutureProvider.autoDispose
    .family<Paged<Invoice>, int>(
      (ref, p) => cacheForNavigation(
        ref,
        () => ref.watch(salesRepositoryProvider).myInvoices(p),
      ),
    );
final myInvoiceProvider = FutureProvider.autoDispose.family<Invoice, int>(
  (ref, id) => cacheForNavigation(
    ref,
    () => ref.watch(salesRepositoryProvider).myInvoice(id),
  ),
);
final myPaymentsProvider = FutureProvider.autoDispose
    .family<Paged<PaymentRow>, int>(
      (ref, p) => cacheForNavigation(
        ref,
        () => ref.watch(salesRepositoryProvider).myPayments(p),
      ),
    );

/// Refreshes everything that billing / payments can change.
void invalidateSales(WidgetRef ref) {
  ref.invalidate(quotationsProvider);
  ref.invalidate(subscriptionsProvider);
  ref.invalidate(subscriptionDetailProvider);
  ref.invalidate(invoicesProvider);
  ref.invalidate(invoiceDetailProvider);
  ref.invalidate(paymentsProvider);
  ref.invalidate(dashboardSummaryProvider);
}
