import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../theme/workspace_theme.dart';

/// Shared visual language for the customer workspace.
class CustomerPanel extends StatelessWidget {
  const CustomerPanel({super.key, required this.child, this.soft = false});
  final Widget child;
  final bool soft;

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(24),
    decoration: BoxDecoration(
      color: soft ? WorkspaceTheme.soft : Colors.white,
      borderRadius: BorderRadius.circular(22),
      border: Border.all(color: WorkspaceTheme.border),
    ),
    child: child,
  );
}

class CustomerIcon extends StatelessWidget {
  const CustomerIcon(this.icon, {super.key});
  final IconData icon;
  @override
  Widget build(BuildContext context) => Container(
    width: 48,
    height: 48,
    decoration: BoxDecoration(
      color: WorkspaceTheme.lavender,
      borderRadius: BorderRadius.circular(15),
    ),
    child: Icon(icon, color: WorkspaceTheme.ink, size: 23),
  );
}

class CustomerIntro extends StatelessWidget {
  const CustomerIntro({
    super.key,
    required this.icon,
    required this.eyebrow,
    required this.title,
    required this.message,
    this.action,
  });
  final IconData icon;
  final String eyebrow, title, message;
  final Widget? action;

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(28),
    decoration: BoxDecoration(
      borderRadius: BorderRadius.circular(24),
      gradient: const LinearGradient(
        colors: [WorkspaceTheme.lavender, Color(0xFFF7F4FF)],
      ),
    ),
    child: LayoutBuilder(
      builder: (context, constraints) {
        final copy = Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              eyebrow.toUpperCase(),
              style: const TextStyle(
                fontSize: 11,
                letterSpacing: 1.6,
                fontWeight: FontWeight.w600,
                color: WorkspaceTheme.muted,
              ),
            ),
            const SizedBox(height: 12),
            Text(
              title,
              style: const TextStyle(
                fontSize: 25,
                height: 1.3,
                letterSpacing: -0.7,
                fontWeight: FontWeight.w500,
              ),
            ),
            const SizedBox(height: 10),
            Text(
              message,
              style: const TextStyle(fontSize: 14, color: WorkspaceTheme.muted),
            ),
          ],
        );
        if (constraints.maxWidth < 650 ||
            MediaQuery.textScalerOf(context).scale(1) > 1.5) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              CustomerIcon(icon),
              const SizedBox(height: 20),
              copy,
              if (action != null) ...[const SizedBox(height: 20), action!],
            ],
          );
        }
        return Row(
          children: [
            CustomerIcon(icon),
            const SizedBox(width: 24),
            Expanded(child: copy),
            if (action != null) ...[const SizedBox(width: 24), action!],
          ],
        );
      },
    ),
  );
}

class CustomerSection extends StatelessWidget {
  const CustomerSection({
    super.key,
    required this.title,
    required this.caption,
    required this.child,
  });
  final String title, caption;
  final Widget child;
  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      const SizedBox(height: 28),
      Text(
        title,
        style: const TextStyle(
          fontSize: 20,
          letterSpacing: -0.5,
          fontWeight: FontWeight.w500,
        ),
      ),
      const SizedBox(height: 6),
      Text(caption, style: const TextStyle(color: WorkspaceTheme.muted)),
      const SizedBox(height: 18),
      child,
    ],
  );
}

class CustomerEmpty extends StatelessWidget {
  const CustomerEmpty({
    super.key,
    required this.icon,
    required this.title,
    required this.message,
    this.browse = false,
  });
  final IconData icon;
  final String title, message;
  final bool browse;
  @override
  Widget build(BuildContext context) => CustomerPanel(
    child: Padding(
      padding: const EdgeInsets.symmetric(vertical: 32),
      child: Column(
        children: [
          CustomerIcon(icon),
          const SizedBox(height: 20),
          Text(
            title,
            textAlign: TextAlign.center,
            style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
          ),
          const SizedBox(height: 10),
          ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 440),
            child: Text(
              message,
              textAlign: TextAlign.center,
              style: const TextStyle(color: WorkspaceTheme.muted),
            ),
          ),
          if (browse) ...[
            const SizedBox(height: 24),
            FilledButton.icon(
              onPressed: () => context.go('/plans'),
              icon: const Icon(Icons.arrow_forward_rounded, size: 18),
              iconAlignment: IconAlignment.end,
              label: const Text('Explore plans'),
            ),
          ],
        ],
      ),
    ),
  );
}

class CustomerRecord extends StatelessWidget {
  const CustomerRecord({
    super.key,
    required this.icon,
    required this.title,
    required this.reference,
    required this.status,
    required this.facts,
    this.action,
  });
  final IconData icon;
  final String title, reference;
  final Widget status;
  final List<(String, String)> facts;
  final Widget? action;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 16),
    child: CustomerPanel(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              CustomerIcon(icon),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: const TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      reference,
                      style: const TextStyle(
                        fontSize: 12,
                        color: WorkspaceTheme.muted,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Align(alignment: Alignment.centerLeft, child: status),
          const Divider(height: 32),
          LayoutBuilder(
            builder: (context, c) {
              final count = c.maxWidth >= 800
                  ? 3
                  : c.maxWidth >= 450
                  ? 2
                  : 1;
              return Wrap(
                spacing: 24,
                runSpacing: 20,
                children: [
                  for (final (label, value) in facts)
                    SizedBox(
                      width: (c.maxWidth - (count - 1) * 24) / count,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            label,
                            style: const TextStyle(
                              color: WorkspaceTheme.muted,
                              fontSize: 12,
                            ),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            value,
                            style: const TextStyle(
                              fontSize: 15,
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                        ],
                      ),
                    ),
                ],
              );
            },
          ),
          if (action != null) ...[
            const SizedBox(height: 20),
            Align(alignment: Alignment.centerRight, child: action!),
          ],
        ],
      ),
    ),
  );
}
