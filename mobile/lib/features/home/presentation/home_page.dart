// 首页：用户每次启动的落地页。
//
// 本页应承载（对齐 doc/13-移动端设计.md §四 首页 与 PRD §4 首页模块）：
// 1. 顶部搜索入口 —— 点击跳转 /search 跨实体检索
// 2. 朝代时间线横滑条 —— 精选朝代缩略卡，点击跳转 /timeline
// 3. 推荐人物区 —— 横滑人物卡，点击跳转 /detail/person/:id
// 4. 推荐著作区 —— 横滑著作卡，点击跳转 /detail/book/:id
// 5. 继续学习卡（已登录）—— 最近未完成课时入口（后续 P4+ 接入）
//
// 数据来源：homeProvider -> HomeRepository -> HistoryApiClient。

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../shared/models/book.dart';
import '../../../shared/models/dynasty.dart';
import '../../../shared/models/person.dart';
import '../../../shared/widgets/tcm_scaffold.dart';
import '../domain/home_state.dart';
import 'home_provider.dart';

class HomePage extends ConsumerWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncHome = ref.watch(homeProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('首页')),
      body: SafeArea(
        child: asyncHome.when(
          data: (state) => _HomeContent(state: state),
          loading: () => const LoadingIndicator(),
          error: (e, _) => ErrorView(
            message: '加载失败：$e',
            onRetry: () => ref.invalidate(homeProvider),
          ),
        ),
      ),
    );
  }
}

class _HomeContent extends StatelessWidget {
  final HomeState state;
  const _HomeContent({required this.state});

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.symmetric(vertical: 12),
      children: [
        // 1. 搜索入口
        _SearchEntry(onTap: () => context.go('/search')),
        const SizedBox(height: 16),
        // 2. 朝代时间线入口
        _SectionTitle(title: '发展时间线', onMore: () => context.go('/timeline')),
        SizedBox(
          height: 92,
          child: ListView.separated(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 16),
            itemCount: state.dynasties.length,
            separatorBuilder: (_, __) => const SizedBox(width: 12),
            itemBuilder: (_, i) => _DynastyChip(dynasty: state.dynasties[i]),
          ),
        ),
        const SizedBox(height: 16),
        // 3. 推荐人物
        _SectionTitle(title: '推荐人物', onMore: () => context.go('/search')),
        SizedBox(
          height: 120,
          child: ListView.separated(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 16),
            itemCount: state.recommendedPersons.length,
            separatorBuilder: (_, __) => const SizedBox(width: 12),
            itemBuilder: (_, i) => _PersonCard(person: state.recommendedPersons[i]),
          ),
        ),
        const SizedBox(height: 16),
        // 4. 推荐著作
        _SectionTitle(title: '推荐著作', onMore: () => context.go('/search')),
        SizedBox(
          height: 120,
          child: ListView.separated(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 16),
            itemCount: state.recommendedBooks.length,
            separatorBuilder: (_, __) => const SizedBox(width: 12),
            itemBuilder: (_, i) => _BookCard(book: state.recommendedBooks[i]),
          ),
        ),
      ],
    );
  }
}

class _SearchEntry extends StatelessWidget {
  final VoidCallback onTap;
  const _SearchEntry({required this.onTap});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(24),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surfaceContainerHigh,
            borderRadius: BorderRadius.circular(24),
          ),
          child: Row(
            children: [
              Icon(Icons.search, color: Theme.of(context).colorScheme.onSurfaceVariant),
              const SizedBox(width: 8),
              Text('搜索人物、著作、学派、事件…',
                  style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        color: Theme.of(context).colorScheme.onSurfaceVariant,
                      )),
            ],
          ),
        ),
      ),
    );
  }
}

class _SectionTitle extends StatelessWidget {
  final String title;
  final VoidCallback? onMore;
  const _SectionTitle({required this.title, this.onMore});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          if (onMore != null)
            TextButton(onPressed: onMore, child: const Text('更多')),
        ],
      ),
    );
  }
}

class _DynastyChip extends StatelessWidget {
  final Dynasty dynasty;
  const _DynastyChip({required this.dynasty});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 120,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHigh,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(dynasty.name, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 4),
          Expanded(
            child: Text(dynasty.description,
                maxLines: 2, overflow: TextOverflow.ellipsis),
          ),
        ],
      ),
    );
  }
}

class _PersonCard extends StatelessWidget {
  final Person person;
  const _PersonCard({required this.person});

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 160,
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(person.name, style: Theme.of(context).textTheme.titleMedium),
              if (person.title.isNotEmpty) ...[
                const SizedBox(height: 4),
                Text(person.title, style: Theme.of(context).textTheme.bodySmall),
              ],
              const SizedBox(height: 4),
              Expanded(
                child: Text(person.biography,
                    maxLines: 2, overflow: TextOverflow.ellipsis),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _BookCard extends StatelessWidget {
  final Book book;
  const _BookCard({required this.book});

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 160,
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(book.title, style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 4),
              Text(book.category, style: Theme.of(context).textTheme.bodySmall),
              const SizedBox(height: 4),
              Expanded(
                child: Text(book.summary,
                    maxLines: 2, overflow: TextOverflow.ellipsis),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
