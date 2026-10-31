enum Role {
  admin,
  staff,
  customer;

  static Role fromJson(String value) => Role.values.firstWhere(
    (r) => r.name == value,
    orElse: () => Role.customer,
  );

  String get label => switch (this) {
    Role.admin => 'Admin',
    Role.staff => 'Staff',
    Role.customer => 'Customer',
  };
}

/// Whether a list screen shows everything (admin/staff → /invoices)
/// or only the caller's own records (customer → /me/invoices).
enum Scope { all, mine }

extension RoleScope on Role {
  Scope get scope => this == Role.customer ? Scope.mine : Scope.all;
}
