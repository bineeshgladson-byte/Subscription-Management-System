import 'dart:io';
import 'dart:ui' as ui;

import 'package:brightloop/core/models/role.dart';
import 'package:brightloop/core/network/api_client.dart';
import 'package:brightloop/core/router/app_router.dart';
import 'package:brightloop/core/theme/workspace_theme.dart';
import 'package:brightloop/core/widgets/app_shell.dart';
import 'package:brightloop/core/widgets/customer_widgets.dart';
import 'package:brightloop/features/auth/application/auth_controller.dart';
import 'package:brightloop/features/auth/data/auth_user.dart';
import 'package:brightloop/features/dashboard/presentation/dashboard_screen.dart';
import 'package:brightloop/features/plans/presentation/plans_screen.dart';
import 'package:brightloop/features/plans/data/plans_api.dart';
import 'package:brightloop/features/products/presentation/products_screen.dart';
import 'package:brightloop/features/portal/presentation/portal_screens.dart';
import 'package:brightloop/features/profile/presentation/profile_screen.dart';
import 'package:brightloop/features/sales/data/sales_api.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

/// Returns API-shaped JSON (as the Go backend sends it) instead of calling the
/// network, so screens are tested with their real parsing code.
class FakeApi extends ApiClient {
  FakeApi(this.role, {this.portalRecords = true, this.extraProduct = false})
    : super(Dio());
  final Role role;
  final bool portalRecords;
  final bool extraProduct;
  final requests = <String, int>{};
  final posted = <String, Object?>{};

  static Map<String, dynamic> plan(
    int id,
    String name,
    String price,
    String label, {
    int subs = 0,
  }) => {
    'id': id,
    'name': name,
    'description': 'Live plan $id',
    'price': price,
    'billing_period': 'monthly',
    'billing_interval': 1,
    'billing_label': label,
    'min_quantity': 1,
    'available_from': '2026-01-01',
    'available_until': null,
    'is_available_now': true,
    'auto_close_enabled': false,
    'auto_close_after_cycles': null,
    'is_pausable': true,
    'is_closable': true,
    'is_renewable': true,
    'status': 'active',
    'products': [
      {
        'id': 1,
        'product_id': 7,
        'product_name': 'Software Subscription',
        'variant_id': 12,
        'variant_name': 'Odoo Brand',
        'quantity': 1,
        'price_override': null,
        'unit_price': price,
        'line_total': price,
      },
    ],
    if (true) 'active_subscriptions': subs,
  };

  static Map<String, dynamic> product(bool admin) => {
    'id': 7,
    'name': 'Software Subscription',
    'product_type': 'software',
    'sales_price': '999.00',
    'cost_price': admin ? '300.00' : null,
    'margin': admin ? '699.00' : null,
    'description': 'Cloud ERP seat',
    'tax': {'id': 1, 'name': 'GST 18%', 'rate': '18.00'},
    'status': 'active',
    'variants': [
      {
        'id': 11,
        'attribute': 'Brand',
        'name': 'Brand A',
        'extra_price': '0.00',
        'price': '999.00',
        'sku': null,
        'is_active': true,
      },
      {
        'id': 12,
        'attribute': 'Brand',
        'name': 'Odoo Brand',
        'extra_price': '560.00',
        'price': '1559.00',
        'sku': null,
        'is_active': true,
      },
    ],
  };

