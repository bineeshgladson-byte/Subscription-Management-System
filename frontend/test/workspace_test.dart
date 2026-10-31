import 'dart:io';
import 'dart:ui' as ui;

import 'package:brightloop/core/models/role.dart';
import 'package:brightloop/core/network/api_client.dart';
import 'package:brightloop/core/theme/workspace_theme.dart';
import 'package:brightloop/core/widgets/app_shell.dart';
import 'package:brightloop/features/auth/application/auth_controller.dart';
import 'package:brightloop/features/auth/data/auth_user.dart';
import 'package:brightloop/features/dashboard/presentation/dashboard_screen.dart';
import 'package:brightloop/features/plans/presentation/plans_screen.dart';
import 'package:brightloop/features/pricing/presentation/pricing_screen.dart';
import 'package:brightloop/features/products/presentation/products_screen.dart';
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
  FakeApi(this.role) : super(Dio());
  final Role role;
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
    '/products' => [product(role == Role.admin)],
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
    final items = _list(path).map(fromJson).toList();
    return Paged(
      items: items,
      page: 1,
      pageSize: 20,
      total: path == '/customers' ? 12 : items.length,
    );
  }

  @override
  Future<dynamic> get(String path, {Map<String, dynamic>? query}) async =>
      path.startsWith('/products/') ? product(role == Role.admin) : {};

  @override
  Future<dynamic> post(
    String path, {
    Object? body,
    Map<String, String>? headers,
  }) async {
    posted[path] = body;
    if (path == '/pricing/preview') {
      return {
        'date': '2026-11-01',
        'subtotal': '15590.00',
        'discount_total': '1559.00',
        'tax_total': '2525.58',
        'total': '16556.58',
        'discounts': [
          {
            'discount_id': 5,
            'name': '10% off orders over 10,000',
            'scope': 'subscription',
            'amount': '1559.00',
          },
        ],
        'lines': [
          {
            'product_id': 7,
            'description': 'Software Subscription (Odoo Brand)',
            'quantity': 10,
            'unit_price': '1559.00',
            'subtotal': '15590.00',
            'product_discount': '0.00',
            'product_discount_name': null,
            'subscription_discount': '1559.00',
            'tax_rate': '18.00',
            'tax_amount': '2525.58',
            'total': '16556.58',
          },
        ],
      };
    }
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
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final api = FakeApi(role);
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
            GoRoute(path: '/pricing', builder: (_, _) => const PricingScreen()),
            GoRoute(
              path: '/profile',
              builder: (_, _) => const Scaffold(body: Text('Profile')),
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
        ],
        child: MaterialApp.router(
          theme: WorkspaceTheme.data,
          routerConfig: router,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(
              textScaler: TextScaler.linear(textScale),
              disableAnimations: true,
            ),
            child: child!,
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

  testWidgets('admin sees live plans with prices, subscribers and management', (
    tester,
  ) async {
    await mount(tester);
    expect(find.text('Starter Monthly'), findsOneWidget);
    expect(find.text('Business Yearly'), findsOneWidget);
    expect(find.text('₹999.00'), findsOneWidget);
    expect(find.text('₹1,559.00'), findsWidgets);
    expect(find.text('Create Plan'), findsOneWidget);
    expect(find.text('5'), findsOneWidget); // active subscribers 3 + 2
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
    expect(find.text('Create Plan'), findsNothing);
    expect(find.byTooltip('Manage Starter Monthly'), findsNothing);
    expect(find.text('Subscribers: 3'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('customer catalogue hides company metrics', (tester) async {
    await mount(tester, role: Role.customer);
    expect(find.text('Available plans'), findsOneWidget);
    expect(find.textContaining('Subscribers'), findsNothing);
    expect(find.text('Active subscribers'), findsNothing);
    expect(find.text('Create Plan'), findsNothing);
    expect(tester.takeException(), isNull);
  });

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

  testWidgets('products: staff see no cost and cannot create', (tester) async {
    await mount(tester, path: '/products', role: Role.staff);
    expect(find.textContaining('margin'), findsNothing);
    expect(find.text('New product'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('price calculator sends lines and shows the breakdown', (
    tester,
  ) async {
    final api = await mount(tester, path: '/pricing');
    await tester.tap(find.byType(DropdownButtonFormField<int>).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Software Subscription · ₹999.00').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Calculate'));
    await tester.pumpAndSettle();
    expect(api.posted['/pricing/preview'], isNotNull);
    expect(find.text('₹16,556.58'), findsOneWidget);
    expect(find.text('10% off orders over 10,000: −₹1,559.00'), findsOneWidget);
    expect(tester.takeException(), isNull);
    await capture(tester, 'pricing-desktop');
  });
}
