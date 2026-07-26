// 路由守卫：认证拦截。
//
// MVP 策略对齐 PRD：游客可浏览公开内容（首页 / 检索 / 详情 / 时间线），
// 仅「我的」与登录后可见的功能需登录。未登录访问受保护路径时重定向到 /login
// 并携带 returnUrl；已登录用户访问 /login / /register 时重定向回 /home。

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../network/token_storage.dart';

class RouteGuards {
  /// 需要登录才能访问的路径。
  static const _protectedRoutes = <String>['/profile'];

  /// 认证守卫。
  static String? authGuard(Ref ref, BuildContext context, GoRouterState state) {
    final isLoggedIn = ref.read(tokenStorageProvider).isLoggedIn;
    final location = state.matchedLocation;
    final isAuthRoute = location == '/login' || location == '/register';

    // 已登录用户访问登录/注册页 -> 回首页
    if (isLoggedIn && isAuthRoute) {
      final from = state.uri.queryParameters['from'];
      return from ?? '/home';
    }

    // 未登录用户访问受保护路径 -> 跳登录页并携带 returnUrl
    if (!isLoggedIn && _protectedRoutes.contains(location)) {
      return '/login?from=$location';
    }

    return null;
  }
}
