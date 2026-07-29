// 时间线 Repository：拉取朝代列表与朝代下事件。
//
// 对齐 doc/13-移动端设计.md §四 时间线页：
//   GET /api/v1/history/dynasties            -> 朝代列表（按 sort_order 排序）
//   GET /api/v1/history/events?dynasty_id=N  -> 朝代下事件
//
// 当前 MVP 仅拉取朝代列表，事件按需懒加载（点击朝代时触发）。

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';
import '../../../shared/models/history_event.dart';
import '../domain/timeline_state.dart';

final timelineRepositoryProvider = Provider<TimelineRepository>((ref) {
  return TimelineRepository(ref.read(historyApiClientProvider));
});

class TimelineRepository {
  TimelineRepository(this._api);
  final HistoryApiClient _api;

  /// 拉取时间线骨架（朝代列表）。
  Future<TimelineState> fetchTimeline() async {
    final dynasties = await _api.listDynasties();
    return TimelineState(dynasties: dynasties);
  }

  /// 拉取某朝代下的事件列表（懒加载）。
  Future<List<HistoryEvent>> fetchEventsForDynasty(int dynastyId) {
    return _api.listEvents(dynastyId: dynastyId);
  }
}
