// 认证 Repository：登录 / 注册调用，成功后落库 token。
//
// 对齐 frontend/packages/api/src/modules/auth.ts：
//   POST /api/v1/auth/login
//   POST /api/v1/auth/register
// 返回 TokenPair 后通过 TokenStorage 持久化 access_token / refresh_token。

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';
import '../../../core/network/token_storage.dart';

final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepository(
    api: ref.read(authApiClientProvider),
    tokenStorage: ref.read(tokenStorageProvider),
  );
});

class AuthRepository {
  AuthRepository({required this.api, required this.tokenStorage});
  final AuthApiClient api;
  final TokenStorage tokenStorage;

  /// 登录：调 /auth/login，成功后落库 token。
  Future<void> login({
    required String username,
    required String password,
  }) async {
    final tokens = await api.login(username: username, password: password);
    await tokenStorage.saveTokens(
      accessToken: tokens.accessToken,
      refreshToken: tokens.refreshToken,
    );
  }

  /// 注册：调 /auth/register，成功后直接落库 token（自动登录）。
  Future<void> register({
    required String username,
    required String password,
    String? email,
    String? phone,
  }) async {
    final tokens = await api.register(
      username: username,
      password: password,
      email: email,
      phone: phone,
    );
    await tokenStorage.saveTokens(
      accessToken: tokens.accessToken,
      refreshToken: tokens.refreshToken,
    );
  }

  /// 退出登录：清空本地 token。
  Future<void> logout() async {
    await tokenStorage.clear();
  }
}
