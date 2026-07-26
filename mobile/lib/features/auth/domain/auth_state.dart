// 认证状态：登录/注册表单状态与异步提交结果。
//
// 对齐 doc/13-移动端设计.md §四 登录/注册页 与 PRD：
//   POST /api/v1/auth/login    { username, password }
//   POST /api/v1/auth/register { username, password, email?, phone? }
// 成功后返回 TokenPair（access_token + refresh_token），存入 TokenStorage。

import 'package:flutter/foundation.dart';

import '../../../core/network/api_client.dart';

@immutable
class AuthState {
  final AuthStatus status;
  final String? errorMessage;
  final TokenPair? tokens;

  const AuthState({
    this.status = AuthStatus.idle,
    this.errorMessage,
    this.tokens,
  });

  AuthState copyWith({
    AuthStatus? status,
    String? errorMessage,
    TokenPair? tokens,
  }) {
    return AuthState(
      status: status ?? this.status,
      errorMessage: errorMessage ?? this.errorMessage,
      tokens: tokens ?? this.tokens,
    );
  }
}

enum AuthStatus { idle, loading, success, error }
