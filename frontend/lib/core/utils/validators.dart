/// Form validators. They mirror backend/internal/validator so users get
/// instant feedback, but the server always validates again.
class Validators {
  static String? required(String? v, [String field = 'This field']) =>
      (v == null || v.trim().isEmpty) ? '$field is required' : null;

  static String? email(String? v) {
    if (v == null || v.trim().isEmpty) return 'Email is required';
    final ok = RegExp(r'^[^@\s]+@[^@\s]+\.[^@\s]+$').hasMatch(v.trim());
    return ok ? null : 'Enter a valid email';
  }

  /// 9+ characters ("more than 8"), uppercase, lowercase, special character.
  static String? password(String? v) {
    if (v == null || v.length < 9) return 'At least 9 characters';
    if (!v.contains(RegExp(r'[A-Z]'))) return 'Add an uppercase letter';
    if (!v.contains(RegExp(r'[a-z]'))) return 'Add a lowercase letter';
    if (!v.contains(RegExp(r'[^A-Za-z0-9\s]'))) {
      return 'Add a special character';
    }
    return null;
  }

  static String? positiveMoney(String? v) {
    final n = num.tryParse(v ?? '');
    if (n == null) return 'Enter an amount';
    if (n <= 0) return 'Must be greater than 0';
    if (!RegExp(r'^\d+(\.\d{1,2})?$').hasMatch(v!.trim())) {
      return 'Max 2 decimal places';
    }
    return null;
  }
}
