import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Stores the JWT in the platform's secure store (Keychain / Android Keystore).
/// On Flutter web it falls back to browser storage, which is why tokens are short-lived.
class TokenStorage {
  TokenStorage(this._storage);
  final FlutterSecureStorage _storage;
  static const _key = 'access_token';

  String? _cache;

  Future<String?> read() async => _cache ??= await _storage.read(key: _key);

  Future<void> write(String token) async {
    _cache = token;
    await _storage.write(key: _key, value: token);
  }

  Future<void> clear() async {
    _cache = null;
    await _storage.delete(key: _key);
  }
}

final tokenStorageProvider = Provider<TokenStorage>(
  (ref) => TokenStorage(const FlutterSecureStorage()),
);
