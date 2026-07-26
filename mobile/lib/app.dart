// App 根 Widget：MaterialApp.router 装配，使用 go_router + Material 3 主题，中文优先。
//
// 路由配置来自 appRouterProvider，主题来自 AppTheme；本地化代理接入
// flutter_localizations 以支持中文 Material 控件文案。

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/router/app_router.dart';
import 'core/theme/app_theme.dart';

class TcmHistoryApp extends ConsumerWidget {
  const TcmHistoryApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(appRouterProvider);

    return MaterialApp.router(
      title: '中医发展史 AI',
      debugShowCheckedModeBanner: false,
      // 主题
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      themeMode: ThemeMode.system,
      // 路由
      routerConfig: router,
      // 中文优先
      locale: const Locale('zh', 'CN'),
      supportedLocales: const [
        Locale('zh', 'CN'),
        Locale('en', 'US'),
      ],
      localizationsDelegates: const [
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
    );
  }
}