  List<Map<String, dynamic>> _list(String path) => switch (path) {
    '/me/subscriptions' => portalRecords ? [subscription()] : [],
    '/me/invoices' => portalRecords ? [invoice()] : [],
    '/me/payments' => portalRecords ? [payment()] : [],
    '/plans' => [
      () {
        final p = plan(1, 'Starter Monthly', '999.00', 'Monthly', subs: 3);
        if (role == Role.customer) p.remove('active_subscriptions');
        return p;
      }(),
      () {
        final p = plan(2, 'Business Yearly', '1559.00', 'Yearly', subs: 2);
        if (role == Role.customer) p.remove('active_subscriptions');
        return p;
      }(),
    ],
    '/products' => [
      product(role == Role.admin),
      if (extraProduct)
        {
          ...product(role == Role.admin),
          'id': 8,
          'name': 'Support',
          'variants': <Map<String, dynamic>>[],
        },
    ],
    '/customers' => [
      {
        'id': 1,
        'name': 'Acme Pvt Ltd',
        'email': 'accounts@acme.in',
        'phone': null,
        'company_name': 'Acme Pvt Ltd',
        'billing_address': null,
        'gstin': null,
        'has_login': true,
        'created_at': '2026-10-31T10:00:00Z',
      },
    ],
    '/discounts' => const [],
    '/staff' => const [],
    '/taxes' => [
      {
        'id': 1,
        'name': 'GST 18%',
        'rate': '18.00',
        'is_active': true,
        'products_using': 1,
      },
    ],
    _ => const [],
  };

  @override
  Future<Paged<T>> getPaged<T>(
    String path,
    T Function(Map<String, dynamic>) fromJson, {
    Map<String, dynamic>? query,
  }) async {
    requests.update(path, (count) => count + 1, ifAbsent: () => 1);
    var rows = _list(path);
    if (path == '/plans' && query?['product_id'] != null) {
      rows = rows
          .where(
            (p) => (p['products'] as List).any(
              (line) => line['product_id'] == query!['product_id'],
            ),
          )
          .toList();
    }
    final items = rows.map(fromJson).toList();
    return Paged(
      items: items,
      page: 1,
      pageSize: 20,
      total: path == '/customers' ? 12 : items.length,
    );
  }

  @override
  Future<dynamic> get(String path, {Map<String, dynamic>? query}) async =>
      switch (path) {
        '/auth/me' => {
          'id': 1,
          'email': 'thomas@brightloop.in',
          'full_name': 'Thomas Wills',
          'role': role.name,
          'profile': {
            'phone': '9876543210',
            'company_name': 'Acme Pvt Ltd',
            'billing_address': 'Kochi, Kerala',
            'gstin': null,
          },
        },
        '/me/subscriptions/1' => subscription(),
        '/me/invoices/1' => invoice(),
        '/dashboard/summary' => <String, dynamic>{
          'from': '2026-10-01',
          'to': '2026-10-31',
        },
        _ =>
          path.startsWith('/products/')
              ? product(role == Role.admin)
              : <String, dynamic>{},
      };

  static Map<String, dynamic> subscription() => {
    'id': 1,
    'quotation_number': 'QUO-001',
    'subscription_number': 'SUB-001',
    'plan_name': 'Starter Monthly',
    'billing_label': 'Monthly',
    'status': 'active',
    'total': '1178.82',
    'start_date': '2026-10-01',
    'next_billing_date': '2026-11-01',
    'is_pausable': true,
    'is_closable': true,
    'is_renewable': true,
    'items': <Map<String, dynamic>>[],
    'invoices': <Map<String, dynamic>>[],
  };
  static Map<String, dynamic> invoice() => {
    'id': 1,
    'invoice_number': 'INV-001',
    'invoice_date': '2026-10-01',
    'due_date': '2026-10-15',
    'status': 'confirmed',
    'payment_status': 'partial',
    'total': '1178.82',
    'amount_due': '178.82',
    'amount_paid': '1000.00',
    'subtotal': '999.00',
    'tax_total': '179.82',
    'items': <Map<String, dynamic>>[],
  };
  static Map<String, dynamic> payment() => {
    'id': 1,
    'payment_number': 'PAY-001',
    'method': 'upi',
    'amount': '1000.00',
    'payment_date': '2026-10-10',
    'status': 'completed',
    'invoice_id': 1,
    'invoice_number': 'INV-001',
    'reference': 'UPI-REFERENCE-001',
  };

  @override
  Future<dynamic> post(
    String path, {
    Object? body,
    Map<String, String>? headers,
  }) async {
    posted[path] = body;
    if (path == '/products') return product(role == Role.admin);
    return {};
  }
}

