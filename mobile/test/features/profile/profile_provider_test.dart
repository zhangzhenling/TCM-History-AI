// ProfileProvider 单元测试：验证登录态切换 ProfileState。

import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:tcm_history_ai/core/network/token_storage.dart';
import 'package:tcm_history_ai/features/profile/domain/profile_state.dart';
import 'package:tcm_history_ai/features/profile/presentation/profile_provider.dart';

void main() {
  late ProviderContainer container;
  late TokenStorage tokenStorage;

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    final prefs = await SharedPreferences.getInstance();
    tokenStorage = TokenStorage(prefs);
    container = ProviderContainer(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(prefs),
      ],
    );
  });

  tearDown(() => container.dispose());

  test('未登录时应返回 guest 态', () {
    final profile = container.read(profileProvider);
    expect(profile.isLoggedIn, isFalse);
    expect(profile, isA<ProfileState>());
  });

  test('登录后应切换为已登录态', () async {
    await tokenStorage.saveTokens(
      accessToken: 'access',
      refreshToken: 'refresh',
    );
    container.invalidate(profileProvider);
    final profile = container.read(profileProvider);
    expect(profile.isLoggedIn, isTrue);
    expect(profile.username, 'learner');
  });

  test('清空 token 后应回到 guest 态', () async {
    await tokenStorage.saveTokens(
      accessToken: 'access',
      refreshToken: 'refresh',
    );
    container.invalidate(profileProvider);
    expect(container.read(profileProvider).isLoggedIn, isTrue);

    await tokenStorage.clear();
    container.invalidate(profileProvider);
    expect(container.read(profileProvider).isLoggedIn, isFalse);
  });
}
