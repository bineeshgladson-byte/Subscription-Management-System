import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:go_router/go_router.dart';

import '../../../core/network/api_exception.dart';
import '../../../core/utils/validators.dart';
import '../application/auth_controller.dart';
import '../data/auth_repository.dart';
import 'widgets/brand_logo.dart';
import 'widgets/logo_video_loader.dart';

enum AuthMode { signIn, signUp }

/// Interactive split-screen authentication screen with a smoothly sliding
/// pitch-black overlay container and dynamic spinning & scaling logo animations.
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key, this.initialMode = AuthMode.signIn});

  final AuthMode initialMode;

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen>
    with SingleTickerProviderStateMixin {
  late AuthMode _mode;

  // Slide Animation Controller
  late final AnimationController _slideController;
  late final Animation<double> _slideAnimation;

  // Forms
  final _signInForm = GlobalKey<FormState>();
  final _signUpForm = GlobalKey<FormState>();

  // Sign In controllers
  final _loginEmail = TextEditingController();
  final _loginPassword = TextEditingController();
  bool _loginObscure = true;
  bool _rememberMe = true;

  // Register controllers
  final _regName = TextEditingController();
  final _regEmail = TextEditingController();
  final _regPhone = TextEditingController();
  final _regPassword = TextEditingController();
  bool _regObscure = true;

  // Feedback State
  bool _busy = false;
  String? _error;
  Map<String, String> _fieldErrors = {};

  @override
  void initState() {
    super.initState();
    _mode = widget.initialMode;

    _slideController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 700),
      value: widget.initialMode == AuthMode.signUp ? 1.0 : 0.0,
    );

    _slideAnimation = CurvedAnimation(
      parent: _slideController,
      curve: Curves.easeInOutCubic,
    );
  }

  @override
  void dispose() {
    _slideController.dispose();
    _loginEmail.dispose();
    _loginPassword.dispose();
    _regName.dispose();
    _regEmail.dispose();
    _regPhone.dispose();
    _regPassword.dispose();
    super.dispose();
  }

  void _toggleMode() {
    setState(() {
      _mode = _mode == AuthMode.signIn ? AuthMode.signUp : AuthMode.signIn;
      _error = null;
      _fieldErrors = {};
    });

    if (_mode == AuthMode.signUp) {
      _slideController.forward();
    } else {
      _slideController.reverse();
    }
  }

  Future<void> _submitSignIn() async {
    if (!_signInForm.currentState!.validate()) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref
          .read(authControllerProvider.notifier)
          .login(_loginEmail.text, _loginPassword.text);
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } catch (_) {
      setState(
        () => _error = 'An unexpected error occurred. Please try again.',
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _submitSignUp() async {
    setState(() => _fieldErrors = {});
    if (!_signUpForm.currentState!.validate()) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref
          .read(authRepositoryProvider)
          .register(
            fullName: _regName.text.trim(),
            email: _regEmail.text.trim(),
            password: _regPassword.text,
            phone: _regPhone.text.trim(),
          );
      await ref
          .read(authControllerProvider.notifier)
          .login(_regEmail.text, _regPassword.text);
    } on ApiException catch (e) {
      setState(() {
        _error = e.message;
        _fieldErrors = e.details;
      });
    } catch (_) {
      setState(() => _error = 'Registration failed. Please try again.');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      body: LayoutBuilder(
        builder: (context, constraints) {
          final isWide = constraints.maxWidth >= 900;

          if (!isWide) {
            return _buildMobileLayout();
          }

          final halfWidth = constraints.maxWidth / 2;

          return Stack(
            children: [
              // 1. Background White Base containing both forms
              Positioned.fill(
                child: Row(
                  children: [
                    // Left half: Register Form (visible when black panel slides to the right)
                    SizedBox(
                      width: halfWidth,
                      height: double.infinity,
                      child: Center(
                        child: SingleChildScrollView(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 48,
                            vertical: 36,
                          ),
                          child: ConstrainedBox(
                            constraints: const BoxConstraints(maxWidth: 420),
                            child: _buildRegisterFormContent(),
                          ),
                        ),
                      ),
                    ),

                    // Right half: Sign In Form (visible when black panel is on the left)
                    SizedBox(
                      width: halfWidth,
                      height: double.infinity,
                      child: Center(
                        child: SingleChildScrollView(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 48,
                            vertical: 36,
                          ),
                          child: ConstrainedBox(
                            constraints: const BoxConstraints(maxWidth: 420),
                            child: _buildSignInFormContent(),
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),

              // 2. Sliding Pitch-Black Overlay Container with Animated Logo
              AnimatedBuilder(
                animation: _slideAnimation,
                builder: (context, child) {
                  final progress = _slideAnimation.value; // 0.0 -> 1.0
                  final leftPos = progress * halfWidth;

                  // Dynamic logo transformations
                  final rotationAngle =
                      progress * 2 * math.pi; // 360 degree spin
                  final scaleMultiplier =
                      1.0 +
                      0.22 * math.sin(progress * math.pi); // 1.0 -> 1.22 -> 1.0

                  // Smooth text crossfade at mid-point
                  final isRightSide = progress >= 0.5;
                  final midProximity = (1.0 - 2.5 * (progress - 0.5).abs())
                      .clamp(0.0, 1.0);
                  final textOpacity = (1.0 - midProximity).clamp(0.0, 1.0);

                  return Positioned(
                    left: leftPos,
                    top: 0,
                    bottom: 0,
                    width: halfWidth,
                    child: Container(
                      decoration: BoxDecoration(
                        color: Colors.black,
                        boxShadow: [
                          BoxShadow(
                            color: Colors.black.withValues(alpha: 0.35),
                            blurRadius: 32,
                            spreadRadius: 6,
                            offset: Offset(progress < 0.5 ? 6 : -6, 0),
                          ),
                        ],
                      ),
                      child: Center(
                        child: SingleChildScrollView(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 32,
                            vertical: 40,
                          ),
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            crossAxisAlignment: CrossAxisAlignment.center,
                            children: [
                              // Animated Logo: Rotates and scales during transit, or plays video animation on loading
                              if (_busy)
                                const LogoVideoLoader(width: 150, height: 85)
                              else
                                Transform.scale(
                                  scale: scaleMultiplier,
                                  child: Transform.rotate(
                                    angle: rotationAngle,
                                    child: SizedBox(
                                      width: 140,
                                      height: 80,
                                      child: SvgPicture.asset(
                                        'assets/logo_mark.svg',
                                        colorFilter: const ColorFilter.mode(
                                          Colors.white,
                                          BlendMode.srcIn,
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                              const SizedBox(height: 20),

                              // Brand Name
                              const Text(
                                'Brightloop',
                                style: TextStyle(
                                  color: Colors.white,
                                  fontSize: 34,
                                  fontWeight: FontWeight.w700,
                                  letterSpacing: -0.8,
                                ),
                              ),

                              if (_busy) ...[
                                const SizedBox(height: 24),
                                Row(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    const SizedBox.square(
                                      dimension: 14,
                                      child: CircularProgressIndicator(
                                        strokeWidth: 2,
                                        color: Colors.white,
                                      ),
                                    ),
                                    const SizedBox(width: 12),
                                    Text(
                                      _mode == AuthMode.signIn
                                          ? 'Signing in to Brightloop...'
                                          : 'Creating your account...',
                                      style: const TextStyle(
                                        color: Colors.white70,
                                        fontSize: 14,
                                        fontWeight: FontWeight.w500,
                                        letterSpacing: 0.2,
                                      ),
                                    ),
                                  ],
                                ),
                              ] else ...[
                                const SizedBox(height: 36),

                                // White prompt & link with smooth mid-flight cross-fade
                                Opacity(
                                  opacity: textOpacity,
                                  child: Column(
                                    mainAxisSize: MainAxisSize.min,
                                    children: [
                                      Text(
                                        !isRightSide
                                            ? "Don't have an account?"
                                            : 'Already have an account?',
                                        style: TextStyle(
                                          color: Colors.white.withValues(
                                            alpha: 0.75,
                                          ),
                                          fontSize: 14,
                                          fontWeight: FontWeight.w400,
                                        ),
                                      ),
                                      const SizedBox(height: 10),
                                      InkWell(
                                        onTap: _toggleMode,
                                        borderRadius: BorderRadius.circular(6),
                                        child: Padding(
                                          padding: const EdgeInsets.symmetric(
                                            horizontal: 16,
                                            vertical: 8,
                                          ),
                                          child: Text(
                                            !isRightSide
                                                ? 'Register'
                                                : 'Sign In',
                                            style: const TextStyle(
                                              color: Colors.white,
                                              fontSize: 18,
                                              fontWeight: FontWeight.w700,
                                              letterSpacing: 0.3,
                                              decoration:
                                                  TextDecoration.underline,
                                              decorationColor: Colors.white,
                                              decorationThickness: 1.5,
                                            ),
                                          ),
                                        ),
                                      ),
                                    ],
                                  ),
                                ),
                              ],
                            ],
                          ),
                        ),
                      ),
                    ),
                  );
                },
              ),
            ],
          );
        },
      ),
    );
  }

  // --- Right-side Sign In Form ---
  Widget _buildSignInFormContent() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        const Text(
          'Sign In',
          style: TextStyle(
            fontSize: 32,
            fontWeight: FontWeight.w800,
            letterSpacing: -0.8,
            color: Color(0xFF09090B),
          ),
        ),
        const SizedBox(height: 8),
        const Text(
          'Welcome back. Enter your credentials to access your account.',
          style: TextStyle(fontSize: 14, color: Color(0xFF71717A), height: 1.4),
        ),
        const SizedBox(height: 24),

        if (_error != null && _mode == AuthMode.signIn) ...[
          _buildErrorBanner(_error!),
          const SizedBox(height: 16),
        ],

        Form(
          key: _signInForm,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              TextFormField(
                key: const Key('login-email'),
                controller: _loginEmail,
                decoration: const InputDecoration(
                  labelText: 'Email',
                  hintText: 'name@company.com',
                  prefixIcon: Icon(
                    Icons.alternate_email_rounded,
                    size: 18,
                    color: Color(0xFF71717A),
                  ),
                ),
                keyboardType: TextInputType.emailAddress,
                autofillHints: const [AutofillHints.email],
                validator: Validators.email,
              ),
              const SizedBox(height: 14),
              TextFormField(
                key: const Key('login-password'),
                controller: _loginPassword,
                obscureText: _loginObscure,
                decoration: InputDecoration(
                  labelText: 'Password',
                  hintText: '••••••••',
                  prefixIcon: const Icon(
                    Icons.lock_outline_rounded,
                    size: 18,
                    color: Color(0xFF71717A),
                  ),
                  suffixIcon: IconButton(
                    icon: Icon(
                      _loginObscure
                          ? Icons.visibility_outlined
                          : Icons.visibility_off_outlined,
                      size: 18,
                      color: const Color(0xFF71717A),
                    ),
                    onPressed: () =>
                        setState(() => _loginObscure = !_loginObscure),
                  ),
                ),
                validator: (v) => Validators.required(v, 'Password'),
                onFieldSubmitted: (_) => _submitSignIn(),
              ),
              const SizedBox(height: 12),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      SizedBox(
                        width: 22,
                        height: 22,
                        child: Checkbox(
                          value: _rememberMe,
                          onChanged: (val) =>
                              setState(() => _rememberMe = val ?? true),
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(4),
                          ),
                          activeColor: const Color(0xFF09090B),
                        ),
                      ),
                      const SizedBox(width: 8),
                      GestureDetector(
                        onTap: () => setState(() => _rememberMe = !_rememberMe),
                        child: const Text(
                          'Remember me',
                          style: TextStyle(
                            fontSize: 13,
                            color: Color(0xFF3F3F46),
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ),
                    ],
                  ),
                  TextButton(
                    onPressed: () => context.push('/forgot-password'),
                    style: TextButton.styleFrom(
                      padding: EdgeInsets.zero,
                      minimumSize: Size.zero,
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    ),
                    child: const Text(
                      'Forgot password?',
                      style: TextStyle(
                        fontSize: 13,
                        color: Color(0xFF09090B),
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 24),
              FilledButton(
                onPressed: _busy ? null : _submitSignIn,
                child: _busy
                    ? const SizedBox.square(
                        dimension: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Text('Sign In'),
              ),
            ],
          ),
        ),
      ],
    );
  }

  // --- Left-side Register Form ---
  Widget _buildRegisterFormContent() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        const Text(
          'Register',
          style: TextStyle(
            fontSize: 32,
            fontWeight: FontWeight.w800,
            letterSpacing: -0.8,
            color: Color(0xFF09090B),
          ),
        ),
        const SizedBox(height: 8),
        const Text(
          'Create a new account to start managing subscriptions.',
          style: TextStyle(fontSize: 14, color: Color(0xFF71717A), height: 1.4),
        ),
        const SizedBox(height: 24),

        if (_error != null && _mode == AuthMode.signUp) ...[
          _buildErrorBanner(_error!),
          const SizedBox(height: 16),
        ],

        Form(
          key: _signUpForm,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              TextFormField(
                key: const Key('register-name'),
                controller: _regName,
                decoration: InputDecoration(
                  labelText: 'Full Name',
                  hintText: 'Jane Doe',
                  errorText: _fieldErrors['full_name'],
                  prefixIcon: const Icon(
                    Icons.person_outline_rounded,
                    size: 18,
                    color: Color(0xFF71717A),
                  ),
                ),
                validator: (v) => Validators.required(v, 'Full name'),
              ),
              const SizedBox(height: 14),
              TextFormField(
                key: const Key('register-email'),
                controller: _regEmail,
                decoration: InputDecoration(
                  labelText: 'Email',
                  hintText: 'jane@company.com',
                  errorText: _fieldErrors['email'],
                  prefixIcon: const Icon(
                    Icons.alternate_email_rounded,
                    size: 18,
                    color: Color(0xFF71717A),
                  ),
                ),
                keyboardType: TextInputType.emailAddress,
                validator: Validators.email,
              ),
              const SizedBox(height: 14),
              TextFormField(
                key: const Key('register-phone'),
                controller: _regPhone,
                decoration: InputDecoration(
                  labelText: 'Phone (Optional)',
                  hintText: '+1 (555) 000-0000',
                  errorText: _fieldErrors['phone'],
                  prefixIcon: const Icon(
                    Icons.phone_outlined,
                    size: 18,
                    color: Color(0xFF71717A),
                  ),
                ),
                keyboardType: TextInputType.phone,
              ),
              const SizedBox(height: 14),
              TextFormField(
                key: const Key('register-password'),
                controller: _regPassword,
                obscureText: _regObscure,
                decoration: InputDecoration(
                  labelText: 'Password',
                  hintText: 'Minimum 8 characters',
                  errorText: _fieldErrors['password'],
                  prefixIcon: const Icon(
                    Icons.lock_outline_rounded,
                    size: 18,
                    color: Color(0xFF71717A),
                  ),
                  suffixIcon: IconButton(
                    icon: Icon(
                      _regObscure
                          ? Icons.visibility_outlined
                          : Icons.visibility_off_outlined,
                      size: 18,
                      color: const Color(0xFF71717A),
                    ),
                    onPressed: () => setState(() => _regObscure = !_regObscure),
                  ),
                ),
                validator: (v) {
                  final req = Validators.required(v, 'Password');
                  if (req != null) return req;
                  if (v!.length < 8) {
                    return 'Password must be at least 8 characters';
                  }
                  return null;
                },
                onFieldSubmitted: (_) => _submitSignUp(),
              ),
              const SizedBox(height: 24),
              FilledButton(
                onPressed: _busy ? null : _submitSignUp,
                child: _busy
                    ? const SizedBox.square(
                        dimension: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Text('Create Account'),
              ),
            ],
          ),
        ),
      ],
    );
  }

  // Mobile / compact fallback layout
  Widget _buildMobileLayout() {
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Column(
            children: [
              if (_busy)
                const Center(child: LogoVideoLoader(width: 140, height: 80))
              else
                const Center(child: BrandLogo(size: 38, isLight: false)),
              const SizedBox(height: 28),
              if (_mode == AuthMode.signIn)
                _buildSignInFormContent()
              else
                _buildRegisterFormContent(),
              const SizedBox(height: 24),
              InkWell(
                onTap: _toggleMode,
                child: Text(
                  _mode == AuthMode.signIn
                      ? "Don't have an account? Register"
                      : 'Already have an account? Sign In',
                  style: const TextStyle(
                    color: Color(0xFF09090B),
                    fontWeight: FontWeight.w600,
                    decoration: TextDecoration.underline,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildErrorBanner(String message) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: const Color(0xFFFEF2F2),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: const Color(0xFFFCA5A5)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(
            Icons.error_outline_rounded,
            color: Color(0xFFDC2626),
            size: 18,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              style: const TextStyle(
                color: Color(0xFFB91C1C),
                fontSize: 13,
                fontWeight: FontWeight.w500,
                height: 1.3,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
