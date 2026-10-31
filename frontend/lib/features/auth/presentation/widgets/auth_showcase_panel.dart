import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

/// Minimalist showcase panel: pure black background, centered white logo, and name.
class AuthShowcasePanel extends StatelessWidget {
  const AuthShowcasePanel({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      color: Colors.black,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            // White Brightloop logo mark in the center
            SizedBox(
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
            const SizedBox(height: 22),
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
          ],
        ),
      ),
    );
  }
}
