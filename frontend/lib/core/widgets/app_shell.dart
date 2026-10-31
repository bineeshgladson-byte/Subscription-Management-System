import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/application/auth_controller.dart';
import '../../features/auth/data/auth_user.dart';
import '../router/nav_items.dart';
import '../theme/workspace_theme.dart';
import 'brand_mark.dart';

class AppShell extends ConsumerWidget {
  const AppShell({super.key, required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authControllerProvider).value;
    if (user == null) return const SizedBox.shrink();
    final location = GoRouterState.of(context).matchedLocation;
    return Theme(
      data: WorkspaceTheme.data,
      child: LayoutBuilder(
        builder: (context, constraints) {
          final desktop = constraints.maxWidth >= 1000;
          final sidebar = _Sidebar(user: user, location: location);
          final workspace = ClipRRect(
            borderRadius: BorderRadius.only(
              topLeft: Radius.circular(desktop ? 24 : 0),
              bottomLeft: Radius.circular(desktop ? 24 : 0),
            ),
            child: ColoredBox(
              color: Colors.white,
              child: Column(
                children: [
                  _Topbar(user: user, mobile: !desktop),
                  Expanded(child: child),
                ],
              ),
            ),
          );
          return Scaffold(
            backgroundColor: WorkspaceTheme.background,
            drawer: desktop ? null : Drawer(child: sidebar),
            body: SafeArea(
              child: DecoratedBox(
                decoration: const BoxDecoration(
                  gradient: LinearGradient(
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                    colors: [WorkspaceTheme.lavender, Color(0xFFFAF9FF)],
                  ),
                ),
                child: Row(
                  children: [
                    if (desktop)
                      SizedBox(
                        width: constraints.maxWidth >= 1400 ? 250 : 220,
                        child: sidebar,
                      ),
                    Expanded(child: workspace),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}

class _Sidebar extends StatelessWidget {
  const _Sidebar({required this.user, required this.location});
  final AuthUser user;
  final String location;

  @override
  Widget build(BuildContext context) {
    final items = navItemsFor(user.role);
    final mainItems = items.where(
      (i) => i.path != '/settings' && i.path != '/profile',
    );
    Widget destination(NavItem item) {
      final selected =
          location == item.path || location.startsWith('${item.path}/');
      return Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 2),
        child: Material(
          color: selected ? WorkspaceTheme.selected : Colors.transparent,
          borderRadius: BorderRadius.circular(24),
          child: ListTile(
            minTileHeight: 44,
            contentPadding: const EdgeInsets.symmetric(horizontal: 14),
            horizontalTitleGap: 12,
            leading: Icon(item.icon, size: 20, color: WorkspaceTheme.ink),
            title: Text(
              item.label,
              style: TextStyle(
                fontSize: 12,
                fontWeight: selected ? FontWeight.w500 : FontWeight.w400,
              ),
            ),
            selected: selected,
            textColor: WorkspaceTheme.ink,
            selectedColor: WorkspaceTheme.ink,
            onTap: () {
              if (Scaffold.of(context).isDrawerOpen) {
                Navigator.of(context).pop();
              }
              context.go(item.path);
            },
          ),
        ),
      );
    }

    return SafeArea(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(28, 26, 24, 30),
            child: Row(
              children: [
                SvgPicture.asset(
                  'assets/logo_mark.svg',
                  width: 34,
                  height: 34,
                  colorFilter: const ColorFilter.mode(
                    Color(0xFF555298),
                    BlendMode.srcIn,
                  ),
                ),
                const SizedBox(width: 10),
                const Text(
                  Brand.name,
                  style: TextStyle(
                    fontSize: 27,
                    fontWeight: FontWeight.w600,
                    letterSpacing: -1,
                  ),
                ),
              ],
            ),
          ),
          Expanded(
            child: ListView(
              padding: EdgeInsets.zero,
              children: [for (final item in mainItems) destination(item)],
            ),
          ),
          for (final item in items.where((i) => i.path == '/settings'))
            destination(item),
          Padding(
            padding: const EdgeInsets.fromLTRB(24, 20, 24, 24),
            child: Column(
              children: [
                Container(
                  width: 52,
                  height: 52,
                  decoration: BoxDecoration(
                    gradient: const LinearGradient(
                      colors: [Color(0xFFE5DDFF), Color(0xFFB9A5EA)],
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                    ),
                    borderRadius: BorderRadius.circular(18),
                    boxShadow: const [
                      BoxShadow(
                        color: Color(0x196D51AD),
                        blurRadius: 16,
                        offset: Offset(0, 6),
                      ),
                    ],
                  ),
                  child: const Icon(
                    Icons.auto_awesome_outlined,
                    color: Color(0xFF7453B1),
                    size: 28,
                  ),
                ),
                const SizedBox(height: 12),
                const Text(
                  'Simply connected.',
                  style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
                ),
                const SizedBox(height: 6),
                const Text(
                  'Your subscriptions, billing and\ncustomers. All in one place.',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 11,
                    color: WorkspaceTheme.muted,
                    height: 1.7,
                  ),
                ),
                const SizedBox(height: 14),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: () => context.go('/plans'),
                    style: FilledButton.styleFrom(
                      minimumSize: const Size(0, 36),
                      shape: const StadiumBorder(),
                    ),
                    child: const Text(
                      'Explore Plans',
                      style: TextStyle(fontSize: 11),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _Topbar extends StatelessWidget {
  const _Topbar({required this.user, required this.mobile});
  final AuthUser user;
  final bool mobile;

  @override
  Widget build(BuildContext context) => Padding(
    padding: EdgeInsets.fromLTRB(mobile ? 12 : 24, 20, mobile ? 12 : 24, 12),
    child: Row(
      children: [
        if (mobile)
          IconButton(
            tooltip: 'Open navigation',
            onPressed: () => Scaffold.of(context).openDrawer(),
            icon: const Icon(Icons.menu),
          ),
        if (!mobile)
          SizedBox(
            width: 276,
            child: Autocomplete<NavItem>(
              displayStringForOption: (item) => item.label,
              optionsBuilder: (value) => value.text.trim().isEmpty
                  ? const Iterable<NavItem>.empty()
                  : navItemsFor(user.role).where(
                      (item) => item.label.toLowerCase().contains(
                        value.text.toLowerCase(),
                      ),
                    ),
              onSelected: (item) => context.go(item.path),
              fieldViewBuilder: (context, controller, focus, submit) =>
                  TextField(
                    controller: controller,
                    focusNode: focus,
                    style: const TextStyle(fontSize: 12),
                    decoration: const InputDecoration(
                      hintText: 'Search pages...',
                      hintStyle: TextStyle(
                        fontSize: 11,
                        color: WorkspaceTheme.muted,
                      ),
                      prefixIcon: Icon(Icons.search, size: 20),
                      contentPadding: EdgeInsets.symmetric(
                        horizontal: 16,
                        vertical: 12,
                      ),
                    ),
                    onSubmitted: (_) => submit(),
                  ),
            ),
          ),
        const Spacer(),
        Consumer(
          builder: (context, ref, _) => PopupMenuButton<String>(
            tooltip: 'Account menu',
            onSelected: (value) {
              if (value == 'logout') {
                ref.read(authControllerProvider.notifier).logout();
              } else {
                context.go('/profile');
              }
            },
            itemBuilder: (_) => const [
              PopupMenuItem(value: 'profile', child: Text('My profile')),
              PopupMenuItem(value: 'logout', child: Text('Log out')),
            ],
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 4),
              child: Row(
                children: [
                  CircleAvatar(
                    radius: 18,
                    backgroundColor: const Color(0xFFE9DEF1),
                    child: Text(
                      user.fullName.trim().isEmpty
                          ? 'S'
                          : user.fullName.trim()[0].toUpperCase(),
                      style: const TextStyle(
                        fontSize: 14,
                        color: Color(0xFF71527D),
                      ),
                    ),
                  ),
                  if (!mobile) ...[
                    const SizedBox(width: 10),
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          user.fullName,
                          style: const TextStyle(fontSize: 12),
                        ),
                        Text(
                          user.role.label,
                          style: const TextStyle(
                            fontSize: 10,
                            color: WorkspaceTheme.muted,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(width: 8),
                  ],
                  const Icon(Icons.keyboard_arrow_down, size: 18),
                ],
              ),
            ),
          ),
        ),
      ],
    ),
  );
}
