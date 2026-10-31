import 'package:flutter/material.dart';

/// Reference workspace palette, separate from the authentication UI.
abstract final class WorkspaceTheme {
  static const background = Color(0xFFE4E1EE);
  static const lavender = Color(0xFFECE8FC);
  static const selected = Color(0xFFD6CEF5);
  static const purple = Color(0xFF8066FF);
  static const ink = Color(0xFF151517);
  static const muted = Color(0xFF707076);
  static const border = Color(0xFFEAEAEA);
  static const soft = Color(0xFFF8F8F8);
  static const green = Color(0xFF389960);

  static ThemeData get data => ThemeData(
    useMaterial3: true,
    fontFamily: 'Poppins',
    scaffoldBackgroundColor: Colors.white,
    colorScheme: ColorScheme.fromSeed(
      seedColor: purple,
      brightness: Brightness.light,
      surface: Colors.white,
      onSurface: ink,
      primary: purple,
      outline: border,
    ),
    dividerTheme: const DividerThemeData(color: border, space: 28),
    textTheme: const TextTheme(
      headlineMedium: TextStyle(
        fontSize: 28,
        fontWeight: FontWeight.w500,
        letterSpacing: -0.9,
      ),
      titleLarge: TextStyle(
        fontSize: 18,
        fontWeight: FontWeight.w500,
        letterSpacing: -0.5,
      ),
      bodyMedium: TextStyle(fontSize: 13, height: 1.6),
      bodySmall: TextStyle(fontSize: 12, color: muted, height: 1.6),
    ),
    inputDecorationTheme: InputDecorationTheme(
      isDense: true,
      filled: true,
      fillColor: Colors.white,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(16),
        borderSide: const BorderSide(color: border),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(16),
        borderSide: const BorderSide(color: border),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(16),
        borderSide: const BorderSide(color: purple),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: ink,
        minimumSize: const Size(0, 46),
        padding: const EdgeInsets.symmetric(horizontal: 18),
        side: const BorderSide(color: purple),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: ink,
        foregroundColor: Colors.white,
        minimumSize: const Size(0, 46),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      ),
    ),
    cardTheme: CardThemeData(
      color: Colors.white,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(18),
        side: const BorderSide(color: border),
      ),
    ),
    popupMenuTheme: const PopupMenuThemeData(
      color: Colors.white,
      surfaceTintColor: Colors.transparent,
    ),
  );
}
