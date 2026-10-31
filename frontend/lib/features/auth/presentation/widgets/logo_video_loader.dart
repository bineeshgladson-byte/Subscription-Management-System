import 'package:flutter/material.dart';
import 'package:video_player/video_player.dart';

/// Reusable video player widget for logo_animation.webm during loading states.
class LogoVideoLoader extends StatefulWidget {
  const LogoVideoLoader({
    super.key,
    this.width = 140,
    this.height = 80,
  });

  final double width;
  final double height;

  @override
  State<LogoVideoLoader> createState() => _LogoVideoLoaderState();
}

class _LogoVideoLoaderState extends State<LogoVideoLoader> {
  late VideoPlayerController _controller;
  bool _initialized = false;

  @override
  void initState() {
    super.initState();
    _controller = VideoPlayerController.asset('assets/logo_animation.webm')
      ..setLooping(true)
      ..setVolume(0.0)
      ..initialize().then((_) {
        if (mounted) {
          setState(() => _initialized = true);
          _controller.play();
        }
      }).catchError((_) {
        // Silently catch in case asset format is unsupported on specific runner
      });
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (!_initialized) {
      return SizedBox(
        width: widget.width,
        height: widget.height,
      );
    }

    return SizedBox(
      width: widget.width,
      height: widget.height,
      child: Center(
        child: AspectRatio(
          aspectRatio: _controller.value.aspectRatio > 0
              ? _controller.value.aspectRatio
              : (widget.width / widget.height),
          child: VideoPlayer(_controller),
        ),
      ),
    );
  }
}
