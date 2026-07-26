// 学派实体，对齐 frontend/packages/api/src/modules/history-types.ts 的 School。

import 'package:flutter/foundation.dart';

@immutable
class School {
  final int id;
  final String name;
  final int dynastyId;
  final int founderPersonId;
  final String summary;
  final int establishedYear;

  const School({
    required this.id,
    required this.name,
    this.dynastyId = 0,
    this.founderPersonId = 0,
    this.summary = '',
    this.establishedYear = 0,
  });

  factory School.fromJson(Map<String, dynamic> json) {
    return School(
      id: json['id'] as int,
      name: json['name'] as String? ?? '',
      dynastyId: json['dynasty_id'] as int? ?? 0,
      founderPersonId: json['founder_person_id'] as int? ?? 0,
      summary: json['summary'] as String? ?? '',
      establishedYear: json['established_year'] as int? ?? 0,
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'dynasty_id': dynastyId,
        'founder_person_id': founderPersonId,
        'summary': summary,
        'established_year': establishedYear,
      };
}
