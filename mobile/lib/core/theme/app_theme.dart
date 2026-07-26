// Material 3 主题装配：基于 AppColors 构建 ColorScheme，浅/深双模式。
//
// 字体默认使用系统字体；后续可在 pubspec 引入思源宋体（标题）与思源黑体（正文）
// 资源文件，呼应古籍排印气质。

import 'package:flutter/material.dart';

import 'app_colors.dart';

class AppTheme {
  const AppTheme._();

  /// 浅色主题：宣纸背景 + 朱砂主色。
  static ThemeData get light {
    final scheme = ColorScheme.fromSeed(
      seedColor: AppColors.cinnabar,
      brightness: Brightness.light,
    ).copyWith(
      primary: AppColors.cinnabar,
      secondary: AppColors.inkBlue,
      surface: AppColors.ricePaper,
    );
    return _buildTheme(scheme);
  }

  /// 深色主题：墨色背景 + 降饱和主色。
  static ThemeData get dark {
    final scheme = ColorScheme.fromSeed(
      seedColor: AppColors.cinnabar,
      brightness: Brightness.dark,
    ).copyWith(
      primary: AppColors.cinnabar,
      secondary: AppColors.inkBlue,
      surface: AppColors.inkBlack,
    );
    return _buildTheme(scheme);
  }

  static ThemeData _buildTheme(ColorScheme scheme) {
    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: scheme.surface,
      appBarTheme: AppBarTheme(
        centerTitle: true,
        backgroundColor: scheme.surface,
        foregroundColor: scheme.onSurface,
        elevation: 0,
        scrolledUnderElevation: 0.5,
      ),
      cardTheme: CardThemeData(
        elevation: 1,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
        color: scheme.surfaceContainerLow,
      ),
      bottomNavigationBarTheme: BottomNavigationBarThemeData(
        type: BottomNavigationBarType.fixed,
        selectedItemColor: scheme.primary,
        unselectedItemColor: scheme.onSurfaceVariant,
        backgroundColor: scheme.surface,
        elevation: 8,
      ),
    );
  }
}
