// 时间线 Provider：暴露异步 TimelineState。
//
// 当前骨架返回 TimelineState.placeholder()，接入真实后端时改为：
//   final repo = ref.read(timelineRepositoryProvider);
//   return repo.fetchTimeline();

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../domain/timeline_state.dart';

final timelineProvider = FutureProvider<TimelineState>((ref) async {
  // TODO: 接入 timelineRepository 拉取真实数据
  // final repo = ref.read(timelineRepositoryProvider);
  // return repo.fetchTimeline();
  await Future<void>.delayed(const Duration(milliseconds: 200));
  return TimelineState.placeholder();
});
