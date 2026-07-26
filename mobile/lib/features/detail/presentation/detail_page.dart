// 实体详情页：根据 entityType 渲染人物 / 著作 / 学派不同布局。
//
// 路由 /detail/:type/:id 注入 entityType 与 entityId 两个 path 参数。
// 对齐 doc/13-移动端设计.md §四 人物/经典/学派详情页设计：
//   - 人物：肖像 + 生平 + 主要成就 + 朝代归属
//   - 著作：书名 + 成书年代 + 类别 + 卷数 + 内容摘要
//   - 学派：学派名 + 创始人 + 建立年代 + 学派特点
//
// 数据来源：detailProvider((type, id)) -> DetailRepository -> HistoryApiClient。

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../shared/models/book.dart';
import '../../../shared/models/person.dart';
import '../../../shared/models/school.dart';
import '../../../shared/widgets/tcm_scaffold.dart';
import 'detail_provider.dart';

class DetailPage extends ConsumerWidget {
  final String entityType;
  final String entityId;

  const DetailPage({
    super.key,
    required this.entityType,
    required this.entityId,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncDetail = ref.watch(
      detailProvider(DetailParams(entityType: entityType, entityId: entityId)),
    );

    return Scaffold(
      appBar: AppBar(title: Text(_titleFor(entityType))),
      body: SafeArea(
        child: asyncDetail.when(
          data: (state) => _DetailContent(state: state),
          loading: () => const LoadingIndicator(),
          error: (e, _) => ErrorView(
            message: '加载失败：$e',
            onRetry: () => ref.invalidate(
              detailProvider(
                DetailParams(entityType: entityType, entityId: entityId),
              ),
            ),
          ),
        ),
      ),
    );
  }

  String _titleFor(String type) {
    switch (type) {
      case 'person':
        return '人物详情';
      case 'book':
        return '著作详情';
      case 'school':
        return '学派详情';
      default:
        return '详情';
    }
  }
}

class _DetailContent extends StatelessWidget {
  final dynamic state; // DetailState
  const _DetailContent({required this.state});

  @override
  Widget build(BuildContext context) {
    final type = state.entityType as String;
    switch (type) {
      case 'person':
        return _PersonDetail(person: state.person as Person);
      case 'book':
        return _BookDetail(book: state.book as Book);
      case 'school':
        return _SchoolDetail(school: state.school as School);
      default:
        return const EmptyState(hint: '暂未支持的实体类型');
    }
  }
}

class _PersonDetail extends StatelessWidget {
  final Person person;
  const _PersonDetail({required this.person});

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        // 头部：姓名 + 称号
        Row(
          children: [
            CircleAvatar(
              radius: 32,
              backgroundColor: Theme.of(context).colorScheme.primaryContainer,
              child: Text(
                person.name.isNotEmpty ? person.name.characters.first : '?',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(person.name, style: Theme.of(context).textTheme.headlineSmall),
                  if (person.title.isNotEmpty)
                    Text(person.title, style: Theme.of(context).textTheme.bodyMedium),
                  const SizedBox(height: 4),
                  Text(
                    '${person.birthYear} — ${person.deathYear}',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                ],
              ),
            ),
          ],
        ),
        const SizedBox(height: 16),
        if (person.courtesyName.isNotEmpty || person.aliasName.isNotEmpty)
          _InfoRow(label: '字号', value: [person.courtesyName, person.aliasName].where((s) => s.isNotEmpty).join(' / ')),
        _Section(title: '生平', body: person.biography),
        if (person.achievements.isNotEmpty)
          _Section(title: '主要成就', body: person.achievements),
      ],
    );
  }
}

class _BookDetail extends StatelessWidget {
  final Book book;
  const _BookDetail({required this.book});

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text(book.title, style: Theme.of(context).textTheme.headlineSmall),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          children: [
            if (book.category.isNotEmpty) Chip(label: Text(book.category)),
            if (book.volumeCount > 0) Chip(label: Text('${book.volumeCount} 卷')),
            Chip(label: Text(book.isExtant ? '存世' : '已佚')),
          ],
        ),
        const SizedBox(height: 16),
        if (book.publishedYear != 0)
          _InfoRow(label: '成书年代', value: '${book.publishedYear} 年'),
        _Section(title: '内容摘要', body: book.summary),
      ],
    );
  }
}

class _SchoolDetail extends StatelessWidget {
  final School school;
  const _SchoolDetail({required this.school});

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text(school.name, style: Theme.of(context).textTheme.headlineSmall),
        const SizedBox(height: 8),
        if (school.establishedYear != 0)
          _InfoRow(label: '建立年代', value: '${school.establishedYear} 年'),
        _Section(title: '学派特点', body: school.summary),
      ],
    );
  }
}

class _InfoRow extends StatelessWidget {
  final String label;
  final String value;
  const _InfoRow({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 80,
            child: Text(label,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    )),
          ),
          Expanded(child: Text(value)),
        ],
      ),
    );
  }
}

class _Section extends StatelessWidget {
  final String title;
  final String body;
  const _Section({required this.title, required this.body});

  @override
  Widget build(BuildContext context) {
    if (body.isEmpty) return const SizedBox.shrink();
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
          Text(body, style: Theme.of(context).textTheme.bodyMedium),
        ],
      ),
    );
  }
}
