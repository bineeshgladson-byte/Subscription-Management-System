import 'package:flutter/material.dart';

import '../../../core/utils/formatters.dart';
import '../../../core/widgets/ui_kit.dart';
import '../data/sales_api.dart';

String statusTone(String s) => switch (s) {
  'active' || 'paid' || 'completed' => 'good',
  'paused' || 'partial' || 'quotation' || 'draft' => 'warn',
  'cancelled' || 'closed' || 'overdue' || 'failed' => 'bad',
  'confirmed' => 'brand',
  _ => 'neutral',
};

Widget statusPill(String s) =>
    Pill(s.replaceAll('_', ' '), tone: statusTone(s));

Widget invoicePills(InvoiceBrief i) => Wrap(
  spacing: 6,
  runSpacing: 6,
  children: [
    statusPill(i.status),
    if (i.status != 'cancelled')
      Pill(i.paymentStatus, tone: statusTone(i.paymentStatus)),
    if (i.isOverdue) const Pill('Overdue', tone: 'bad'),
  ],
);

Widget moneyRow(
  BuildContext context,
  String label,
  String value, {
  bool bold = false,
}) => Padding(
  padding: const EdgeInsets.symmetric(vertical: 3),
  child: Row(
    children: [
      Expanded(
        child: Text(
          label,
          style: TextStyle(
            fontWeight: bold ? FontWeight.w600 : FontWeight.w400,
          ),
        ),
      ),
      Text(
        value,
        style: TextStyle(fontWeight: bold ? FontWeight.w600 : FontWeight.w400),
      ),
    ],
  ),
);

String period(InvoiceBrief i) => i.periodStart == null
    ? Fmt.date(i.invoiceDate)
    : '${Fmt.date(i.periodStart!)} – ${Fmt.date(i.periodEnd!)}';
