import 'package:flutter/material.dart';
import 'login_screen.dart';

/// RegisterScreen directs users to the unified authentication screen
/// with the "Create Account" tab preselected.
class RegisterScreen extends StatelessWidget {
  const RegisterScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const LoginScreen(initialMode: AuthMode.signUp);
  }
}
