// 基于 SharedPreferences 的 JWT 持久化。
//
// sharedPreferencesProvider 必须在 main() 中通过 override 注入实例（因为
// SharedPreferences.getInstance() 是异步的，而 Dio 拦截器与路由守卫需要同步读取 token）。
// TokenStorage 封装 access_token / refresh_token 的读写与清除。

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _kAccessToken = 'access_token';
const _kRefreshToken = 'refresh_token';

/// SharedPreferences Provider，必须在 main() 中 override 注入。
final sharedPreferencesProvider = Provider<SharedPreferences>((ref) {
  throw UnimplementedError('sharedPreferencesProvider 必须在 main() 中 override 注入');
});

/// Token 读写工具。
class TokenStorage {
  final SharedPreferences prefs;
  TokenStorage(this.prefs);

  String? get accessToken => prefs.getString(_kAccessToken);
  String? get refreshToken => prefs.getString(_kRefreshToken);

  bool get isLoggedIn => accessToken != null && accessToken!.isNotEmpty;

  Future<void> saveTokens({
    required String accessToken,
    required String refreshToken,
  }) async {
    await prefs.setString(_kAccessToken, accessToken);
    await prefs.setString(_kRefreshToken, refreshToken);
  }

  Future<void> clear() async {
    await prefs.remove(_kAccessToken);
    await prefs.remove(_kRefreshToken);
  }
}

final tokenStorageProvider = Provider<TokenStorage>(
  (ref) => TokenStorage(ref.read(sharedPreferencesProvider)),
);
