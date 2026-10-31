import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/application/auth_controller.dart';
import '../router/nav_items.dart';

/// One layout for every role. The menu is filtered by role from nav_items.dart:
/// sidebar (NavigationRail) on wide screens, drawer on phones.
class AppShell extends ConsumerWidget {
  const AppShell({super.key, required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authControllerProvider).value;
    if (user == null) return const SizedBox.shrink();

    final items = navItemsFor(user.role);
    final loc = GoRouterState.of(context).matchedLocation;
    final selected = items.indexWhere(
      (i) => loc == i.path || loc.startsWith('${i.path}/'),
    );
    final wide = MediaQuery.sizeOf(context).width >= 900;

    final appBar = AppBar(
      title: Text(selected >= 0 ? items[selected].label : 'Brightloop'),
      actions: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8),
          child: Center(child: Text('${user.fullName} · ${user.role.label}')),
        ),
        IconButton(
          tooltip: 'Log out',
          icon: const Icon(Icons.logout),
          onPressed: () => ref.read(authControllerProvider.notifier).logout(),
        ),
      ],
    );

    if (!wide) {
      return Scaffold(
        appBar: appBar,
        drawer: Drawer(
          child: ListView(
            children: [
              const DrawerHeader(
                child: Text('Brightloop', style: TextStyle(fontSize: 22)),
              ),
              for (final (i, item) in items.indexed)
                ListTile(
                  leading: Icon(item.icon),
                  title: Text(item.label),
                  selected: i == selected,
                  onTap: () {
                    Navigator.of(context).pop();
                    context.go(item.path);
                  },
                ),
            ],
          ),
        ),
        body: child,
      );
    }

    return Scaffold(
      appBar: appBar,
      body: Row(
        children: [
          NavigationRail(
            extended: MediaQuery.sizeOf(context).width >= 1200,
            selectedIndex: selected >= 0 ? selected : null,
            onDestinationSelected: (i) => context.go(items[i].path),
            labelType: NavigationRailLabelType.none,
            destinations: [
              for (final item in items)
                NavigationRailDestination(
                  icon: Icon(item.icon),
                  label: Text(item.label),
                ),
            ],
          ),
          const VerticalDivider(width: 1),
          Expanded(child: child),
        ],
      ),
    );
  }
}
