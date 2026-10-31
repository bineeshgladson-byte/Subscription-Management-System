import 'package:flutter/material.dart';
import '../theme/workspace_theme.dart';

class WorkspaceHeading extends StatelessWidget {
  const WorkspaceHeading({
    super.key,
    required this.title,
    required this.subtitle,
    this.action,
  });
  final String title;
  final String subtitle;
  final Widget? action;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final heading = Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.headlineMedium),
          const SizedBox(height: 4),
          Text(
            subtitle,
            style: const TextStyle(fontSize: 12, color: WorkspaceTheme.muted),
          ),
        ],
      );
      if (constraints.maxWidth < 600) {
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            heading,
            if (action != null) ...[const SizedBox(height: 16), action!],
          ],
        );
      }
      return Row(
        children: [
          Expanded(child: heading),
          ?action,
        ],
      );
    },
  );
}

class MetricTile extends StatelessWidget {
  const MetricTile({super.key, required this.label, required this.value});
  final String label;
  final String value;
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
    decoration: BoxDecoration(
      border: Border.all(color: WorkspaceTheme.border),
      borderRadius: BorderRadius.circular(17),
    ),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: const TextStyle(fontSize: 12, color: WorkspaceTheme.muted),
        ),
        const SizedBox(height: 10),
        Text(
          value,
          style: const TextStyle(
            fontSize: 22,
            fontWeight: FontWeight.w500,
            letterSpacing: -0.6,
          ),
        ),
      ],
    ),
  );
}

class PreviewNotice extends StatelessWidget {
  const PreviewNotice({super.key});
  @override
  Widget build(BuildContext context) => const Row(
    children: [
      Icon(Icons.info_outline, size: 13, color: WorkspaceTheme.muted),
      SizedBox(width: 6),
      Expanded(
        child: Text(
          'Design preview · Sample data. Changes stay in this session.',
          style: TextStyle(fontSize: 10, color: WorkspaceTheme.muted),
        ),
      ),
    ],
  );
}
