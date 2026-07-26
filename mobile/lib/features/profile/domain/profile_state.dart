// 「我的」页面状态：当前用户信息 + 学习记录摘要。
//
// 对齐 doc/13-移动端设计.md §四 我的页 与 PRD §4：
//   - 未登录态：展示登录/注册入口
//   - 已登录态：用户头像/昵称/账号信息 + 学习进度 + 设置入口
//
// 当前 MVP 仅区分登录态与未登录态，学习记录摘要为后续 P4+ 接入。

import 'package:flutter/foundation.dart';

@immutable
class ProfileState {
  final bool isLoggedIn;
  final int? userId;
  final String username;
  final String nickname;
  final String avatarUrl;
  final String bio;

  const ProfileState({
    required this.isLoggedIn,
    this.userId,
    this.username = '',
    this.nickname = '',
    this.avatarUrl = '',
    this.bio = '',
  });

  /// 未登录态占位。
  const ProfileState.guest() : this(isLoggedIn: false);

  /// 已登录态占位（后续接入真实用户信息接口替换）。
  const ProfileState.placeholder()
      : this(
          isLoggedIn: true,
          userId: 1,
          username: 'learner',
          nickname: '学习者',
          avatarUrl: '',
          bio: '中医发展史学习中',
        );
}
