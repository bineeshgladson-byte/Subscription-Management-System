import 'package:brightloop/core/models/role.dart';
import 'package:brightloop/core/router/nav_items.dart';
import 'package:brightloop/core/utils/formatters.dart';
import 'package:brightloop/core/utils/validators.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('password policy (must match backend/internal/validator)', () {
    test(
      'rejects 8 characters',
      () => expect(Validators.password('Short#1a'), isNotNull),
    );
    test(
      'accepts a strong password',
      () => expect(Validators.password('Brightloop#2026'), isNull),
    );
    test(
      'needs uppercase',
      () => expect(Validators.password('brightloop#2026'), isNotNull),
    );
    test(
      'needs special char',
      () => expect(Validators.password('Brightloop2026'), isNotNull),
    );
  });

  test('money formats in Indian style', () {
    expect(Fmt.inr('117882.50'), '₹1,17,882.50');
  });

  test('role-based route access', () {
    expect(canAccess(Role.customer, '/discounts'), isFalse);
    expect(canAccess(Role.staff, '/staff'), isFalse);
    expect(canAccess(Role.admin, '/staff'), isTrue);
    expect(canAccess(Role.customer, '/invoices'), isTrue);
  });
}
