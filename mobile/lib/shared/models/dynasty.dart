// 朝代实体，对齐 frontend/packages/api/src/modules/history-types.ts 的 Dynasty。
// 字段沿用后端 snake_case 命名。

import 'package:flutter/foundation.dart';

@immutable
class Dynasty {
  final int id;
  final String name;
  final int startYear;
  final int endYear;
  final int sortOrder;
  final String description;

  const Dynasty({
    required this.id,
    required this.name,
    required this.startYear,
    required this.endYear,
    required this.sortOrder,
    required this.description,
  });

  factory Dynasty.fromJson(Map<String, dynamic> json) {
    return Dynasty(
      id: json['id'] as int,
      name: json['name'] as String? ?? '',
      startYear: json['start_year'] as int? ?? 0,
      endYear: json['end_year'] as int? ?? 0,
      sortOrder: json['sort_order'] as int? ?? 0,
      description: json['description'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'start_year': startYear,
        'end_year': endYear,
        'sort_order': sortOrder,
        'description': description,
      };
}
