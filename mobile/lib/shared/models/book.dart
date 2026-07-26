// 著作实体，对齐 frontend/packages/api/src/modules/history-types.ts 的 Book。

import 'package:flutter/foundation.dart';

@immutable
class Book {
  final int id;
  final String title;
  final int dynastyId;
  final int publishedYear;
  final String category;
  final String summary;
  final int volumeCount;
  final bool isExtant;
  final String fileUrl;

  const Book({
    required this.id,
    required this.title,
    this.dynastyId = 0,
    this.publishedYear = 0,
    this.category = '',
    this.summary = '',
    this.volumeCount = 0,
    this.isExtant = true,
    this.fileUrl = '',
  });

  factory Book.fromJson(Map<String, dynamic> json) {
    return Book(
      id: json['id'] as int,
      title: json['title'] as String? ?? '',
      dynastyId: json['dynasty_id'] as int? ?? 0,
      publishedYear: json['published_year'] as int? ?? 0,
      category: json['category'] as String? ?? '',
      summary: json['summary'] as String? ?? '',
      volumeCount: json['volume_count'] as int? ?? 0,
      isExtant: json['is_extant'] as bool? ?? true,
      fileUrl: json['file_url'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'title': title,
        'dynasty_id': dynastyId,
        'published_year': publishedYear,
        'category': category,
        'summary': summary,
        'volume_count': volumeCount,
        'is_extant': isExtant,
        'file_url': fileUrl,
      };
}
