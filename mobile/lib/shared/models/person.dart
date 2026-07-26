// 人物实体，对齐 frontend/packages/api/src/modules/history-types.ts 的 Person。

import 'package:flutter/foundation.dart';

@immutable
class Person {
  final int id;
  final String name;
  final String courtesyName;
  final String aliasName;
  final int dynastyId;
  final int birthYear;
  final int deathYear;
  final String gender;
  final String title;
  final String biography;
  final String achievements;
  final String portraitUrl;

  const Person({
    required this.id,
    required this.name,
    this.courtesyName = '',
    this.aliasName = '',
    this.dynastyId = 0,
    this.birthYear = 0,
    this.deathYear = 0,
    this.gender = '',
    this.title = '',
    this.biography = '',
    this.achievements = '',
    this.portraitUrl = '',
  });

  factory Person.fromJson(Map<String, dynamic> json) {
    return Person(
      id: json['id'] as int,
      name: json['name'] as String? ?? '',
      courtesyName: json['courtesy_name'] as String? ?? '',
      aliasName: json['alias_name'] as String? ?? '',
      dynastyId: json['dynasty_id'] as int? ?? 0,
      birthYear: json['birth_year'] as int? ?? 0,
      deathYear: json['death_year'] as int? ?? 0,
      gender: json['gender'] as String? ?? '',
      title: json['title'] as String? ?? '',
      biography: json['biography'] as String? ?? '',
      achievements: json['achievements'] as String? ?? '',
      portraitUrl: json['portrait_url'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'courtesy_name': courtesyName,
        'alias_name': aliasName,
        'dynasty_id': dynastyId,
        'birth_year': birthYear,
        'death_year': deathYear,
        'gender': gender,
        'title': title,
        'biography': biography,
        'achievements': achievements,
        'portrait_url': portraitUrl,
      };
}
