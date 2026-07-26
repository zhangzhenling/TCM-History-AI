// AuthController 单元测试：验证登录成功 / 失败状态切换与 token 落库。
//
// 通过 override authRepositoryProvider 注入伪 Repository，避免真实网络调用。

import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:tcm_history_ai/core/error/app_exception.dart';
import 'package:tcm_history_ai/core/network/token_storage.dart';
import 'package:tcm_history_ai/features/auth/data/auth_repository.dart';
import 'package:tcm_history_ai/features/auth/domain/auth_state.dart';
import 'package:tcm_history_ai/features/auth/presentation/auth_provider.dart';

/// 伪 AuthRepository：可控制 login/register 抛异常或成功。
class _FakeAuthRepository implements AuthRepository {
  _FakeAuthRepository(this.tokenStorage);
  final TokenStorage tokenStorage;

  bool shouldThrow = false;
  int loginCallCount = 0;
  int registerCallCount = 0;

  @override
  Future<void> login({
    required String username,
    required String password,
  }) async {
    loginCallCount++;
    if (shouldThrow) {
      throw const BusinessException(code: 401, message: '用户名或密码错误');
    }
    await tokenStorage.saveTokens(
      accessToken: 'access-$username',
      refreshToken: 'refresh-$username',
    );
  }

  @override
  Future<void> register({
    required String username,
    required String password,
    String? email,
    String? phone,
  }) async {
    registerCallCount++;
    if (shouldThrow) {
      throw const BusinessException(code: 409, message: '用户名已存在');
    }
    await tokenStorage.saveTokens(
      accessToken: 'access-$username',
      refreshToken: 'refresh-$username',
    );
  }

  @override
  Future<void> logout() async {
    await tokenStorage.clear();
  }
}

void main() {
  late SharedPreferences prefs;
  late TokenStorage tokenStorage;
  late _FakeAuthRepository fakeRepo;
  late ProviderContainer container;

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    prefs = await SharedPreferences.getInstance();
    tokenStorage = TokenStorage(prefs);
    fakeRepo = _FakeAuthRepository(tokenStorage);

    container = ProviderContainer(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(prefs),
        authRepositoryProvider.overrideWithValue(fakeRepo),
      ],
    );
  });

  tearDown(() => container.dispose());

  test('初始状态应为 idle', () {
    final state = container.read(authProvider);
    expect(state.status, AuthStatus.idle);
  });

  test('login 成功应切换到 success 并落库 token', () async {
    final ok = await container.read(authProvider.notifier).login(
          username: 'alice',
          password: 'secret',
        );
    expect(ok, isTrue);
    expect(fakeRepo.loginCallCount, 1);
    expect(tokenStorage.accessToken, 'access-alice');
    expect(tokenStorage.refreshToken, 'refresh-alice');
    expect(container.read(authProvider).status, AuthStatus.success);
  });

  test('login 失败应切换到 error 并带上 errorMessage', () async {
    fakeRepo.shouldThrow = true;
    final ok = await container.read(authProvider.notifier).login(
          username: 'alice',
          password: 'wrong',
        );
    expect(ok, isFalse);
    final state = container.read(authProvider);
    expect(state.status, AuthStatus.error);
    expect(state.errorMessage, '用户名或密码错误');
    expect(tokenStorage.isLoggedIn, isFalse);
  });

  test('register 成功后应自动落库 token（自动登录）', () async {
    final ok = await container.read(authProvider.notifier).register(
          username: 'bob',
          password: 'password123',
          email: 'bob@example.com',
        );
    expect(ok, isTrue);
    expect(fakeRepo.registerCallCount, 1);
    expect(tokenStorage.accessToken, 'access-bob');
  });

  test('reset 应将状态恢复为 idle', () async {
    await container.read(authProvider.notifier).login(
          username: 'alice',
          password: 'secret',
        );
    container.read(authProvider.notifier).reset();
    expect(container.read(authProvider).status, AuthStatus.idle);
  });
}
