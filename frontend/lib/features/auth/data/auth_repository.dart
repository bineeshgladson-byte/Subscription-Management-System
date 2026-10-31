import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import 'auth_user.dart';

/// REFERENCE repository: copy this shape for every feature.
/// Repositories call ApiClient and return typed models. Screens never touch Dio.
class AuthRepository {
  AuthRepository(this._api);
  final ApiClient _api;

  /// Returns (token, user).
  Future<(String, AuthUser)> login(String email, String password) async {
    final data = await _api.post(
      '/auth/login',
      body: {'email': email, 'password': password},
    );
    final map = data as Map<String, dynamic>;
    return (
      map['access_token'] as String,
      AuthUser.fromJson(map['user'] as Map<String, dynamic>),
    );
  }

  Future<void> register({
    required String fullName,
    required String email,
    required String password,
    String? phone,
  }) => _api.post(
    '/auth/register',
    body: {
      'full_name': fullName,
      'email': email,
      'password': password,
      if (phone != null && phone.isNotEmpty) 'phone': phone,
    },
  );

  Future<AuthUser> me() async =>
      AuthUser.fromJson(await _api.get('/auth/me') as Map<String, dynamic>);

  Future<void> forgotPassword(String email) =>
      _api.post('/auth/forgot-password', body: {'email': email});
}

final authRepositoryProvider = Provider<AuthRepository>(
  (ref) => AuthRepository(ref.watch(apiClientProvider)),
);
