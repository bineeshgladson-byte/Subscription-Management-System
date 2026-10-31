import 'dart:async';

import 'package:flutter/material.dart';

import '../network/api_exception.dart';
import '../theme/workspace_theme.dart';
import 'workspace_widgets.dart';

/// Shared building blocks for every data screen, so all of them look and
/// behave the same: heading + toolbar + content + pager, consistent loading,
/// empty and error states, and forms that show server-side field errors.

/// Scrollable page body with the standard heading.
class PageFrame extends StatelessWidget {
  const PageFrame({
    super.key,
    required this.title,
    required this.subtitle,
    this.action,
    this.toolbar,
    required this.child,
  });
  final String title;
  final String subtitle;
  final Widget? action;
  final Widget? toolbar;
  final Widget child;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, c) {
      final pad = c.maxWidth < 600 ? 16.0 : 28.0;
      return SingleChildScrollView(
        padding: EdgeInsets.fromLTRB(pad, 20, pad, 32),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            WorkspaceHeading(title: title, subtitle: subtitle, action: action),
            if (toolbar != null) ...[const SizedBox(height: 18), toolbar!],
            const SizedBox(height: 18),
            child,
          ],
        ),
      );
    },
  );
}

/// Search box that waits until typing pauses before calling [onChanged].
class SearchBox extends StatefulWidget {
  const SearchBox({super.key, required this.hint, required this.onChanged});
  final String hint;
  final ValueChanged<String> onChanged;
  @override
  State<SearchBox> createState() => _SearchBoxState();
}

class _SearchBoxState extends State<SearchBox> {
  Timer? _debounce;
  @override
  void dispose() {
    _debounce?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => SizedBox(
    width: 320,
    child: TextField(
      decoration: InputDecoration(
        hintText: widget.hint,
        prefixIcon: const Icon(Icons.search, size: 18),
      ),
      onChanged: (v) {
        _debounce?.cancel();
        _debounce = Timer(
          const Duration(milliseconds: 350),
          () => widget.onChanged(v.trim()),
        );
      },
    ),
  );
}

/// A compact dropdown filter: value null = "All".
class FilterDropdown<T> extends StatelessWidget {
  const FilterDropdown({
    super.key,
    required this.label,
    required this.value,
    required this.options,
    required this.onChanged,
    this.allLabel = 'All',
  });
  final String label;
  final T? value;
  final Map<T, String> options;
  final ValueChanged<T?> onChanged;
  final String allLabel;
  @override
  Widget build(BuildContext context) => SizedBox(
    width: 190,
    child: DropdownButtonFormField<T?>(
      initialValue: value,
      isExpanded: true,
      decoration: InputDecoration(labelText: label),
      items: [
        DropdownMenuItem<T?>(value: null, child: Text(allLabel)),
        for (final e in options.entries)
          DropdownMenuItem<T?>(value: e.key, child: Text(e.value)),
      ],
      onChanged: onChanged,
    ),
  );
}

class Toolbar extends StatelessWidget {
  const Toolbar({super.key, required this.children});
  final List<Widget> children;
  @override
  Widget build(BuildContext context) => Wrap(
    spacing: 12,
    runSpacing: 12,
    crossAxisAlignment: WrapCrossAlignment.center,
    children: children,
  );
}

/// "Showing 1–20 of 134" with previous / next.
class Pager extends StatelessWidget {
  const Pager({
    super.key,
    required this.page,
    required this.pageSize,
    required this.total,
    required this.onPage,
  });
  final int page, pageSize, total;
  final ValueChanged<int> onPage;
  @override
  Widget build(BuildContext context) {
    if (total <= pageSize && page == 1) {
      return Padding(
        padding: const EdgeInsets.only(top: 8),
        child: Text(
          '$total total',
          style: Theme.of(context).textTheme.bodySmall,
        ),
      );
    }
    final from = total == 0 ? 0 : (page - 1) * pageSize + 1;
    final to = (page * pageSize).clamp(0, total);
    return Row(
      children: [
        Text(
          'Showing $from–$to of $total',
          style: Theme.of(context).textTheme.bodySmall,
        ),
        const Spacer(),
        IconButton(
          tooltip: 'Previous page',
          onPressed: page > 1 ? () => onPage(page - 1) : null,
          icon: const Icon(Icons.chevron_left),
        ),
        IconButton(
          tooltip: 'Next page',
          onPressed: to < total ? () => onPage(page + 1) : null,
          icon: const Icon(Icons.chevron_right),
        ),
      ],
    );
  }
}

class EmptyState extends StatelessWidget {
  const EmptyState({
    super.key,
    required this.icon,
    required this.title,
    required this.message,
    this.action,
  });
  final IconData icon;
  final String title, message;
  final Widget? action;
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(vertical: 48, horizontal: 24),
    decoration: BoxDecoration(
      border: Border.all(color: WorkspaceTheme.border),
      borderRadius: BorderRadius.circular(18),
    ),
    child: Column(
      children: [
        Icon(icon, size: 36, color: WorkspaceTheme.muted),
        const SizedBox(height: 12),
        Text(
          title,
          style: Theme.of(context).textTheme.titleLarge,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 6),
        Text(
          message,
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.bodySmall,
        ),
        if (action != null) ...[const SizedBox(height: 16), action!],
      ],
    ),
  );
}

