import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';

/// GET/PUT /auth/me including the customer billing profile.
class Profile {
  Profile.fromJson(Map<String, dynamic> j)
    : email = str(j['email']),
      fullName = str(j['full_name']),
      role = str(j['role']),
      phone = (j['profile'] as Map?)?['phone']?.toString(),
      companyName = (j['profile'] as Map?)?['company_name']?.toString(),
      billingAddress = (j['profile'] as Map?)?['billing_address']?.toString(),
      gstin = (j['profile'] as Map?)?['gstin']?.toString(),
      isCustomer = j['profile'] is Map;
  final String email, fullName, role;
  final String? phone, companyName, billingAddress, gstin;
  final bool isCustomer;
}

class ProfileRepository {
  ProfileRepository(this._api);
  final ApiClient _api;
  Future<Profile> get() async =>
      Profile.fromJson(await _api.get('/auth/me') as Map<String, dynamic>);
  Future<void> update(Map<String, dynamic> body) =>
      _api.put('/auth/me', body: body);

  /// Returns the NEW access token (all other sessions are signed out).
  Future<String> changePassword(String current, String next) async {
    final d =
        await _api.post(
              '/auth/change-password',
              body: {'current_password': current, 'new_password': next},
            )
            as Map<String, dynamic>;
    return d['access_token'] as String;
  }
}

final profileRepositoryProvider = Provider(
  (ref) => ProfileRepository(ref.watch(apiClientProvider)),
);
final profileProvider = FutureProvider.autoDispose<Profile>(
  (ref) => ref.watch(profileRepositoryProvider).get(),
);
