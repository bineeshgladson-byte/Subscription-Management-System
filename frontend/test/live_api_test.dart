// Opt-in: runs the app's real repositories against a running backend to prove
// the screens' parsers match the live API. Skipped unless LIVE_API is set:
//   LIVE_API=http://localhost:8080/api/v1 flutter test test/live_api_test.dart
import 'dart:io';

import 'package:brightloop/core/network/api_client.dart';
import 'package:brightloop/features/customers/data/customers_api.dart';
import 'package:brightloop/features/discounts/data/discounts_api.dart';
import 'package:brightloop/features/plans/data/plans_api.dart';
import 'package:brightloop/features/products/data/products_api.dart';
import 'package:brightloop/features/sales/data/sales_api.dart';
import 'package:brightloop/features/staff/data/staff_api.dart';
import 'package:brightloop/features/taxes/data/taxes_api.dart';
import 'package:brightloop/features/templates/data/templates_api.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final base = Platform.environment['LIVE_API'];
  test('live API matches the app\'s parsers', () async {
    final dio = Dio(
      BaseOptions(baseUrl: base!, contentType: Headers.jsonContentType),
    );
    final login = await dio.post(
      '/auth/login',
      data: {
        'email': Platform.environment['LIVE_EMAIL'] ?? 'admin@brightloop.local',
        'password':
            Platform.environment['LIVE_PASSWORD'] ?? 'Admin#Brightloop1',
      },
    );
    dio.options.headers['Authorization'] =
        'Bearer ${login.data['data']['access_token']}';
    final api = ApiClient(dio);

    final customers = await CustomersRepository(
      api,
    ).list((q: '', hasLogin: null, page: 1));
    final staff = await StaffRepository(
      api,
    ).list((q: '', active: null, page: 1));
    final taxes = await TaxesRepository(api).list();
    final products = await ProductsRepository(
      api,
    ).list((q: '', status: 'active', type: null, page: 1));
    final plans = await PlansRepository(
      api,
    ).list((q: '', status: 'active', period: null));
    final templates = await TemplatesRepository(
      api,
    ).list((q: '', active: null, page: 1));
    final discounts = await DiscountsRepository(
      api,
    ).list((q: '', state: null, page: 1));
    final seat =
        products.items.where((p) => p.variants.isNotEmpty).firstOrNull ??
        products.items.first;
    expect(seat.costPrice, isNotNull, reason: 'admin sees cost');
    expect(plans.items.first.activeSubscriptions, isNotNull);
    if (customers.items.isNotEmpty) {
      final detail = await CustomersRepository(
        api,
      ).get(customers.items.first.id);
      expect(detail.stats, isNotNull);
    }

    final preview = await DiscountsRepository(api).preview([
      {'product_id': seat.id, 'quantity': 10},
    ]);
    expect(preview.total, isNotEmpty);

    final sales = SalesRepository(api);
    final quotes = await sales.quotations((q: '', status: null, page: 1));
    final subs = await sales.subscriptions((q: '', status: null, page: 1));
    final invoices = await sales.invoices((
      q: '',
      status: null,
      overdue: false,
      page: 1,
    ));
    final payments = await sales.payments((q: '', method: null, page: 1));
    final dash = await sales.dashboard();
    if (invoices.items.isNotEmpty) await sales.invoice(invoices.items.first.id);
    if (subs.items.isNotEmpty) await sales.subscription(subs.items.first.id);
    expect(dash.mrr, isNotNull, reason: 'admin sees MRR');
    // ignore: avoid_print
    print(
      'sales: ${quotes.total} quotations, ${subs.total} subscriptions, ${invoices.total} invoices, ${payments.total} payments, MRR ${dash.mrr}',
    );
    // ignore: avoid_print
    print(
      'live: ${customers.total} customers, ${staff.total} staff, ${taxes.total} taxes, ${products.total} products, '
      '${plans.total} plans, ${templates.total} templates, ${discounts.total} discounts · '
      'Acme preview total ${preview.total} (${preview.discounts.map((d) => d.name).join(', ')})',
    );
  }, skip: base == null ? 'set LIVE_API to run against a live backend' : false);
}
