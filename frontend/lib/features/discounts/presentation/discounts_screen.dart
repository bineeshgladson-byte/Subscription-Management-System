import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/widgets/simple_screens.dart';
import '../../auth/application/auth_controller.dart';

/// TODO: replace PlaceholderView with the real screen.
/// Pattern: data/x_repository.dart → application/x_providers.dart → this screen.
/// See features/auth for a complete example.
class DiscountsScreen extends ConsumerWidget {
  const DiscountsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final role = ref.watch(authControllerProvider).value?.role ?? Role.customer;
    return PlaceholderView(
      title: 'Discounts',
      description: role.scope == Scope.mine
          ? 'Customer view: only your own records (/me endpoints).'
          : 'Fixed or percentage, product or subscription scope, minimums, dates, usage limits.',
      endpoints: const [
        'GET|POST /discounts',
        'GET|PUT|DELETE /discounts/:id',
        'GET /discounts/:id/usages',
      ],
    );
  }
}