/// One row in a list: title, subtitle, pills and a trailing area.
class ResourceTile extends StatelessWidget {
  const ResourceTile({
    super.key,
    required this.title,
    this.subtitle,
    this.pills = const [],
    this.trailing,
    this.onTap,
    this.leading,
  });
  final String title;
  final String? subtitle;
  final List<Widget> pills;
  final Widget? trailing;
  final Widget? leading;
  final VoidCallback? onTap;
  @override
  Widget build(BuildContext context) => Card(
    margin: const EdgeInsets.only(bottom: 10),
    clipBehavior: Clip.antiAlias,
    child: InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
        child: Row(
          children: [
            if (leading != null) ...[leading!, const SizedBox(width: 14)],
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  if (subtitle != null) ...[
                    const SizedBox(height: 2),
                    Text(
                      subtitle!,
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                  if (pills.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Wrap(spacing: 6, runSpacing: 6, children: pills),
                  ],
                ],
              ),
            ),
            ?trailing,
          ],
        ),
      ),
    ),
  );
}

/// Small rounded label. tone: neutral | good | warn | bad | brand
class Pill extends StatelessWidget {
  const Pill(this.text, {super.key, this.tone = 'neutral'});
  final String text;
  final String tone;
  @override
  Widget build(BuildContext context) {
    final (bg, fg) = switch (tone) {
      'good' => (const Color(0xFFE3F4EA), WorkspaceTheme.green),
      'warn' => (const Color(0xFFFFF1DC), const Color(0xFF9A5B00)),
      'bad' => (const Color(0xFFFBE4E2), const Color(0xFFB3261E)),
      'brand' => (WorkspaceTheme.lavender, WorkspaceTheme.purple),
      _ => (WorkspaceTheme.soft, WorkspaceTheme.muted),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 3),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(99),
      ),
      child: Text(
        text,
        style: TextStyle(fontSize: 11, color: fg, fontWeight: FontWeight.w500),
      ),
    );
  }
}

/// Label / value pair used in detail dialogs.
class InfoRow extends StatelessWidget {
  const InfoRow(this.label, this.value, {super.key});
  final String label, value;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 4),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 150,
          child: Text(label, style: Theme.of(context).textTheme.bodySmall),
        ),
        Expanded(child: Text(value)),
      ],
    ),
  );
}

void showMessage(BuildContext context, String message, {bool error = false}) {
  ScaffoldMessenger.of(context)
    ..hideCurrentSnackBar()
    ..showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: error ? const Color(0xFFB3261E) : WorkspaceTheme.ink,
        behavior: SnackBarBehavior.floating,
      ),
    );
}

