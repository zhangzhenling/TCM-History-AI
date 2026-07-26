// 时间线状态：朝代列表 + 每个朝代下的事件分组。
//
// 对齐 doc/13-移动端设计.md §四 时间线页：
//   GET /api/v1/history/dynasties        -> 朝代顺序骨架
//   GET /api/v1/history/events?dynasty_id -> 朝代下事件
//
// 当前 MVP 仅渲染朝代列表，事件分组为后续 P7+ 扩展点。

import 'package:flutter/foundation.dart';

import '../../../shared/models/dynasty.dart';
import '../../../shared/models/history_event.dart';

@immutable
class TimelineState {
  final List<Dynasty> dynasties;
  final Map<int, List<HistoryEvent>> eventsByDynasty;

  const TimelineState({
    required this.dynasties,
    this.eventsByDynasty = const {},
  });

  bool get isEmpty => dynasties.isEmpty;

  /// 占位数据，用于骨架阶段展示。
  factory TimelineState.placeholder() => const TimelineState(
        dynasties: [
          Dynasty(
              id: 1,
              name: '先秦',
              startYear: -2070,
              endYear: -221,
              sortOrder: 1,
              description: '中医起源与原始积累'),
          Dynasty(
              id: 2,
              name: '汉',
              startYear: -206,
              endYear: 220,
              sortOrder: 2,
              description: '《黄帝内经》《伤寒论》成书'),
          Dynasty(
              id: 3,
              name: '魏晋',
              startYear: 220,
              endYear: 589,
              sortOrder: 3,
              description: '王叔和《脉经》、皇甫谧《针灸甲乙经》'),
          Dynasty(
              id: 4,
              name: '隋唐',
              startYear: 581,
              endYear: 907,
              sortOrder: 4,
              description: '《新修本草》、孙思邈《千金方》'),
          Dynasty(
              id: 5,
              name: '宋金元',
              startYear: 960,
              endYear: 1368,
              sortOrder: 5,
              description: '金元四大家争鸣'),
          Dynasty(
              id: 6,
              name: '明清',
              startYear: 1368,
              endYear: 1911,
              sortOrder: 6,
              description: '李时珍《本草纲目》、温病学派兴起'),
        ],
      );
}
