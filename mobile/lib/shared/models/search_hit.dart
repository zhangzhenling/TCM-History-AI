// 跨实体检索响应，对齐 frontend/packages/api/src/modules/history-types.ts 的
// SearchHit / SearchResponse / SearchParams。

import 'package:flutter/foundation.dart';

/// 检索命中项。type 标识实体类型（person/book/school/event/...），source 为原始字段。
@immutable
class SearchHit {
  final String type;
  final int id;
  final double? score;
  final Map<String, dynamic> source;

  const SearchHit({
    required this.type,
    required this.id,
    this.score,
    required this.source,
  });

  factory SearchHit.fromJson(Map<String, dynamic> json) {
    return SearchHit(
      type: json['type'] as String? ?? '',
      id: json['id'] as int? ?? 0,
      score: (json['score'] as num?)?.toDouble(),
      source: (json['source'] as Map?)?.cast<String, dynamic>() ?? const {},
    );
  }

  Map<String, dynamic> toJson() => {
        'type': type,
        'id': id,
        if (score != null) 'score': score,
        'source': source,
      };
}

/// 检索响应。
@immutable
class SearchResponse {
  final int total;
  final List<SearchHit> items;

  const SearchResponse({required this.total, required this.items});

  factory SearchResponse.fromJson(Map<String, dynamic> json) {
    return SearchResponse(
      total: json['total'] as int? ?? 0,
      items: (json['items'] as List? ?? const [])
          .map((e) => SearchHit.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }
}

/// 检索参数，对齐 TS SearchParams。
@immutable
class SearchParams {
  final String q;
  final List<String>? types;
  final int? page;
  final int? pageSize;

  const SearchParams({
    required this.q,
    this.types,
    this.page,
    this.pageSize,
  });

  Map<String, dynamic> toQuery() {
    final m = <String, dynamic>{'q': q};
    if (types != null && types!.isNotEmpty) m['types'] = types!.join(',');
    if (page != null) m['page'] = page;
    if (pageSize != null) m['page_size'] = pageSize;
    return m;
  }
}
