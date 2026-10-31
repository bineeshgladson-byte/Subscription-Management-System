import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/utils/formatters.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../sales/data/sales_api.dart';

/// Admin settings: run billing now (the scheduler also runs it every hour).
class SettingsScreen extends ConsumerStatefulWidget {
  const SettingsScreen({super.key});
  @override
  ConsumerState<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends ConsumerState<SettingsScreen> {
  final _asOf = TextEditingController(text: Fmt.apiDate(DateTime.now()));
  Map<String, dynamic>? _result;
  bool _busy = false;

  @override
  void dispose() {
    _asOf.dispose();
    super.dispose();
  }

  Future<void> _run() async {
    setState(() => _busy = true);
    final today = Fmt.apiDate(DateTime.now());
    await runAction(context, () async {
      final r = await ref
          .read(salesRepositoryProvider)
          .runBilling(
            asOf: _asOf.text.trim() == today ? null : _asOf.text.trim(),
          );
      setState(() => _result = r);
      invalidateSales(ref);
    });
    if (mounted) setState(() => _busy = false);
  }

  @override
  Widget build(BuildContext context) => PageFrame(
    title: 'Settings',
    subtitle:
        'Billing runs automatically every hour. Run it now, or simulate a future date (development only).',
    child: Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Run billing', style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 6),
            Text(
              'Activates subscriptions whose start date has arrived, closes expired quotations, and creates every due invoice. '
              'Safe to run any number of times: a period is never billed twice.',
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 14),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: [
                SizedBox(
                  width: 220,
                  child: DateField(
                    controller: _asOf,
                    label: 'Bill as of',
                    required: true,
                  ),
                ),
                FilledButton.icon(
                  onPressed: _busy ? null : _run,
                  icon: const Icon(Icons.play_arrow),
                  label: Text(_busy ? 'Running…' : 'Run billing'),
                ),
              ],
            ),
            if (_result != null) ...[
              const SizedBox(height: 14),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  Pill(
                    'As of ${Fmt.date('${_result!['as_of']}')}',
                    tone: 'brand',
                  ),
                  Pill(
                    '${_result!['invoices_created']} invoices created',
                    tone: 'good',
                  ),
                  Pill('${_result!['activated']} activated'),
                  Pill('${_result!['closed']} closed'),
                  for (final e in (_result!['errors'] as List? ?? const []))
                    Pill('$e', tone: 'bad'),
                ],
              ),
            ],
          ],
        ),
      ),
    ),
  );
}
