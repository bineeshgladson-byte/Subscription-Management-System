/// Small helpers for reading API JSON safely.
String str(Object? v, [String fallback = '']) => v?.toString() ?? fallback;
String? strOrNull(Object? v) => v?.toString();
int toInt(Object? v, [int fallback = 0]) =>
    v is int ? v : int.tryParse('$v') ?? fallback;
int? intOrNull(Object? v) =>
    v == null ? null : (v is int ? v : int.tryParse('$v'));
bool toBool(Object? v) => v == true;
List<Map<String, dynamic>> listOf(Object? v) =>
    (v as List? ?? const []).cast<Map<String, dynamic>>();
DateTime? dateOrNull(Object? v) => v == null ? null : DateTime.tryParse('$v');
