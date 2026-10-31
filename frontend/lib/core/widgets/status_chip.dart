import 'package:flutter/material.dart';

/// Coloured pill for subscription / invoice / payment statuses.
class StatusChip extends StatelessWidget {
  const StatusChip(this.status, {super.key});
  final String status;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final (bg, fg) = switch (status) {
      'active' ||
      'paid' ||
      'completed' => (scheme.primaryContainer, scheme.onPrimaryContainer),
      'overdue' ||
      'failed' ||
      'cancelled' => (scheme.errorContainer, scheme.onErrorContainer),
      'paused' ||
      'partial' ||
      'pending' ||
      'quotation' => (scheme.tertiaryContainer, scheme.onTertiaryContainer),
      _ => (scheme.surfaceContainerHighest, scheme.onSurfaceVariant),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(99),
      ),
      child: Text(
        status,
        style: TextStyle(color: fg, fontSize: 12, fontWeight: FontWeight.w600),
      ),
    );
  }
}
