import '../../../core/models/role.dart';

class AuthUser {
  const AuthUser({
    required this.id,
    required this.email,
    required this.fullName,
    required this.role,
    this.customerId,
  });

  final int id;
  final String email;
  final String fullName;
  final Role role;
  final int? customerId;

  factory AuthUser.fromJson(Map<String, dynamic> json) => AuthUser(
    id: json['id'] as int,
    email: json['email'] as String,
    fullName: json['full_name'] as String,
    role: Role.fromJson(json['role'] as String),
    customerId: json['customer_id'] as int?,
  );
}
