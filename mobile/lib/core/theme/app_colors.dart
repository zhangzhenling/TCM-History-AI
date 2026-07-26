// 中医风格配色，取自传统色谱。对齐 doc/13-移动端设计.md §十一。
//
// 朱砂红用于主操作与人物标识，墨青用于经典与链接，赭石用于方剂，
// 黛蓝用于概念，宣纸色模拟古籍纸感背景，墨色用于深色模式背景。

import 'package:flutter/material.dart';

class AppColors {
  const AppColors._();

  /// 朱砂红 —— 主操作、人物标识。
  static const Color cinnabar = Color(0xFFB22222);

  /// 墨青 —— 经典、链接。
  static const Color inkBlue = Color(0xFF2F5C6E);

  /// 赭石 —— 方剂。
  static const Color ochre = Color(0xFF8B5A2B);

  /// 黛蓝 —— 概念。
  static const Color daiBlue = Color(0xFF3A506B);

  /// 宣纸色 —— 浅色模式背景。
  static const Color ricePaper = Color(0xFFF5F0E6);

  /// 墨色 —— 深色模式背景。
  static const Color inkBlack = Color(0xFF1C1B19);
}
