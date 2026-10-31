import 'package:decimal/decimal.dart';
import 'package:decimal/intl.dart';
import 'package:intl/intl.dart';

/// Money arrives from the API as a string ("1178.82"). Parse it with Decimal,
/// NEVER double, and format it for display here.
class Fmt {
  static final _inr = DecimalFormatter(
    NumberFormat.currency(locale: 'en_IN', symbol: '₹', decimalDigits: 2),
  );
  static final _date = DateFormat('d MMM yyyy');

  static Decimal money(String raw) => Decimal.parse(raw);

  /// "1178.82" → "₹1,178.82"
  static String inr(String raw) => _inr.format(Decimal.parse(raw));

  /// "2026-11-16" → "16 Nov 2026"
  static String date(String isoDate) => _date.format(DateTime.parse(isoDate));

  /// For sending dates to the API.
  static String apiDate(DateTime d) => DateFormat('yyyy-MM-dd').format(d);
}
