import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/json.dart';

class StaffMember {
  StaffMember.fromJson(Map<String, dynamic> j)
    : id = toInt(j['id']),
      email = str(j['email']),
      fullName = str(j['full_name']),
      isActive = toBool(j['is_active']),
      lastLoginAt = dateOrNull(j['last_login_at']),
      createdAt = dateOrNull(j['created_at']);
  final int id;
  final String email, fullName;
  final bool isActive;
  final DateTime? lastLoginAt, createdAt;
}

typedef StaffQuery = ({String q, bool? active, int page});

class StaffRepository {
  StaffRepository(this._api);
  final ApiClient _api;

  Future<Paged<StaffMember>> list(StaffQuery q) => _api.getPaged(
    '/staff',
    StaffMember.fromJson,
    query: {
      'q': q.q,
      'is_active': q.active?.toString(),
      'page': q.page,
      'page_size': 20,
      'sort': '-created_at',
    },
  );

  /// Returns whether an invite email was sent.
  Future<bool> create(Map<String, dynamic> body) async {
    final d = await _api.post('/staff', body: body) as Map<String, dynamic>;
    return d['invite_sent'] == true;
  }

  Future<void> update(int id, Map<String, dynamic> body) =>
      _api.put('/staff/$id', body: body);
  Future<void> setActive(int id, bool active) =>
      _api.patch('/staff/$id/status', body: {'is_active': active});
}

final staffRepositoryProvider = Provider(
  (ref) => StaffRepository(ref.watch(apiClientProvider)),
);

final staffProvider = FutureProvider.autoDispose
    .family<Paged<StaffMember>, StaffQuery>(
      (ref, q) => ref.watch(staffRepositoryProvider).list(q),
    );
