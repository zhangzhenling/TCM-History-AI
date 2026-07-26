// Dio 单例装配：注册拦截器链（ErrorInterceptor -> AuthInterceptor）。
//
// 拦截器顺序说明（请求方向自上而下）：
//   ErrorInterceptor.onRequest (空) -> AuthInterceptor.onRequest (注入 JWT)
// 响应/错误方向：
//   ErrorInterceptor 拆 envelope / 映射异常 -> AuthInterceptor 处理 401 跳转
//
// AuthInterceptor 的 onUnauthorized 回调读取 appRouterProvider 跳转 /login，
// 实现 401 自动跳转登录。对齐 doc/13-移动端设计.md §六。

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../config/env_config.dart';
import '../router/app_router.dart';
import 'api_client.dart';
import 'interceptors/auth_interceptor.dart';
import 'interceptors/error_interceptor.dart';
import 'token_storage.dart';

/// Dio 单例 Provider。
final dioProvider = Provider<Dio>((ref) {
  final tokenStorage = ref.read(tokenStorageProvider);
  final router = ref.read(appRouterProvider);

  final dio = Dio(
    BaseOptions(
      baseUrl: EnvConfig.apiBaseUrl,
      connectTimeout: const Duration(milliseconds: EnvConfig.connectTimeoutMs),
      receiveTimeout: const Duration(milliseconds: EnvConfig.receiveTimeoutMs),
      headers: const {'Content-Type': 'application/json'},
    ),
  );

  dio.interceptors.add(ErrorInterceptor());
  dio.interceptors.add(
    AuthInterceptor(
      tokenStorage: tokenStorage,
      onUnauthorized: () => router.go('/login'),
    ),
  );

  return dio;
});

/// 类型化 API 客户端 Provider。
final historyApiClientProvider = Provider<HistoryApiClient>(
  (ref) => HistoryApiClient(ref.read(dioProvider)),
);

/// 鉴权 API 客户端 Provider。
final authApiClientProvider = Provider<AuthApiClient>(
  (ref) => AuthApiClient(ref.read(dioProvider)),
);
