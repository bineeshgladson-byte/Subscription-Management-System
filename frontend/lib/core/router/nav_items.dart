import 'package:flutter/material.dart';

import '../models/role.dart';

/// Single source of truth for the sidebar AND for route-level role checks.
/// Mirrors the permission matrix in the architecture plan; the backend
/// enforces the same rules, this only shapes the UI.
class NavItem {
  const NavItem(this.path, this.label, this.icon, this.roles);
  final String path;
  final String label;
  final IconData icon;
  final Set<Role> roles;
}

const _all = {Role.admin, Role.staff, Role.customer};
const _team = {Role.admin, Role.staff};
const _admin = {Role.admin};

const navItems = <NavItem>[
  NavItem('/dashboard', 'Dashboard', Icons.space_dashboard_outlined, _all),
  NavItem('/customers', 'Customers', Icons.people_outline, _team),
  NavItem('/staff', 'Staff', Icons.badge_outlined, _admin),
  NavItem('/products', 'Products', Icons.inventory_2_outlined, _team),
  NavItem('/plans', 'Plans', Icons.layers_outlined, _all),
  NavItem('/quotations', 'Quotations', Icons.request_quote_outlined, _team),
  NavItem('/subscriptions', 'Subscriptions', Icons.autorenew, _all),
  NavItem('/invoices', 'Invoices', Icons.receipt_long_outlined, _all),
  NavItem('/payments', 'Payments', Icons.payments_outlined, _all),
  NavItem('/discounts', 'Discounts', Icons.local_offer_outlined, _admin),
  NavItem('/reports', 'Reports', Icons.insights_outlined, _team),
  NavItem('/settings', 'Settings', Icons.settings_outlined, _admin),
  NavItem('/profile', 'Profile', Icons.person_outline, _all),
];

List<NavItem> navItemsFor(Role role) =>
    navItems.where((i) => i.roles.contains(role)).toList();

/// Returns false if [location] belongs to a nav item the role can't access.
bool canAccess(Role role, String location) {
  for (final item in navItems) {
    if (location == item.path || location.startsWith('${item.path}/')) {
      return item.roles.contains(role);
    }
  }
  return true; // routes not in the menu (e.g. /forbidden) are open to logged-in users
}
