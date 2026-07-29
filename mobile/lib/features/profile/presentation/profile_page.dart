// 「我的」页面：登录态切换 + 用户信息 + 学习记录入口 + 设置入口。
//
// 本页应承载（对齐 doc/13-移动端设计.md §四 我的页 与 PRD §4）：
// 1. 未登录态：登录 / 注册按钮 → 跳转 /login / /register
// 2. 已登录态：头像 + 昵称 + 用户名 + 简介
// 3. 学习记录入口（后续 P4+ 接入 /api/v1/learning/progress）
// 4. 设置入口（语言 / 主题 / 通知偏好）
// 5. 退出登录按钮 —— 清空 token 并刷新
//
// 数据来源：profileProvider（同步读取 tokenStorage 登录态）。

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/network/token_storage.dart';
import '../../../shared/widgets/tcm_scaffold.dart';
import '../domain/profile_state.dart';
import 'profile_provider.dart';

class ProfilePage extends ConsumerWidget {
  const ProfilePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncProfile = ref.watch(profileProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('我的')),
      body: SafeArea(
        child: asyncProfile.when(
          data: (profile) => profile.isLoggedIn
              ? _LoggedInView(profile: profile)
              : const _GuestView(),
          loading: () => const LoadingIndicator(),
          error: (e, _) => ErrorView(
            message: '加载失败：$e',
            onRetry: () => ref.invalidate(profileProvider),
          ),
        ),
      ),
    );
  }
}

class _GuestView extends StatelessWidget {
  const _GuestView();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.lock_outline,
                size: 48, color: Theme.of(context).colorScheme.outline),
            const SizedBox(height: 16),
            Text('登录后查看学习记录',
                style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Text(
              '登录账号后可同步学习进度、收藏与笔记',
              style: Theme.of(context).textTheme.bodySmall,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: () => context.go('/login?from=/profile'),
              child: const Text('登录'),
            ),
            const SizedBox(height: 12),
            TextButton(
              onPressed: () => context.go('/register?from=/profile'),
              child: const Text('注册新账号'),
            ),
          ],
        ),
      ),
    );
  }
}

class _LoggedInView extends ConsumerWidget {
  final ProfileState profile;
  const _LoggedInView({required this.profile});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ListView(
      padding: const EdgeInsets.symmetric(vertical: 16),
      children: [
        // 用户信息卡
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Row(
            children: [
              CircleAvatar(
                radius: 32,
                backgroundColor: Theme.of(context).colorScheme.primaryContainer,
                backgroundImage: profile.avatarUrl.isNotEmpty
                    ? NetworkImage(profile.avatarUrl)
                    : null,
                child: profile.avatarUrl.isEmpty
                    ? Text(
                        profile.nickname.isNotEmpty
                            ? profile.nickname.characters.first
                            : '?',
                        style: Theme.of(context).textTheme.headlineSmall,
                      )
                    : null,
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(profile.nickname,
                        style: Theme.of(context).textTheme.titleLarge),
                    const SizedBox(height: 4),
                    Text('@${profile.username}',
                        style: Theme.of(context).textTheme.bodySmall),
                    if (profile.bio.isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Text(profile.bio,
                          style: Theme.of(context).textTheme.bodyMedium),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 24),
        // 学习记录入口
        _MenuTile(
          icon: Icons.school_outlined,
          title: '学习记录',
          subtitle: '查看你的学习进度与历史',
          onTap: () {
            // TODO: 后续接入学习记录页
          },
        ),
        _MenuTile(
          icon: Icons.favorite_outline,
          title: '我的收藏',
          subtitle: '收藏的人物、著作与学派',
          onTap: () {},
        ),
        _MenuTile(
          icon: Icons.settings_outlined,
          title: '设置',
          subtitle: '语言、主题、通知偏好',
          onTap: () {},
        ),
        const SizedBox(height: 24),
        // 退出登录
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: OutlinedButton.icon(
            icon: const Icon(Icons.logout),
            label: const Text('退出登录'),
            onPressed: () async {
              await ref.read(tokenStorageProvider).clear();
              ref.invalidate(profileProvider);
            },
          ),
        ),
      ],
    );
  }
}

class _MenuTile extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback? onTap;

  const _MenuTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Icon(icon),
      title: Text(title),
      subtitle: Text(subtitle),
      trailing: const Icon(Icons.chevron_right),
      onTap: onTap,
    );
  }
}
