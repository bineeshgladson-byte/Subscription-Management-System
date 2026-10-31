import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../features/auth/application/auth_controller.dart';

/// Reuse successful page loads briefly when navigating away and back.
/// Account changes invalidate the data; failures remain retryable.
Future<T> cacheForNavigation<T>(Ref ref, Future<T> Function() load) async {
  ref.watch(
    authControllerProvider.select((auth) => (auth.value?.id, auth.value?.role)),
  );
  final result = await load();
  if (!ref.mounted) return result;
  final link = ref.keepAlive();
  Timer? expiry;
  ref.onCancel(() {
    expiry?.cancel();
    expiry = Timer(const Duration(minutes: 2), link.close);
  });
  ref.onResume(() => expiry?.cancel());
  ref.onDispose(() => expiry?.cancel());
  return result;
}
