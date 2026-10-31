/// Build-time configuration. Pass values with --dart-define, e.g.
///   flutter run -d chrome --dart-define=API_BASE_URL=http://localhost:8080/api/v1
///
/// Android emulator: the host machine is 10.0.2.2, not localhost.
/// Prism mock server (make mock-api): http://localhost:4010
class Env {
  static const apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://localhost:8080/api/v1',
  );
}
