// go_router 路由表：覆盖 5 个核心页面 + login + register，共 7 条路由。
//
// 使用 ShellRoute 持久化底部导航栏（首页/检索/时间线/我的），切换 Tab 时导航栏不重建。
// 详情页 /detail/:type/:id 通过 type 路由参数区分人物/著作/学派，对齐 PC 端
// learner 路由的 persons/:id / books/:id / schools/:id。
// 认证守卫通过根路由 redirect 钩子统一拦截。

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/presentation/login_page.dart';
import '../../features/auth/presentation/register_page.dart';
import '../../features/detail/presentation/detail_page.dart';
import '../../features/home/presentation/home_page.dart';
import '../../features/profile/presentation/profile_page.dart';
import '../../features/search/presentation/search_page.dart';
import '../../features/timeline/presentation/timeline_page.dart';
import 'route_guards.dart';

final appRouterProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    initialLocation: '/home',
    debugLogDiagnostics: true,
    redirect: (context, state) => RouteGuards.authGuard(ref, context, state),
    routes: [
      // 主导航：ShellRoute 持久化底部导航栏
      ShellRoute(
        builder: (context, state, child) => _MainShell(
          location: state.matchedLocation,
          child: child,
        ),
        routes: [
          GoRoute(
            path: '/home',
            name: 'home',
            builder: (_, __) => const HomePage(),
          ),
          GoRoute(
            path: '/search',
            name: 'search',
            builder: (_, __) => const SearchPage(),
          ),
          GoRoute(
            path: '/timeline',
            name: 'timeline',
            builder: (_, __) => const TimelinePage(),
          ),
          GoRoute(
            path: '/profile',
            name: 'profile',
            builder: (_, __) => const ProfilePage(),
          ),
        ],
      ),
      // 认证页
      GoRoute(
        path: '/login',
        name: 'login',
        builder: (_, __) => const LoginPage(),
      ),
      GoRoute(
        path: '/register',
        name: 'register',
        builder: (_, __) => const RegisterPage(),
      ),
      // 实体详情页：type=person/book/school
      GoRoute(
        path: '/detail/:type/:id',
        name: 'detail',
        builder: (_, state) => DetailPage(
          entityType: state.pathParameters['type']!,
          entityId: state.pathParameters['id']!,
        ),
      ),
    ],
  );
});

/// 主导航外壳：底部 4 Tab + 持久化 body。
class _MainShell extends StatelessWidget {
  final String location;
  final Widget child;

  const _MainShell({required this.location, required this.child});

  int get _currentIndex {
    if (location.startsWith('/search')) return 1;
    if (location.startsWith('/timeline')) return 2;
    if (location.startsWith('/profile')) return 3;
    return 0; // 默认首页
  }

  static const _paths = ['/home', '/search', '/timeline', '/profile'];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: _currentIndex,
        type: BottomNavigationBarType.fixed,
        items: const [
          BottomNavigationBarItem(
              icon: Icon(Icons.home_outlined), activeIcon: Icon(Icons.home), label: '首页'),
          BottomNavigationBarItem(icon: Icon(Icons.search), label: '检索'),
          BottomNavigationBarItem(
              icon: Icon(Icons.timeline_outlined),
              activeIcon: Icon(Icons.timeline),
              label: '时间线'),
          BottomNavigationBarItem(
              icon: Icon(Icons.person_outline),
              activeIcon: Icon(Icons.person),
              label: '我的'),
        ],
        onTap: (i) => context.go(_paths[i]),
      ),
    );
  }
}
