// 朝代时间线页：纵向展示朝代演进脉络。
//
// 本页应承载（对齐 doc/13-移动端设计.md §四 时间线页 与 PRD §4）：
// 1. 朝代时间轴 —— 按 sort_order 纵向排列，左侧时间刻度，右侧朝代卡片
// 2. 朝代下事件 —— 点击朝代展开事件列表（后续 P7+ 接入）
// 3. 关键事件高亮 —— 标记医学史关键节点（成书、人物生卒等）
//
// 数据来源：timelineProvider -> TimelineRepository -> HistoryApiClient。

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../shared/models/dynasty.dart';
import '../../../shared/widgets/tcm_scaffold.dart';
import 'timeline_provider.dart';

class TimelinePage extends ConsumerWidget {
  const TimelinePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncTimeline = ref.watch(timelineProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('发展时间线')),
      body: SafeArea(
        child: asyncTimeline.when(
          data: (state) => state.isEmpty
              ? const EmptyState(hint: '暂无朝代数据')
              : _TimelineView(state: state),
          loading: () => const LoadingIndicator(),
          error: (e, _) => ErrorView(
            message: '加载失败：$e',
            onRetry: () => ref.invalidate(timelineProvider),
          ),
        ),
      ),
    );
  }
}

class _TimelineView extends StatelessWidget {
  final dynamic state; // TimelineState
  const _TimelineView({required this.state});

  @override
  Widget build(BuildContext context) {
    final dynasties = state.dynasties as List<Dynasty>;
    return ListView.builder(
      padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 12),
      itemCount: dynasties.length,
      itemBuilder: (context, i) {
        final dynasty = dynasties[i];
        final isLast = i == dynasties.length - 1;
        return _DynastyTimelineTile(dynasty: dynasty, isLast: isLast);
      },
    );
  }
}

class _DynastyTimelineTile extends StatelessWidget {
  final Dynasty dynasty;
  final bool isLast;
  const _DynastyTimelineTile({required this.dynasty, required this.isLast});

  String _formatYear(int year) {
    return year < 0 ? '公元前 ${-year}' : '公元 $year';
  }

  @override
  Widget build(BuildContext context) {
    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // 左侧时间刻度 + 轴线
          SizedBox(
            width: 72,
            child: Column(
              children: [
                Text(
                  dynasty.name,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        color: Theme.of(context).colorScheme.primary,
                      ),
                ),
                const SizedBox(height: 4),
                Text(
                  _formatYear(dynasty.startYear),
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                if (!isLast)
                  Expanded(
                    child: VerticalDivider(
                      width: 2,
                      color: Theme.of(context).colorScheme.outlineVariant,
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(width: 12),
          // 右侧朝代卡片
          Expanded(
            child: Container(
              margin: EdgeInsets.only(bottom: isLast ? 0 : 16),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.surfaceContainerHigh,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '${_formatYear(dynasty.startYear)} — ${_formatYear(dynasty.endYear)}',
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          color: Theme.of(context).colorScheme.onSurfaceVariant,
                        ),
                  ),
                  const SizedBox(height: 8),
                  Text(dynasty.description,
                      style: Theme.of(context).textTheme.bodyMedium),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
