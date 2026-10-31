import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class SplashScreen extends StatelessWidget {
  const SplashScreen({super.key});
  @override
  Widget build(BuildContext context) =>
      const Scaffold(body: Center(child: CircularProgressIndicator()));
}

class ForbiddenScreen extends StatelessWidget {
  const ForbiddenScreen({super.key});
  @override
  Widget build(BuildContext context) => Center(
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        const Icon(Icons.lock_outline, size: 48),
        const SizedBox(height: 12),
        const Text("You don't have access to this page."),
        TextButton(
          onPressed: () => context.go('/dashboard'),
          child: const Text('Go to dashboard'),
        ),
      ],
    ),
  );
}

class NotFoundScreen extends StatelessWidget {
  const NotFoundScreen({super.key, required this.path});
  final String path;
  @override
  Widget build(BuildContext context) => Scaffold(
    body: Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text('No page at $path'),
          TextButton(
            onPressed: () => context.go('/dashboard'),
            child: const Text('Go to dashboard'),
          ),
        ],
      ),
    ),
  );
}

/// Temporary body for screens that haven't been built yet. It tells the
/// developer which endpoints the screen will use. Delete it when you build the screen.
class PlaceholderView extends StatelessWidget {
  const PlaceholderView({
    super.key,
    required this.title,
    required this.description,
    required this.endpoints,
  });

  final String title;
  final String description;
  final List<String> endpoints;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        Text(title, style: t.headlineSmall),
        const SizedBox(height: 8),
        Text(description, style: t.bodyLarge),
        const SizedBox(height: 16),
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Endpoints', style: t.titleSmall),
                const SizedBox(height: 8),
                for (final e in endpoints)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 2),
                    child: Text(
                      e,
                      style: const TextStyle(fontFamily: 'monospace'),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}
