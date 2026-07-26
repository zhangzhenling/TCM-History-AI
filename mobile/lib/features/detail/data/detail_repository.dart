// 详情 Repository：根据实体类型拉取对应实体详情。
//
// 对齐 doc/13-移动端设计.md §四 人物/经典/学派详情页：
//   GET /api/v1/history/persons/{id}      -> Person
//   GET /api/v1/history/books/{id}        -> Book
//   GET /api/v1/history/schools/{id}      -> School
//
// Repository 是业务逻辑聚合点：根据 entityType 路由到对应 API，
// 并将结果统一封装为 DetailState 供 Provider 消费。

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';
import '../domain/detail_state.dart';

final detailRepositoryProvider = Provider<DetailRepository>((ref) {
  return DetailRepository(ref.read(historyApiClientProvider));
});

class DetailRepository {
  DetailRepository(this._api);
  final HistoryApiClient _api;

  /// 按实体类型拉取详情，返回 DetailState。
  Future<DetailState> fetchDetail({
    required String entityType,
    required String entityId,
  }) async {
    switch (entityType) {
      case 'person':
        final person = await _api.getPerson(entityId);
        return DetailState(
          entityType: entityType,
          entityId: entityId,
          person: person,
        );
      case 'book':
        final book = await _api.getBook(entityId);
        return DetailState(
          entityType: entityType,
          entityId: entityId,
          book: book,
        );
      case 'school':
        final school = await _api.getSchool(entityId);
        return DetailState(
          entityType: entityType,
          entityId: entityId,
          school: school,
        );
      default:
        throw ArgumentError('不支持的实体类型：$entityType');
    }
  }
}
