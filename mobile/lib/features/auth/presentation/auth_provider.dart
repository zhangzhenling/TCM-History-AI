// 认证 Provider：StateNotifier 管理登录/注册表单状态。
//
// UI 层通过 ref.watch(authProvider) 观察状态切换 loading / error / success，
// 提交时调用 ref.read(authProvider.notifier).login(...) / register(...)。
// 成功后由调用方执行路由跳转（通常回到 returnUrl 或 /home）。

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/error/app_exception.dart';
import '../data/auth_repository.dart';
import '../domain/auth_state.dart';

final authProvider = StateNotifierProvider<AuthController, AuthState>((ref) {
  return AuthController(ref.read(authRepositoryProvider));
});

class AuthController extends StateNotifier<AuthState> {
  AuthController(this._repo) : super(const AuthState());
  final AuthRepository _repo;

  Future<bool> login({
    required String username,
    required String password,
  }) async {
    state = const AuthState(status: AuthStatus.loading);
    try {
      await _repo.login(username: username, password: password);
      state = const AuthState(status: AuthStatus.success);
      return true;
    } on AppException catch (e) {
      state = AuthState(status: AuthStatus.error, errorMessage: e.message);
      return false;
    } catch (e) {
      state = AuthState(status: AuthStatus.error, errorMessage: '$e');
      return false;
    }
  }

  Future<bool> register({
    required String username,
    required String password,
    String? email,
    String? phone,
  }) async {
    state = const AuthState(status: AuthStatus.loading);
    try {
      await _repo.register(
        username: username,
        password: password,
        email: email,
        phone: phone,
      );
      state = const AuthState(status: AuthStatus.success);
      return true;
    } on AppException catch (e) {
      state = AuthState(status: AuthStatus.error, errorMessage: e.message);
      return false;
    } catch (e) {
      state = AuthState(status: AuthStatus.error, errorMessage: '$e');
      return false;
    }
  }

  /// 重置为初始态（表单重新进入时调用）。
  void reset() {
    state = const AuthState();
  }
}
