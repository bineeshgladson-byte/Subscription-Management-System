import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_exception.dart';
import '../../../core/storage/token_storage.dart';
import '../data/auth_repository.dart';
import '../data/auth_user.dart';

/// Global auth state.
///   AsyncLoading  → app is checking a stored token (splash)
///   AsyncData(null) → logged out
///   AsyncData(user) → logged in
/// The router listens to this provider and redirects on every change.
class AuthController extends AsyncNotifier<AuthUser?> {
  @override
  FutureOr<AuthUser?> build() async {
    final token = await ref.read(tokenStorageProvider).read();
    if (token == null) return null;
    try {
      return await ref.read(authRepositoryProvider).me();
    } on ApiException {
      await ref.read(tokenStorageProvider).clear();
      return null;
    }
  }

  /// Throws ApiException on failure so the login form can show the message.
  Future<void> login(String email, String password) async {
    final (token, user) = await ref
        .read(authRepositoryProvider)
        .login(email.trim(), password);
    await ref.read(tokenStorageProvider).write(token);
    state = AsyncData(user);
  }

  Future<void> logout() async {
    await ref.read(tokenStorageProvider).clear();
    state = const AsyncData(null);
  }

  /// Called by the Dio interceptor on any 401.
  void sessionExpired() {
    if (state.value != null) unawaited(logout());
  }
}

final authControllerProvider = AsyncNotifierProvider<AuthController, AuthUser?>(
  AuthController.new,
);