String errorText(Object e) =>
    e is ApiException ? e.message : 'Something went wrong';

/// Runs [action]; on failure shows the API's message. Returns true on success.
Future<bool> runAction(
  BuildContext context,
  Future<void> Function() action, {
  String? success,
}) async {
  try {
    await action();
    if (context.mounted && success != null) showMessage(context, success);
    return true;
  } catch (e) {
    if (context.mounted) {
      final details = e is ApiException && e.details.isNotEmpty
          ? '\n${e.details.values.join('\n')}'
          : '';
      showMessage(context, '${errorText(e)}$details', error: true);
    }
    return false;
  }
}

Future<bool> confirmAction(
  BuildContext context, {
  required String title,
  required String message,
  required String confirmLabel,
  bool danger = false,
}) async {
  final ok = await showDialog<bool>(
    context: context,
    builder: (c) => AlertDialog(
      title: Text(title),
      content: Text(message),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(c, false),
          child: const Text('Cancel'),
        ),
        FilledButton(
          style: danger
              ? FilledButton.styleFrom(backgroundColor: const Color(0xFFB3261E))
              : null,
          onPressed: () => Navigator.pop(c, true),
          child: Text(confirmLabel),
        ),
      ],
    ),
  );
  return ok ?? false;
}

/// Field error from the server for [key] (also matches "key[...]" style keys).
String? fieldError(Map<String, String> errors, String key) {
  if (errors[key] != null) return errors[key];
  for (final e in errors.entries) {
    if (e.key.startsWith('$key[') || e.key.startsWith('$key.')) return e.value;
  }
  return null;
}

/// A dialog form that calls the API on save and shows server errors under
/// the matching fields. [fields] is rebuilt with the latest errors.
Future<T?> showApiForm<T>(
  BuildContext context, {
  required String title,
  required List<Widget> Function(
    BuildContext context,
    StateSetter setState,
    Map<String, String> errors,
  )
  fields,
  required Future<T> Function() onSave,
  String saveLabel = 'Save',
  double width = 520,
}) async {
  ModalRoute<T>? dialogRoute;
  final result = await showDialog<T>(
    context: context,
    barrierDismissible: false,
    builder: (context) {
      dialogRoute = ModalRoute.of<T>(context);
      return _ApiFormDialog<T>(
        title: title,
        fields: fields,
        onSave: onSave,
        saveLabel: saveLabel,
        width: width,
      );
    },
  );
  // A pop resolves showDialog before its exit animation removes the fields.
  // Callers may dispose their controllers as soon as this helper returns.
  await dialogRoute?.completed;
  return result;
}

class _ApiFormDialog<T> extends StatefulWidget {
  const _ApiFormDialog({
    required this.title,
    required this.fields,
    required this.onSave,
    required this.saveLabel,
    required this.width,
  });
  final String title, saveLabel;
  final double width;
  final List<Widget> Function(BuildContext, StateSetter, Map<String, String>)
  fields;
  final Future<T> Function() onSave;
  @override
  State<_ApiFormDialog<T>> createState() => _ApiFormDialogState<T>();
}

class _ApiFormDialogState<T> extends State<_ApiFormDialog<T>> {
  final _form = GlobalKey<FormState>();
  Map<String, String> _errors = {};
  String? _message;
  bool _busy = false;

