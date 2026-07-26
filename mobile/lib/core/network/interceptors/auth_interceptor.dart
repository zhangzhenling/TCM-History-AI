// JWT 鉴权拦截器：
// 1. 请求方向：从 SharedPreferences（经 TokenStorage）读取 access_token，注入
//    `Authorization: Bearer {token}` 头。
// 2. 错误方向：收到 HTTP 401 时清空本地 token，并通过 onUnauthorized 回调
//    触发路由跳转到 /login。
//
// 对齐 PC 端 frontend/packages/api/src/http.ts 的请求拦截器与 401 处理。
// 当前骨架未实现 refresh token 自动刷新（PC 端通过 onAccessTokenExpired 回调
// 串行化刷新），后续可在此扩展。

import 'package:dio/dio.dart';

import '../token_storage.dart';

class AuthInterceptor extends Interceptor {
  final TokenStorage tokenStorage;
  final void Function()? onUnauthorized;

  AuthInterceptor({required this.tokenStorage, this.onUnauthorized});

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    final token = tokenStorage.accessToken;
    if (token != null && token.isNotEmpty) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
    final status = err.response?.statusCode;
    // 401 表示 access token 失效或未登录：清空 token 并跳转登录页。
    if (status == 401) {
      await tokenStorage.clear();
      onUnauthorized?.call();
    }
    handler.next(err);
  }
}
