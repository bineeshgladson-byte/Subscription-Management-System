import 'package:dio/dio.dart';

/// Every API failure is turned into an ApiException, so screens only ever
/// handle one error type. `details` holds per-field messages for forms.
class ApiException implements Exception {
  ApiException({
    required this.code,
    required this.message,
    this.statusCode,
    this.details = const {},
  });

  final String code;
  final String message;
  final int? statusCode;
  final Map<String, String> details;

  bool get isUnauthenticated => statusCode == 401;
  bool get isForbidden => statusCode == 403;
  bool get isValidation => code == 'VALIDATION_ERROR';
  bool get isNetwork => code == 'NETWORK_ERROR';

  /// The message for a specific form field, if the server sent one.
  String? fieldError(String field) => details[field];

  factory ApiException.fromDio(DioException e) {
    final res = e.response;
    final data = res?.data;
    if (data is Map && data['error'] is Map) {
      final err = data['error'] as Map;
      final rawDetails = err['details'];
      return ApiException(
        code: err['code']?.toString() ?? 'UNKNOWN',
        message: err['message']?.toString() ?? 'Request failed',
        statusCode: res?.statusCode,
        details: rawDetails is Map
            ? rawDetails.map((k, v) => MapEntry(k.toString(), v.toString()))
            : const {},
      );
    }
    if (res == null) {
      return ApiException(
        code: 'NETWORK_ERROR',
        message:
            'Cannot reach the server. Check your connection and that the API is running.',
      );
    }
    return ApiException(
      code: 'HTTP_${res.statusCode}',
      message: 'Request failed (${res.statusCode})',
      statusCode: res.statusCode,
    );
  }

  @override
  String toString() => message;
}
