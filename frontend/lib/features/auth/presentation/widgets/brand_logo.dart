import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

/// Clean minimalist brand logo widget displaying the logo mark and name.
class BrandLogo extends StatelessWidget {
  const BrandLogo({
    super.key,
    this.size = 36,
    this.showWordmark = true,
    this.isLight = false,
  });

  final double size;
  final bool showWordmark;
  final bool isLight;

  @override
  Widget build(BuildContext context) {
    final fgColor = isLight ? Colors.white : const Color(0xFF09090B);

    return Row(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        SizedBox(
          width: size,
          height: size * 0.58,
          child: SvgPicture.asset(
            'assets/logo_mark.svg',
            colorFilter: ColorFilter.mode(fgColor, BlendMode.srcIn),
          ),
        ),
        if (showWordmark) ...[
          const SizedBox(width: 12),
          Text(
            'Brightloop',
            style: TextStyle(
              fontSize: (size * 0.56).clamp(16.0, 24.0),
              fontWeight: FontWeight.w700,
              letterSpacing: -0.5,
              color: fgColor,
            ),
          ),
        ],
      ],
    );
  }
}
