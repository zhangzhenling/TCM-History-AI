// 检索 Repository：封装跨实体检索调用。
//
// 调用 GET /api/v1/history/search?q={q}&types={types}，对齐 PC 端
// frontend/packages/api/src/modules/history.ts 的 search 方法。

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';
import '../../../shared/models/search_hit.dart';
import '../domain/search_state.dart';

final searchRepositoryProvider = Provider<SearchRepository>((ref) {
  return SearchRepository(ref.read(historyApiClientProvider));
});

class SearchRepository {
  SearchRepository(this._api);
  final HistoryApiClient _api;

  /// 执行跨实体检索。types 为空表示检索全部实体类型。
  Future<SearchState> search(String query, {List<String>? types}) async {
    if (query.trim().isEmpty) {
      return const SearchState();
    }
    final response = await _api.search(SearchParams(
      q: query,
      types: types,
      page: 1,
      pageSize: 20,
    ));
    return SearchState(
      query: query,
      hits: response.items,
      total: response.total,
    );
  }
}