class _PreviewAuth extends AuthController {
  _PreviewAuth(this.role);
  final Role role;
  @override
  Future<AuthUser?> build() async => AuthUser(
    id: 1,
    email: 'thomas@brightloop.in',
    fullName: 'Thomas Wills',
    role: role,
  );
  void switchAccount() => state = AsyncData(
    AuthUser(
      id: 2,
      email: 'second@brightloop.in',
      fullName: 'Second Customer',
      role: Role.customer,
    ),
  );
}

void main() {
  setUpAll(() async {
    final loader = FontLoader('Poppins');
    for (final file in ['Regular', 'Medium', 'SemiBold']) {
      loader.addFont(rootBundle.load('assets/fonts/Poppins-$file.ttf'));
    }
    await loader.load();
    final icons = FontLoader('MaterialIcons')
      ..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'));
    await icons.load();
  });

  Future<FakeApi> mount(
    WidgetTester tester, {
    Size size = const Size(1600, 1199),
    Role role = Role.admin,
    String path = '/plans',
    double textScale = 1,
    List<Plan>? dashboardPlans,
    bool appRoutes = false,
    bool portalRecords = true,
    bool extraProduct = false,
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final api = FakeApi(
      role,
      portalRecords: portalRecords,
      extraProduct: extraProduct,
    );
    final router = GoRouter(
      initialLocation: path,
      routes: [
        ShellRoute(
          builder: (_, _, child) => AppShell(child: child),
          routes: [
            GoRoute(path: '/plans', builder: (_, _) => const PlansScreen()),
            GoRoute(
              path: '/dashboard',
              builder: (_, _) => const DashboardScreen(),
            ),
            GoRoute(
              path: '/products',
              builder: (_, _) => const ProductsScreen(),
            ),
            GoRoute(path: '/profile', builder: (_, _) => const ProfileScreen()),
            GoRoute(
              path: '/my-subscriptions',
              builder: (_, _) => const MySubscriptionsScreen(),
            ),
            GoRoute(
              path: '/my-invoices',
              builder: (_, _) => const MyInvoicesScreen(),
            ),
            GoRoute(
              path: '/my-payments',
              builder: (_, _) => const MyPaymentsScreen(),
            ),
          ],
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        retry: (_, _) => null,
        overrides: [
          authControllerProvider.overrideWith(() => _PreviewAuth(role)),
          apiClientProvider.overrideWithValue(api),
          if (dashboardPlans != null)
            customerDashboardProvider.overrideWith((_) async => dashboardPlans),
        ],
        child: Consumer(
          builder: (context, ref, _) => MaterialApp.router(
            theme: WorkspaceTheme.data,
            routerConfig: appRoutes ? ref.watch(routerProvider) : router,
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(context).copyWith(
                textScaler: TextScaler.linear(textScale),
                disableAnimations: true,
              ),
              child: child!,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    return api;
  }

  Future<void> capture(WidgetTester tester, String name) async {
    final directory = Platform.environment['SUBLY_UI_CAPTURE'];
    if (directory == null) return;
    final boundary = tester.renderObject<RenderRepaintBoundary>(
      find.byType(RepaintBoundary).first,
    );
    await tester.runAsync(() async {
      final image = await boundary.toImage();
      final bytes = await image.toByteData(format: ui.ImageByteFormat.png);
      await Directory(directory).create(recursive: true);
      await File(
        '$directory/$name.png',
      ).writeAsBytes(bytes!.buffer.asUint8List());
      image.dispose();
    });
  }

  testWidgets(
    'workspace routes switch immediately and reuse loaded customer data',
    (tester) async {
      final api = await mount(
        tester,
        role: Role.customer,
        path: '/dashboard',
        appRoutes: true,
      );
      expect(find.text('Overview'), findsOneWidget);
      await tester.tap(find.text('Plans').first);
      await tester.pump();
      expect(find.text('Browse products'), findsOneWidget);
      expect(find.text('Overview'), findsNothing);
      for (final navigator in tester.widgetList<Navigator>(
        find.byType(Navigator),
      )) {
        for (final page in navigator.pages) {
          expect(page, isA<NoTransitionPage<void>>());
        }
      }
      await tester.pumpAndSettle();
      expect(api.requests['/plans'], 1);
      await tester.tap(find.text('My subscriptions').first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('My invoices').first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('My subscriptions').first);
      await tester.pumpAndSettle();
      expect(api.requests['/me/subscriptions'], 1);
      await tester.tap(find.text('Refresh'));
      await tester.pumpAndSettle();
      expect(api.requests['/me/subscriptions'], 2);
      expect(tester.takeException(), isNull);
    },
  );

  test('navigation cache resets when the signed-in account changes', () async {
    final api = FakeApi(Role.customer);
    final container = ProviderContainer.test(
      overrides: [
        apiClientProvider.overrideWithValue(api),
        authControllerProvider.overrideWith(() => _PreviewAuth(Role.customer)),
      ],
    );
    await container.read(authControllerProvider.future);
    final subscription = container.listen(
      mySubscriptionsProvider(1),
      (_, _) {},
    );
    await container.read(mySubscriptionsProvider(1).future);
    subscription.close();
    await container.pump();
    await container.read(mySubscriptionsProvider(1).future);
    expect(api.requests['/me/subscriptions'], 1);
    (container.read(authControllerProvider.notifier) as _PreviewAuth)
        .switchAccount();
    await container.pump();
    await container.read(mySubscriptionsProvider(1).future);
    expect(api.requests['/me/subscriptions'], 2);
  });

  for (final (path, name) in [
    ('/my-subscriptions', 'subscriptions'),
    ('/my-invoices', 'invoices'),
    ('/my-payments', 'payments'),
    ('/profile', 'profile'),
  ]) {
    for (final (size, scale, layout) in [
      (const Size(1600, 1000), 1.0, 'desktop'),
      (const Size(375, 812), 1.0, 'phone'),
      (const Size(375, 812), 2.0, 'large-text'),
    ]) {
      testWidgets('customer $name is readable on $layout', (tester) async {
        await mount(
          tester,
          role: Role.customer,
          path: path,
          size: size,
          textScale: scale,
        );
        expect(tester.takeException(), isNull);
        await capture(tester, 'customer-$name-$layout');
      });
    }
    if (path != '/profile') {
      testWidgets('customer $name has an empty state', (tester) async {
        await mount(
          tester,
          role: Role.customer,
          path: path,
          portalRecords: false,
        );
        expect(find.byType(CustomerEmpty), findsOneWidget);
        expect(tester.takeException(), isNull);
        await capture(tester, 'customer-$name-empty');
      });
    }
  }

  testWidgets('customer subscriptions and invoices still open details', (
    tester,
  ) async {
    final api = await mount(
      tester,
      role: Role.customer,
      path: '/my-subscriptions',
    );
    await tester.tap(find.text('View subscription'));
    await tester.pumpAndSettle();
    expect(find.text('Pause'), findsOneWidget);
    await tester.tap(find.text('Pause'));
    await tester.pumpAndSettle();
    expect(api.posted.containsKey('/me/subscriptions/1/pause'), isTrue);
    await tester.tap(find.text('Done'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('My invoices').first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View invoice'));
    await tester.pumpAndSettle();
    expect(find.text('Amount due'), findsWidgets);
    expect(tester.takeException(), isNull);
  });

  testWidgets('admin sees live plans with prices, subscribers and management', (
    tester,
  ) async {
    await mount(tester);
    expect(find.text('Software Subscription'), findsOneWidget);
    await tester.tap(find.text('Manage plans').first);
    await tester.pumpAndSettle();
    expect(find.text('Starter Monthly'), findsOneWidget);
    expect(find.text('Business Yearly'), findsOneWidget);
    // Price is now rendered inside a Text.rich with a "/mo" suffix, so match
    // the money substring instead of the full exact string.
    expect(find.textContaining('₹999.00'), findsWidgets);
    expect(find.textContaining('₹1,559.00'), findsWidgets);
    expect(find.text('Create Plan'), findsOneWidget);
    // Admin metrics tile shows the sum (3 + 2) and the per-card footer shows
    // each plan's own count.
    expect(find.text('Active subscribers'), findsOneWidget);
    expect(find.text('5'), findsOneWidget);
    expect(find.byTooltip('Manage Starter Monthly'), findsOneWidget);
    expect(tester.takeException(), isNull);
    await capture(tester, 'plans-desktop');
    // Create dialog validates before calling the API.
    await tester.tap(find.text('Create Plan'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(find.text('Required'), findsWidgets);
    expect(tester.takeException(), isNull);
  });

  testWidgets('staff can read plans but not manage them', (tester) async {
    await mount(tester, role: Role.staff);
    await tester.tap(find.text('Manage plans').first);
    await tester.pumpAndSettle();
    expect(find.text('Create Plan'), findsNothing);
    expect(find.byTooltip('Manage Starter Monthly'), findsNothing);
    // Staff still see the per-card subscriber footer, now a Subscribers / N
    // label-value pair instead of a single "Subscribers: 3" line.
    expect(find.text('Subscribers'), findsWidgets);
    expect(find.text('3'), findsWidgets);
    expect(tester.takeException(), isNull);
  });

  testWidgets('customer catalogue hides company metrics', (tester) async {
    await mount(tester, role: Role.customer);
    // Customers now land on a browse-by-product grid, not a flat plans list.
    expect(find.text('Browse products'), findsOneWidget);
    expect(find.text('Software Subscription'), findsOneWidget);
    expect(find.text('View plans'), findsWidgets);
    expect(find.textContaining('Subscribers'), findsNothing);
    expect(find.text('Active subscribers'), findsNothing);
    expect(find.text('Create Plan'), findsNothing);
    expect(tester.takeException(), isNull);
    // Picking the product opens the tier-card view with Request quotation.
    await tester.tap(find.text('View plans').first);
    await tester.pumpAndSettle();
    expect(find.text('Starter Monthly'), findsOneWidget);
    expect(find.text('Request quotation'), findsWidgets);
    expect(find.text('All products'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'admin creates a product-specific plan for a product without plans',
    (tester) async {
      final api = await mount(tester, extraProduct: true);
      await tester.tap(find.text('Support'));
      await tester.pumpAndSettle();
      expect(find.text('No plans for this product yet'), findsOneWidget);
      expect(find.text('Starter Monthly'), findsNothing);
      await tester.tap(find.text('Create Plan'));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Plan name'),
        'Support quarterly',
      );
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Every N periods'),
        '3',
      );
      await tester.ensureVisible(find.text('Save'));
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      final body = api.posted['/plans'] as Map<String, dynamic>;
      expect(body['billing_interval'], 3);
      expect((body['products'] as List).single['product_id'], 8);
      expect(find.byType(AlertDialog), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

  for (final scale in [1.0, 2.0]) {
    testWidgets('customer catalogue and pricing fit a phone at $scale text', (
      tester,
    ) async {
      await mount(
        tester,
        role: Role.customer,
        size: const Size(375, 812),
        textScale: scale,
      );
      expect(tester.takeException(), isNull);
      await capture(tester, 'customer-plans-phone-$scale');
      await tester.ensureVisible(find.text('View plans').first);
      await tester.tap(find.text('View plans').first);
      await tester.pumpAndSettle();
      expect(find.text('Request quotation'), findsWidgets);
      expect(tester.takeException(), isNull);
      await capture(tester, 'customer-pricing-phone-$scale');
    });
  }

  testWidgets('phone layout and drawer navigate without overflow', (
    tester,
  ) async {
    await mount(tester, size: const Size(375, 812));
    expect(tester.takeException(), isNull);
    await capture(tester, 'plans-phone');
    await tester.tap(find.byTooltip('Open navigation'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Dashboard'));
    await tester.pumpAndSettle();
    expect(find.text('Overview'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  for (final (name, size) in [
    ('tablet', const Size(768, 1024)),
    ('landscape', const Size(1024, 768)),
  ]) {
    testWidgets('$name plans do not overflow', (tester) async {
      await mount(tester, size: size);
      expect(tester.takeException(), isNull);
      await capture(tester, 'plans-$name');
    });
  }

  testWidgets('large text remains readable without overflow', (tester) async {
    await mount(tester, size: const Size(375, 812), textScale: 2);
    expect(tester.takeException(), isNull);
  });

  testWidgets('dashboard shows live counts on desktop and phone', (
    tester,
  ) async {
    await mount(tester, path: '/dashboard');
    expect(find.text('Overview'), findsOneWidget);
    expect(find.text('12'), findsOneWidget); // customers total from meta
    expect(find.text('Acme Pvt Ltd'), findsWidgets);
    expect(tester.takeException(), isNull);
    await capture(tester, 'overview-desktop');
    await mount(tester, path: '/dashboard', size: const Size(375, 812));
    expect(tester.takeException(), isNull);
    await capture(tester, 'overview-phone');
  });

  testWidgets('products: admin sees cost and margin', (tester) async {
    await mount(tester, path: '/products');
    expect(find.textContaining('margin ₹699.00'), findsOneWidget);
    expect(find.text('Odoo Brand ₹1,559.00'), findsOneWidget);
    expect(find.text('New product'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  for (final (name, size, scale) in [
    ('desktop', const Size(1920, 1080), 1.0),
    ('phone', const Size(375, 812), 1.0),
    ('landscape', const Size(1024, 768), 1.0),
    ('large-text', const Size(375, 812), 2.0),
  ]) {
    testWidgets('customer dashboard adapts to $name', (tester) async {
      await mount(
        tester,
        role: Role.customer,
        path: '/dashboard',
        size: size,
        textScale: scale,
      );
      expect(find.text('Welcome back, Thomas.'), findsOneWidget);
      expect(find.text('Starter Monthly'), findsOneWidget);
      expect(find.text('thomas@brightloop.in'), findsOneWidget);
      expect(find.text('Active subscribers'), findsNothing);
      expect(tester.takeException(), isNull);
      await capture(tester, 'customer-dashboard-$name');
      await tester.ensureVisible(find.text('Explore plans'));
      await tester.tap(find.text('Explore plans'));
      await tester.pumpAndSettle();
      expect(find.text('Browse products'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }

  for (final count in [0, 1]) {
    testWidgets('customer dashboard handles $count available plans', (
      tester,
    ) async {
      await mount(
        tester,
        role: Role.customer,
        path: '/dashboard',
        size: const Size(1920, 1080),
        dashboardPlans: [
          if (count == 1)
            Plan.fromJson(
              FakeApi.plan(1, 'Starter Monthly', '999.00', 'Monthly'),
            ),
        ],
      );
      expect(tester.takeException(), isNull);
      expect(
        find.text('Starter Monthly'),
        count == 1 ? findsOneWidget : findsNothing,
      );
      expect(
        find.text('New possibilities are on the way'),
        count == 0 ? findsOneWidget : findsNothing,
      );
      await capture(tester, 'customer-dashboard-$count-plans');
    });
  }

  testWidgets('products: staff see no cost and cannot create', (tester) async {
    await mount(tester, path: '/products', role: Role.staff);
    expect(find.textContaining('margin'), findsNothing);
    expect(find.text('New product'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  for (final action in ['Cancel', 'Save']) {
    testWidgets('products: $action safely closes a focused form', (
      tester,
    ) async {
      final api = await mount(tester, path: '/products');
      await tester.tap(find.text('New product'));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Name'),
        'Example',
      );
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Sales price (₹)'),
        '10',
      );
      await tester.tap(find.text(action));
      await tester.pumpAndSettle();
      expect(find.byType(AlertDialog), findsNothing);
      expect(api.posted.containsKey('/products'), action == 'Save');
      expect(tester.takeException(), isNull);
      // The same page must remain usable after the dialog has been removed.
      await tester.tap(find.text('New product'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
    });
  }
}
