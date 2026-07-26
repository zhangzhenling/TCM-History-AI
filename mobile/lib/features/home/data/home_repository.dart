// 首页 Repository：聚合首页所需三路数据。
//
// 对齐 doc/13-移动端设计.md §四：homeProvider 并行调用
//   GET /history/timeline/highlights（朝代时间线）
//   GET /history/persons（推荐人物）
//   GET /history/books（推荐著作）
// 三路结果合并为 HomeState。
//
// Repository 是业务逻辑聚合点：Provider 不直接调用 Dio，统一经此层。

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';
import '../../../shared/models/book.dart';
import '../../../shared/models/dynasty.dart';
import '../../../shared/models/person.dart';
import '../domain/home_state.dart';

final homeRepositoryProvider = Provider<HomeRepository>((ref) {
  return HomeRepository(ref.read(historyApiClientProvider));
});

class HomeRepository {
  HomeRepository(this._api);
  final HistoryApiClient _api;

  Future<List<Dynasty>> fetchTimelineHighlights() {
    // TODO: 后端提供 /history/timeline/highlights 精选接口；当前复用 listDynasties
    return _api.listDynasties();
  }

  Future<List<Person>> fetchRecommendedPersons() {
    // TODO: 接入推荐算法；当前拉取人物列表
    return _api.listPersons();
  }

  Future<List<Book>> fetchRecommendedBooks() {
    return _api.listBooks();
  }

  /// 并行拉取三路数据并合并为 HomeState。
  Future<HomeState> fetchHomeState() async {
    final results = await Future.wait<dynamic>([
      fetchTimelineHighlights(),
      fetchRecommendedPersons(),
      fetchRecommendedBooks(),
    ]);
    return HomeState(
      dynasties: results[0] as List<Dynasty>,
      recommendedPersons: results[1] as List<Person>,
      recommendedBooks: results[2] as List<Book>,
    );
  }
}
