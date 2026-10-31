import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/widgets/simple_screens.dart';
import '../../auth/application/auth_controller.dart';

/// TODO: replace PlaceholderView with the real screen.
/// Pattern: data/x_repository.dart → application/x_providers.dart → this screen.
/// See features/auth for a complete example.
class SubscriptionsScreen extends ConsumerWidget {
  const SubscriptionsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final role = ref.watch(authControllerProvider).value?.role ?? Role.customer;
    return PlaceholderView(
      title: 'Subscriptions',
      description: role.scope == Scope.mine
          ? 'Customer view: only your own records (/me endpoints).'
          : 'List and detail with lifecycle actions (pause, resume, close, renew, bill now).',
      endpoints: const [
        'GET /subscriptions?status&customer_id',
        'GET /subscriptions/:id',
        'POST /subscriptions/:id/activate|pause|resume|close|renew|bill-now',
        'GET|POST /me/subscriptions   (customer)',
      ],
    );
  }
}
