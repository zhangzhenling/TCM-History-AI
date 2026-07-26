// 检索状态：查询词 + 命中结果。对齐 PRD §4.3 与 doc/13-移动端设计.md 检索入口。
//
// 跨实体检索覆盖人物/著作/学派/事件，结果以 SearchHit 统一表达（type 区分实体类型）。

import 'package:flutter/foundation.dart';

import '../../../shared/models/search_hit.dart';

@immutable
class SearchState {
  final String query;
  final List<SearchHit> hits;
  final int total;

  const SearchState({
    this.query = '',
    this.hits = const [],
    this.total = 0,
  });

  bool get isEmpty => hits.isEmpty;

  /// 占位结果，用于骨架阶段展示。
  factory SearchState.placeholder(String query) {
    return SearchState(
      query: query,
      total: 4,
      hits: [
        SearchHit(
            type: 'person',
            id: 1,
            score: 0.95,
            source: const {'name': '张仲景', 'title': '医圣', 'dynasty': '汉'}),
        SearchHit(
            type: 'book',
            id: 2,
            score: 0.91,
            source: const {'title': '伤寒杂病论', 'category': '临床'}),
        SearchHit(
            type: 'school',
            id: 1,
            score: 0.82,
            source: const {'name': '伤寒学派', 'summary': '以研究《伤寒论》为中心'}),
        SearchHit(
            type: 'event',
            id: 3,
            score: 0.75,
            source: const {'title': '《伤寒杂病论》成书', 'occurred_year': 200}),
      ],
    );
  }
}
