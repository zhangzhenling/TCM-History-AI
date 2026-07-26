// TokenStorage 单元测试：验证 access_token / refresh_token 读写与清空。

import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:tcm_history_ai/core/network/token_storage.dart';

void main() {
  late SharedPreferences prefs;
  late TokenStorage storage;

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    prefs = await SharedPreferences.getInstance();
    storage = TokenStorage(prefs);
  });

  test('初始状态应为未登录', () {
    expect(storage.isLoggedIn, isFalse);
    expect(storage.accessToken, isNull);
    expect(storage.refreshToken, isNull);
  });

  test('saveTokens 后应能读取并标记为已登录', () async {
    await storage.saveTokens(
      accessToken: 'access-abc',
      refreshToken: 'refresh-xyz',
    );
    expect(storage.accessToken, 'access-abc');
    expect(storage.refreshToken, 'refresh-xyz');
    expect(storage.isLoggedIn, isTrue);
  });

  test('clear 后应清空 token 并标记为未登录', () async {
    await storage.saveTokens(
      accessToken: 'access-abc',
      refreshToken: 'refresh-xyz',
    );
    await storage.clear();
    expect(storage.accessToken, isNull);
    expect(storage.refreshToken, isNull);
    expect(storage.isLoggedIn, isFalse);
  });

  test('仅 access_token 为空字符串时应视为未登录', () async {
    await storage.saveTokens(accessToken: '', refreshToken: 'refresh');
    expect(storage.isLoggedIn, isFalse);
  });
}
