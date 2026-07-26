// 移动端入口：初始化 SharedPreferences（用于 JWT 持久化），包裹 ProviderScope 启动 App。
//
// SharedPreferences 必须在 runApp 之前完成初始化，并通过 override 注入
// sharedPreferencesProvider，供 Dio 拦截器与路由守卫同步读取 token。

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app.dart';
import 'core/network/token_storage.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final prefs = await SharedPreferences.getInstance();

  runApp(
    ProviderScope(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(prefs),
      ],
      child: const TcmHistoryApp(),
    ),
  );
}
