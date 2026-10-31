import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/widgets/simple_screens.dart';
import '../../auth/application/auth_controller.dart';

/// TODO: replace PlaceholderView with the real screen.
/// Pattern: data/x_repository.dart → application/x_providers.dart → this screen.
/// See features/auth for a complete example.
class InvoicesScreen extends ConsumerWidget {
  const InvoicesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final role = ref.watch(authControllerProvider).value?.role ?? Role.customer;
    return PlaceholderView(
      title: 'Invoices',
      description: role.scope == Scope.mine
          ? 'Customer view: only your own records (/me endpoints).'
          : 'Filter by status, overdue, customer and dates. Confirm, cancel, send, print.',
      endpoints: const [
        'GET /invoices?status&overdue&customer_id&from&to',
        'GET /invoices/:id',
        'POST /invoices/:id/confirm|cancel|send',
        'GET /invoices/:id/pdf',
        'GET /me/invoices   (customer)',
      ],
    );
  }
}
