import 'dart:io';
import 'dart:ui' as ui;

import 'package:brightloop/core/models/role.dart';
import 'package:brightloop/core/theme/workspace_theme.dart';
import 'package:brightloop/core/widgets/app_shell.dart';
import 'package:brightloop/features/auth/application/auth_controller.dart';
import 'package:brightloop/features/auth/data/auth_user.dart';
import 'package:brightloop/features/dashboard/presentation/dashboard_screen.dart';
import 'package:brightloop/features/plans/presentation/plans_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

class _PreviewAuth extends AuthController {
  _PreviewAuth(this.role);
  final Role role;
  @override
  Future<AuthUser?> build() async => AuthUser(
    id: 1,
    email: 'preview@example.com',
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

  Future<void> mount(
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
        overrides: [
          authControllerProvider.overrideWith(() => _PreviewAuth(role)),
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
    'desktop pricing matches reference structure and supports preview edits',
    (tester) async {
      await mount(tester);
      expect(find.text('Most Popular'), findsOneWidget);
      expect(find.text('Create Plan'), findsOneWidget);
      expect(find.text('Enterprise'), findsOneWidget);
      expect(find.text(r'$48,546'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await capture(tester, 'pricing-desktop');
      await tester.tap(find.text('Create Plan'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Save preview'));
      await tester.pumpAndSettle();
      expect(find.text('Enter a plan name'), findsOneWidget);
      await tester.enterText(find.byType(TextFormField).at(0), 'Growth');
      await tester.enterText(find.byType(TextFormField).at(1), '129');
      await tester.enterText(
        find.byType(TextFormField).at(2),
        'A little more room to grow',
      );
      await tester.tap(find.text('Save preview'));
      await tester.pumpAndSettle();
      expect(find.text('Growth'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.tap(find.byTooltip('Manage Starter'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Archive plan'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Archive'));
      await tester.pumpAndSettle();
      expect(find.text('Starter'), findsNothing);
    },
  );

  testWidgets('phone layout and drawer navigate without overflow', (
    tester,
  ) async {
    await mount(tester, size: const Size(375, 812));
    expect(tester.takeException(), isNull);
    await capture(tester, 'pricing-phone');
    await tester.tap(find.byTooltip('Open navigation'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Dashboard'));
    await tester.pumpAndSettle();
    expect(find.text('Overview'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('staff cannot manage plans or see financial metrics', (
    tester,
  ) async {
    await mount(tester, role: Role.staff);
    expect(find.text('Create Plan'), findsNothing);
    expect(find.byTooltip('Manage Starter'), findsNothing);
    expect(find.text('Total MRR from Plans'), findsNothing);
    expect(find.text('MRR Contribution'), findsNothing);
    expect(find.text('Subscribers'), findsNWidgets(4));
    expect(tester.takeException(), isNull);
  });

  testWidgets('customer catalogue hides company metrics', (tester) async {
    await mount(tester, role: Role.customer);
    expect(find.text('Active Subscribers'), findsNothing);
    expect(find.text('Subscribers'), findsNothing);
    expect(find.text('Create Plan'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  for (final (name, size) in [
    ('tablet', const Size(768, 1024)),
    ('landscape', const Size(1024, 768)),
  ]) {
    testWidgets('$name pricing does not overflow', (tester) async {
      await mount(tester, size: size);
      expect(tester.takeException(), isNull);
      await capture(tester, 'pricing-$name');
    });
  }

  testWidgets('large text remains readable without overflow', (tester) async {
    await mount(tester, size: const Size(375, 812), textScale: 2);
    expect(tester.takeException(), isNull);
  });

  testWidgets('dashboard adapts to desktop and phone', (tester) async {
    await mount(tester, path: '/dashboard');
    expect(find.text('Overview'), findsOneWidget);
    expect(tester.takeException(), isNull);
    await capture(tester, 'overview-desktop');
    await mount(tester, path: '/dashboard', size: const Size(375, 812));
    expect(tester.takeException(), isNull);
    await capture(tester, 'overview-phone');
  });
}
