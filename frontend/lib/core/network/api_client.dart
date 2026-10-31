import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../features/auth/application/auth_controller.dart';
import '../config/env.dart';
import '../storage/token_storage.dart';
import 'api_exception.dart';

/// One page of a list endpoint ({data: [...], meta: {page, page_size, total}}).
class Paged<T> {
  Paged({
    required this.items,
    required this.page,
    required this.pageSize,
    required this.total,
  });
  final List<T> items;
  final int page;
  final int pageSize;
  final int total;
  bool get hasMore => page * pageSize < total;
}

/// The ONLY class that talks HTTP. Feature repositories call get/post/put/patch/delete
/// and receive the unwrapped `data` from the response envelope.
class ApiClient {
  ApiClient(this._dio);
  final Dio _dio;

  Future<dynamic> get(String path, {Map<String, dynamic>? query}) =>
      _send(() => _dio.get(path, queryParameters: _clean(query)));

  Future<Paged<T>> getPaged<T>(
    String path,
    T Function(Map<String, dynamic>) fromJson, {
    Map<String, dynamic>? query,
  }) async {
    try {
      final res = await _dio.get(path, queryParameters: _clean(query));
      final body = res.data as Map<String, dynamic>;
      final meta = (body['meta'] as Map<String, dynamic>?) ?? const {};
      final items = (body['data'] as List? ?? const [])
          .map((e) => fromJson(e as Map<String, dynamic>))
          .toList();
      return Paged(
        items: items,
        page: meta['page'] as int? ?? 1,
        pageSize: meta['page_size'] as int? ?? items.length,
        total: meta['total'] as int? ?? items.length,
      );
    } on DioException catch (e) {
      throw ApiException.fromDio(e);
    }
  }

  Future<dynamic> post(
    String path, {
    Object? body,
    Map<String, String>? headers,
  }) => _send(
    () => _dio.post(
      path,
      data: body,
      options: Options(headers: headers),
    ),
  );

  Future<dynamic> put(String path, {Object? body}) =>
      _send(() => _dio.put(path, data: body));

  Future<dynamic> patch(String path, {Object? body}) =>
      _send(() => _dio.patch(path, data: body));

  Future<dynamic> delete(String path) => _send(() => _dio.delete(path));

  Future<dynamic> _send(Future<Response<dynamic>> Function() call) async {
    try {
      final res = await call();
      final body = res.data;
      return body is Map<String, dynamic> ? body['data'] : body;
    } on DioException catch (e) {
      throw ApiException.fromDio(e);
    }
  }

  Map<String, dynamic>? _clean(Map<String, dynamic>? q) => q == null
      ? null
      : (Map.of(q)..removeWhere((_, v) => v == null || v == ''));
}

final dioProvider = Provider<Dio>((ref) {
  final dio = Dio(
    BaseOptions(
      baseUrl: Env.apiBaseUrl,
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 20),
      contentType: Headers.jsonContentType,
    ),
  );
  final storage = ref.watch(tokenStorageProvider);

  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) async {
        final token = await storage.read();
        if (token != null) options.headers['Authorization'] = 'Bearer $token';
        handler.next(options);
      },
      onError: (e, handler) {
        final isLogin = e.requestOptions.path.endsWith('/auth/login');
        if (e.response?.statusCode == 401 && !isLogin) {
          // Token expired or revoked: log out; the router then redirects to /login.
          ref.read(authControllerProvider.notifier).sessionExpired();
        }
        handler.next(e);
      },
    ),
  );
  return dio;
});

final apiClientProvider = Provider<ApiClient>(
  (ref) => ApiClient(ref.watch(dioProvider)),
);
