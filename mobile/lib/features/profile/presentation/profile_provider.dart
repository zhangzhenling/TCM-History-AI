// 「我的」Provider：根据 tokenStorage 登录态切换 ProfileState。
//
// 登录态由 tokenStorageProvider.isLoggedIn 同步读取（基于 SharedPreferences），
// 后续接入用户信息接口（GET /api/v1/user/profile）后填充 nickname/avatarUrl 等字段。

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/token_storage.dart';
import '../domain/profile_state.dart';

final profileProvider = Provider<ProfileState>((ref) {
  final tokenStorage = ref.watch(tokenStorageProvider);
  if (!tokenStorage.isLoggedIn) {
    return const ProfileState.guest();
  }
  // TODO: 接入用户信息接口拉取 nickname / avatarUrl / bio
  return const ProfileState.placeholder();
});
