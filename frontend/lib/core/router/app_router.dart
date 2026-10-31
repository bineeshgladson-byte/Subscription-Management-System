import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/application/auth_controller.dart';
import '../../features/auth/presentation/forgot_password_screen.dart';
import '../../features/auth/presentation/login_screen.dart';
import '../../features/auth/presentation/register_screen.dart';
import '../../features/customers/presentation/customers_screen.dart';
import '../../features/dashboard/presentation/dashboard_screen.dart';
import '../../features/discounts/presentation/discounts_screen.dart';
import '../../features/invoices/presentation/invoices_screen.dart';
import '../../features/payments/presentation/payments_screen.dart';
import '../../features/plans/presentation/plans_screen.dart';
import '../../features/portal/presentation/portal_screens.dart';
import '../../features/products/presentation/products_screen.dart';
import '../../features/profile/presentation/profile_screen.dart';
import '../../features/quotations/presentation/quotations_screen.dart';
import '../../features/settings/presentation/settings_screen.dart';
import '../../features/staff/presentation/staff_screen.dart';
import '../../features/subscriptions/presentation/subscriptions_screen.dart';
import '../../features/taxes/presentation/taxes_screen.dart';
import '../../features/templates/presentation/templates_screen.dart';
import '../widgets/app_shell.dart';
import '../widgets/simple_screens.dart';
import 'nav_items.dart';

const _publicPaths = {'/login', '/register', '/forgot-password'};

final routerProvider = Provider<GoRouter>((ref) {
  // Re-run redirect whenever auth state changes (login, logout, 401).
  final refresh = ValueNotifier<int>(0);
  ref.listen(authControllerProvider, (_, _) => refresh.value++);
  ref.onDispose(refresh.dispose);

  return GoRouter(
    initialLocation: '/dashboard',
    refreshListenable: refresh,
    redirect: (context, state) {
      final auth = ref.read(authControllerProvider);
      final loc = state.matchedLocation;

      if (auth.isLoading) return loc == '/splash' ? null : '/splash';

      final user = auth.value;
      final isPublic = _publicPaths.contains(loc);

      if (user == null) return isPublic ? null : '/login';
      if (isPublic || loc == '/splash') return '/dashboard';
      if (!canAccess(user.role, loc)) return '/forbidden';
      return null;
    },
    routes: [
      GoRoute(path: '/splash', builder: (_, _) => const SplashScreen()),
      GoRoute(path: '/login', builder: (_, _) => const LoginScreen()),
      GoRoute(path: '/register', builder: (_, _) => const RegisterScreen()),
      GoRoute(
        path: '/forgot-password',
        builder: (_, _) => const ForgotPasswordScreen(),
      ),
      ShellRoute(
        pageBuilder: (context, state, child) => NoTransitionPage(
          key: state.pageKey,
          child: AppShell(child: child),
        ),
        routes: [
          // Feature owners: add detail routes as children, e.g.
          // GoRoute(path: '/invoices', ..., routes: [GoRoute(path: ':id', ...)])
          GoRoute(
            path: '/dashboard',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const DashboardScreen(),
            ),
          ),
          GoRoute(
            path: '/customers',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const CustomersScreen(),
            ),
          ),
          GoRoute(
            path: '/staff',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const StaffScreen(),
            ),
          ),
          GoRoute(
            path: '/products',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const ProductsScreen(),
            ),
          ),
          GoRoute(
            path: '/taxes',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const TaxesScreen(),
            ),
          ),
          GoRoute(
            path: '/plans',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: PlansScreen(
                initialProductId: int.tryParse(
                  state.uri.queryParameters['product_id'] ?? '',
                ),
                initialProductName: state.uri.queryParameters['product_name'],
              ),
            ),
          ),
          GoRoute(
            path: '/templates',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const TemplatesScreen(),
            ),
          ),
          GoRoute(
            path: '/quotations',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const QuotationsScreen(),
            ),
          ),
          GoRoute(
            path: '/subscriptions',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const SubscriptionsScreen(),
            ),
          ),
          GoRoute(
            path: '/invoices',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const InvoicesScreen(),
            ),
          ),
          GoRoute(
            path: '/payments',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const PaymentsScreen(),
            ),
          ),
          GoRoute(
            path: '/discounts',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const DiscountsScreen(),
            ),
          ),
          GoRoute(
            path: '/settings',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const SettingsScreen(),
            ),
          ),
          GoRoute(
            path: '/profile',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const ProfileScreen(),
            ),
          ),
          GoRoute(
            path: '/my-subscriptions',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const MySubscriptionsScreen(),
            ),
          ),
          GoRoute(
            path: '/my-invoices',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const MyInvoicesScreen(),
            ),
          ),
          GoRoute(
            path: '/my-payments',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const MyPaymentsScreen(),
            ),
          ),
          GoRoute(
            path: '/forbidden',
            pageBuilder: (_, state) => NoTransitionPage(
              key: state.pageKey,
              child: const ForbiddenScreen(),
            ),
          ),
        ],
      ),
    ],
    errorBuilder: (_, state) => NotFoundScreen(path: state.uri.toString()),
  );
});