  Future<void> _save() async {
    if (!_form.currentState!.validate()) return;
    setState(() {
      _busy = true;
      _errors = {};
      _message = null;
    });
    try {
      final result = await widget.onSave();
      if (mounted) Navigator.pop(context, result);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _message = errorText(e);
        _errors = e is ApiException ? e.details : {};
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final w = MediaQuery.sizeOf(context).width;
    return AlertDialog(
      title: Text(widget.title),
      insetPadding: EdgeInsets.symmetric(
        horizontal: w < 600 ? 12 : 40,
        vertical: 24,
      ),
      content: SizedBox(
        width: widget.width,
        child: Form(
          key: _form,
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                ...widget
                    .fields(context, setState, _errors)
                    .expand((f) => [f, const SizedBox(height: 12)]),
                if (_message != null)
                  Text(
                    [
                      _message!,
                      ..._errors.entries
                          .where((e) => e.key.contains('['))
                          .map((e) => '${e.key}: ${e.value}'),
                    ].join('\n'),
                    style: const TextStyle(
                      color: Color(0xFFB3261E),
                      fontSize: 12,
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.pop(context),
          child: const Text('Cancel'),
        ),
        FilledButton(
          onPressed: _busy ? null : _save,
          child: _busy
              ? const SizedBox.square(
                  dimension: 16,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    color: Colors.white,
                  ),
                )
              : Text(widget.saveLabel),
        ),
      ],
    );
  }
}

/// Text field with the server error (if any) under it.
Widget formField(
  TextEditingController c,
  String label, {
  Map<String, String> errors = const {},
  String? errorKey,
  String? hint,
  String? Function(String?)? validator,
  TextInputType? keyboard,
  int maxLines = 1,
  bool obscure = false,
}) => TextFormField(
  controller: c,
  obscureText: obscure,
  keyboardType: keyboard,
  maxLines: obscure ? 1 : maxLines,
  decoration: InputDecoration(
    labelText: label,
    hintText: hint,
    errorText: errorKey == null ? null : fieldError(errors, errorKey),
  ),
  validator: validator,
);

/// "999", "999.5", "999.50" → valid; empty allowed unless [required].
String? Function(String?) moneyValidator({
  bool required = true,
  bool allowNegative = false,
}) => (v) {
  final s = (v ?? '').trim();
  if (s.isEmpty) return required ? 'Required' : null;
  final re = allowNegative
      ? RegExp(r'^-?\d+(\.\d{1,2})?$')
      : RegExp(r'^\d+(\.\d{1,2})?$');
  return re.hasMatch(s) ? null : 'Enter an amount like 999.00';
};

String? requiredValidator(String? v) =>
    (v ?? '').trim().isEmpty ? 'Required' : null;

String? intValidator(String? v, {int min = 1, bool required = true}) {
  final s = (v ?? '').trim();
  if (s.isEmpty) return required ? 'Required' : null;
  final n = int.tryParse(s);
  return n == null || n < min ? 'Enter a whole number ≥ $min' : null;
}

/// Date text field (YYYY-MM-DD) with a calendar button.
class DateField extends StatelessWidget {
  const DateField({
    super.key,
    required this.controller,
    required this.label,
    this.errorText,
    this.required = false,
  });
  final TextEditingController controller;
  final String label;
  final String? errorText;
  final bool required;
  @override
  Widget build(BuildContext context) => TextFormField(
    controller: controller,
    decoration: InputDecoration(
      labelText: label,
      hintText: 'YYYY-MM-DD',
      errorText: errorText,
      suffixIcon: IconButton(
        tooltip: 'Pick a date',
        icon: const Icon(Icons.calendar_today_outlined, size: 18),
        onPressed: () async {
          final now = DateTime.now();
          final picked = await showDatePicker(
            context: context,
            initialDate: DateTime.tryParse(controller.text) ?? now,
            firstDate: DateTime(now.year - 5),
            lastDate: DateTime(now.year + 20),
          );
          if (picked != null) {
            controller.text =
                '${picked.year.toString().padLeft(4, '0')}-${picked.month.toString().padLeft(2, '0')}-${picked.day.toString().padLeft(2, '0')}';
          }
        },
      ),
    ),
    validator: (v) {
      final s = (v ?? '').trim();
      if (s.isEmpty) return required ? 'Required' : null;
      return RegExp(r'^\d{4}-\d{2}-\d{2}$').hasMatch(s) &&
              DateTime.tryParse(s) != null
          ? null
          : 'Use YYYY-MM-DD';
    },
  );
}
