// 跨实体检索页。
//
// 本页应承载（对齐 PRD §4.3 与 PC 端 learner search 页）：
// 1. 顶部搜索框 —— 输入关键词，提交后触发检索
// 2. 实体类型筛选 —— 可选人物/著作/学派/事件（后续接入 types 过滤）
// 3. 结果列表 —— 按 score 排序，每条展示 type 标签 + 名称 + 摘要，点击跳转
//    /detail/{type}/{id}（事件暂无独立详情页，后续扩展）
// 4. 空态 —— 无结果时提示
// 5. 历史搜索 / 热门推荐（后续接入）
//
// 数据来源：searchProvider -> SearchRepository -> HistoryApiClient.search。

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../shared/widgets/tcm_scaffold.dart';
import '../presentation/search_provider.dart';

class SearchPage extends ConsumerWidget {
  const SearchPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final query = ref.watch(searchQueryProvider);
    final asyncResults = ref.watch(searchProvider);
    final controller = TextEditingController(text: query);

    return Scaffold(
      appBar: AppBar(
        title: TextField(
          controller: controller,
          autofocus: true,
          textInputAction: TextInputAction.search,
          decoration: const InputDecoration(
            hintText: '搜索人物、著作、学派、事件…',
            border: InputBorder.none,
          ),
          onSubmitted: (value) => ref.read(searchQueryProvider.notifier).state = value,
        ),
        actions: [
          if (query.isNotEmpty)
            IconButton(
              icon: const Icon(Icons.close),
              onPressed: () {
                controller.clear();
                ref.read(searchQueryProvider.notifier).state = '';
              },
            ),
        ],
      ),
      body: SafeArea(
        child: asyncResults.when(
          data: (state) => state.isEmpty
              ? EmptyState(hint: query.isEmpty ? '输入关键词开始检索' : '未找到相关结果')
              : _SearchResults(state: state),
          loading: () => const LoadingIndicator(),
          error: (e, _) => ErrorView(
            message: '检索失败：$e',
            onRetry: () => ref.invalidate(searchProvider),
          ),
        ),
      ),
    );
  }
}

class _SearchResults extends StatelessWidget {
  final dynamic state; // SearchState
  const _SearchResults({required this.state});

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      padding: const EdgeInsets.symmetric(vertical: 8),
      itemCount: state.hits.length as int,
      separatorBuilder: (_, __) => const Divider(height: 1),
      itemBuilder: (context, i) {
        final hit = state.hits[i];
        return _SearchHitTile(hit: hit);
      },
    );
  }
}

class _SearchHitTile extends StatelessWidget {
  final dynamic hit; // SearchHit
  const _SearchHitTile({required this.hit});

  String _typeLabel(String type) {
    switch (type) {
      case 'person':
        return '人物';
      case 'book':
        return '著作';
      case 'school':
        return '学派';
      case 'event':
        return '事件';
      default:
        return type;
    }
  }

  @override
  Widget build(BuildContext context) {
    final type = hit.type as String;
    final id = hit.id as int;
    final source = hit.source as Map<String, dynamic>;
    final title = (source['name'] ?? source['title'] ?? '未命名') as String;

    return ListTile(
      leading: Chip(label: Text(_typeLabel(type))),
      title: Text(title),
      subtitle: Text(
        (source['summary'] ?? source['biography'] ?? source['description'] ?? '') as String,
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
      ),
      trailing: const Icon(Icons.chevron_right),
      onTap: () {
        // 事件暂无独立详情页，仅 person/book/school 可跳转
        if (type == 'person' || type == 'book' || type == 'school') {
          context.push('/detail/$type/$id');
        }
      },
    );
  }
}
