import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/models/role.dart';
import '../../../core/utils/formatters.dart';
import '../../../core/widgets/async_view.dart';
import '../../../core/widgets/ui_kit.dart';
import '../../auth/application/auth_controller.dart';
import '../../taxes/data/taxes_api.dart';
import '../data/products_api.dart';

/// Products and variants. Admin manages; staff see everything except cost/margin.
class ProductsScreen extends ConsumerStatefulWidget {
  const ProductsScreen({super.key});
  @override
  ConsumerState<ProductsScreen> createState() => _ProductsScreenState();
}

class _ProductsScreenState extends ConsumerState<ProductsScreen> {
  ProductQuery _q = (q: '', status: 'active', type: null, page: 1);
  void _set({
    String? q,
    String? status,
    String? type,
    bool clearType = false,
    int page = 1,
  }) => setState(
    () => _q = (
      q: q ?? _q.q,
      status: status ?? _q.status,
      type: clearType ? null : (type ?? _q.type),
      page: page,
    ),
  );

  @override
  Widget build(BuildContext context) {
    final isAdmin = ref.watch(authControllerProvider).value?.role == Role.admin;
    return PageFrame(
      title: 'Products',
      subtitle:
          'What Brightloop sells. Variants change the price, e.g. the Odoo brand adds ₹560.',
      action: isAdmin
          ? FilledButton.icon(
              onPressed: () => _editProduct(null),
              icon: const Icon(Icons.add),
              label: const Text('New product'),
            )
          : null,
      toolbar: Toolbar(
        children: [
          SearchBox(
            hint: 'Search products',
            onChanged: (v) => _set(q: v),
          ),
          FilterDropdown<String>(
            label: 'Type',
            value: _q.type,
            options: productTypes,
            onChanged: (v) => _set(type: v, clearType: v == null),
          ),
          FilterDropdown<String>(
            label: 'Status',
            value: _q.status == 'active' ? null : _q.status,
            allLabel: 'Active',
            options: const {'archived': 'Archived', 'all': 'All'},
            onChanged: (v) => _set(status: v ?? 'active'),
          ),
        ],
      ),
      child: AsyncView(
        value: ref.watch(productsProvider(_q)),
        onRetry: () => ref.invalidate(productsProvider),
        builder: (page) => Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (page.items.isEmpty)
              const EmptyState(
                icon: Icons.inventory_2_outlined,
                title: 'No products',
                message: 'Nothing matches these filters.',
              ),
            for (final p in page.items)
              ResourceTile(
                title: p.name,
                subtitle: [
                  productTypes[p.productType] ?? p.productType,
                  Fmt.inr(p.salesPrice),
                  if (p.costPrice != null)
                    'cost ${Fmt.inr(p.costPrice!)} · margin ${Fmt.inr(p.margin!)}',
                  p.taxLabel ?? 'no default tax',
                ].join(' · '),
                pills: [
                  if (!p.isActive) const Pill('Archived', tone: 'warn'),
                  for (final v in p.variants)
                    Pill(
                      '${v.name} ${Fmt.inr(v.price)}',
                      tone: v.isActive ? 'brand' : 'neutral',
                    ),
                ],
                onTap: () => _detail(p.id, isAdmin),
                trailing: isAdmin
                    ? IconButton(
                        tooltip: 'Edit ${p.name}',
                        icon: const Icon(Icons.edit_outlined),
                        onPressed: () => _editProduct(p),
                      )
                    : null,
              ),
            Pager(
              page: page.page,
              pageSize: page.pageSize,
              total: page.total,
              onPage: (n) => _set(page: n),
            ),
          ],
        ),
      ),
    );
  }

  void _refresh() {
    ref.invalidate(productsProvider);
    ref.invalidate(productDetailProvider);
    ref.invalidate(activeProductsProvider);
    ref.invalidate(taxesProvider);
  }

  Future<void> _editProduct(Product? p) async {
    final taxes = await ref
        .read(activeTaxesProvider.future)
        .catchError((_) => <Tax>[]);
    if (!mounted) return;
    final name = TextEditingController(text: p?.name);
    final sales = TextEditingController(text: p?.salesPrice);
    final cost = TextEditingController(text: p?.costPrice ?? '');
    final notes = TextEditingController(text: p?.description);
    var type = p?.productType ?? 'software';
    int? taxId = p?.taxId;
    final variants =
        <
          (TextEditingController, TextEditingController, TextEditingController)
        >[]; // create only
    final repo = ref.read(productsRepositoryProvider);
    final saved = await showApiForm<bool>(
      context,
      title: p == null ? 'New product' : 'Edit ${p.name}',
      width: 600,
      fields: (_, setState, e) => [
        formField(
          name,
          'Name',
          errors: e,
          errorKey: 'name',
          validator: requiredValidator,
        ),
        DropdownButtonFormField<String>(
          initialValue: type,
          decoration: const InputDecoration(labelText: 'Type'),
          items: [
            for (final t in productTypes.entries)
              DropdownMenuItem(value: t.key, child: Text(t.value)),
          ],
          onChanged: (v) => type = v ?? type,
        ),
        Row(
          children: [
            Expanded(
              child: formField(
                sales,
                'Sales price (₹)',
                errors: e,
                errorKey: 'sales_price',
                keyboard: TextInputType.number,
                validator: moneyValidator(),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: formField(
                cost,
                'Cost price (₹)',
                errors: e,
                errorKey: 'cost_price',
                keyboard: TextInputType.number,
                validator: moneyValidator(required: false),
              ),
            ),
          ],
        ),
        DropdownButtonFormField<int?>(
          initialValue: taxes.any((t) => t.id == taxId) ? taxId : null,
          decoration: InputDecoration(
            labelText: 'Default tax',
            errorText: fieldError(e, 'tax_id'),
          ),
          items: [
            const DropdownMenuItem<int?>(value: null, child: Text('No tax')),
            for (final t in taxes)
              DropdownMenuItem<int?>(
                value: t.id,
                child: Text('${t.name} (${t.rate}%)'),
              ),
          ],
          onChanged: (v) => taxId = v,
        ),
        formField(
          notes,
          'Notes',
          errors: e,
          errorKey: 'description',
          maxLines: 3,
        ),
        if (p == null) ...[
          Text(
            'Variants (optional)',
            style: Theme.of(context).textTheme.titleSmall,
          ),
          for (final (i, v) in variants.indexed)
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                SizedBox(
                  width: 140,
                  child: formField(
                    v.$1,
                    'Attribute',
                    hint: 'Brand',
                    validator: requiredValidator,
                  ),
                ),
                SizedBox(
                  width: 170,
                  child: formField(
                    v.$2,
                    'Name',
                    hint: 'Odoo Brand',
                    validator: requiredValidator,
                    errors: e,
                    errorKey: 'variants[$i]',
                  ),
                ),
                SizedBox(
                  width: 130,
                  child: formField(
                    v.$3,
                    'Extra price',
                    hint: '560',
                    validator: moneyValidator(
                      required: false,
                      allowNegative: true,
                    ),
                  ),
                ),
              ],
            ),
          Align(
            alignment: Alignment.centerLeft,
            child: TextButton.icon(
              onPressed: () => setState(
                () => variants.add((
                  TextEditingController(text: 'Brand'),
                  TextEditingController(),
                  TextEditingController(),
                )),
              ),
              icon: const Icon(Icons.add, size: 18),
              label: const Text('Add variant'),
            ),
          ),
        ],
      ],
      onSave: () async {
        final body = <String, dynamic>{
          'name': name.text.trim(),
          'product_type': type,
          'sales_price': sales.text.trim(),
          if (cost.text.trim().isNotEmpty) 'cost_price': cost.text.trim(),
          'description': notes.text.trim(),
        };
        if (p == null) {
          if (taxId != null) body['tax_id'] = taxId;
          body['variants'] = [
            for (final v in variants)
              {
                'attribute': v.$1.text.trim(),
                'name': v.$2.text.trim(),
                if (v.$3.text.trim().isNotEmpty)
                  'extra_price': v.$3.text.trim(),
              },
          ];
          await repo.create(body);
        } else {
          body['tax_id'] = taxId ?? 0; // 0 removes the default tax
          await repo.update(p.id, body);
        }
        return true;
      },
    );
    for (final t in [
      name,
      sales,
      cost,
      notes,
      for (final v in variants) ...[v.$1, v.$2, v.$3],
    ]) {
      t.dispose();
    }
    if (saved == true && mounted) {
      _refresh();
      showMessage(context, p == null ? 'Product created' : 'Product updated');
    }
  }

  Future<void> _detail(int id, bool isAdmin) => showDialog<void>(
    context: context,
    builder: (dialogContext) => Consumer(
      builder: (context, ref, _) => AlertDialog(
        title: const Text('Product'),
        content: SizedBox(
          width: 560,
          child: AsyncView(
            value: ref.watch(productDetailProvider(id)),
            builder: (p) => SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(p.name, style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 8),
                  InfoRow('Type', productTypes[p.productType] ?? p.productType),
                  InfoRow('Sales price', Fmt.inr(p.salesPrice)),
                  if (p.costPrice != null)
                    InfoRow(
                      'Cost / margin',
                      '${Fmt.inr(p.costPrice!)} / ${Fmt.inr(p.margin!)}',
                    ),
                  InfoRow('Default tax', p.taxLabel ?? 'None'),
                  InfoRow('Status', p.isActive ? 'Active' : 'Archived'),
                  if (p.description != null) InfoRow('Notes', p.description!),
                  const Divider(),
                  Row(
                    children: [
                      Text(
                        'Variants',
                        style: Theme.of(context).textTheme.titleSmall,
                      ),
                      const Spacer(),
                      if (isAdmin)
                        TextButton.icon(
                          onPressed: () => _editVariant(dialogContext, p, null),
                          icon: const Icon(Icons.add, size: 18),
                          label: const Text('Add variant'),
                        ),
                    ],
                  ),
                  if (p.variants.isEmpty)
                    Text(
                      'No variants: customers buy the base product.',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  for (final v in p.variants)
                    ListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text('${v.attribute}: ${v.name}'),
                      subtitle: Text(
                        '${Fmt.inr(v.extraPrice)} extra → ${Fmt.inr(v.price)}${v.sku == null ? '' : ' · SKU ${v.sku}'}',
                      ),
                      trailing: isAdmin
                          ? Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                IconButton(
                                  tooltip: 'Edit variant ${v.name}',
                                  icon: const Icon(Icons.edit_outlined),
                                  onPressed: () =>
                                      _editVariant(dialogContext, p, v),
                                ),
                                Switch(
                                  value: v.isActive,
                                  onChanged: (on) async {
                                    if (await runAction(
                                      dialogContext,
                                      () => ref
                                          .read(productsRepositoryProvider)
                                          .updateVariant(p.id, v.id, {
                                            'is_active': on,
                                          }),
                                      success: on
                                          ? 'Variant reactivated'
                                          : 'Variant deactivated',
                                    )) {
                                      _refresh();
                                    }
                                  },
                                ),
                              ],
                            )
                          : Pill(
                              v.isActive ? 'Active' : 'Inactive',
                              tone: v.isActive ? 'good' : 'neutral',
                            ),
                    ),
                ],
              ),
            ),
          ),
        ),
        actions: [
          if (isAdmin)
            Consumer(
              builder: (context, ref, _) {
                final p = ref.watch(productDetailProvider(id)).value;
                if (p == null) return const SizedBox.shrink();
                return TextButton(
                  onPressed: () async {
                    final repo = ref.read(productsRepositoryProvider);
                    if (p.isActive &&
                        !await confirmAction(
                          dialogContext,
                          title: 'Archive ${p.name}?',
                          message:
                              'It disappears from new plans and quotations. Existing subscriptions are unaffected.',
                          confirmLabel: 'Archive',
                          danger: true,
                        )) {
                      return;
                    }
                    if (!dialogContext.mounted) return;
                    if (await runAction(
                      dialogContext,
                      () => p.isActive
                          ? repo.archive(p.id)
                          : repo.update(p.id, {'status': 'active'}),
                      success: p.isActive
                          ? 'Product archived'
                          : 'Product restored',
                    )) {
                      _refresh();
                    }
                  },
                  child: Text(p.isActive ? 'Archive' : 'Restore'),
                );
              },
            ),
          TextButton(
            onPressed: () => Navigator.pop(dialogContext),
            child: const Text('Close'),
          ),
        ],
      ),
    ),
  );

  Future<void> _editVariant(BuildContext context, Product p, Variant? v) async {
    final attr = TextEditingController(text: v?.attribute ?? 'Brand');
    final name = TextEditingController(text: v?.name);
    final extra = TextEditingController(text: v?.extraPrice ?? '0');
    final sku = TextEditingController(text: v?.sku);
    final repo = ref.read(productsRepositoryProvider);
    final saved = await showApiForm<bool>(
      context,
      title: v == null ? 'Add variant to ${p.name}' : 'Edit variant',
      fields: (_, _, e) => [
        formField(
          attr,
          'Attribute',
          errors: e,
          errorKey: 'attribute',
          hint: 'Brand',
          validator: requiredValidator,
        ),
        formField(
          name,
          'Name',
          errors: e,
          errorKey: 'name',
          hint: 'Odoo Brand',
          validator: requiredValidator,
        ),
        formField(
          extra,
          'Extra price (₹, can be negative)',
          errors: e,
          errorKey: 'extra_price',
          validator: moneyValidator(allowNegative: true),
        ),
        formField(sku, 'SKU (optional)', errors: e, errorKey: 'sku'),
      ],
      onSave: () async {
        final body = {
          'attribute': attr.text.trim(),
          'name': name.text.trim(),
          'extra_price': extra.text.trim(),
          'sku': sku.text.trim(),
        };
        if (v == null) {
          if (body['sku']!.isEmpty) body.remove('sku');
          await repo.addVariant(p.id, body);
        } else {
          await repo.updateVariant(p.id, v.id, body);
        }
        return true;
      },
    );
    for (final t in [attr, name, extra, sku]) {
      t.dispose();
    }
    if (saved == true && mounted) _refresh();
  }
}
