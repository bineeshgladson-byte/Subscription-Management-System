import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app.dart';

void main() {
  // retry: null → a failed API call (e.g. 403) is shown once instead of being
  // re-sent automatically; screens offer a "Try again" button instead.
  runApp(ProviderScope(retry: (_, _) => null, child: const BrightloopApp()));
}
