// 详情状态：按实体类型（person/book/school）承载不同实体。
//
// 对齐 doc/13-移动端设计.md §四 人物/经典详情页：
//   GET /api/v1/knowledge/figures/{id} 或 /api/v1/knowledge/classics/{id}
// 当前 MVP 复用 History Service 的 /history/persons/:id、/history/books/:id、
// /history/schools/:id。

import 'package:flutter/foundation.dart';

import '../../../shared/models/book.dart';
import '../../../shared/models/person.dart';
import '../../../shared/models/school.dart';

@immutable
class DetailState {
  final String entityType;
  final String entityId;
  final Person? person;
  final Book? book;
  final School? school;

  const DetailState({
    required this.entityType,
    required this.entityId,
    this.person,
    this.book,
    this.school,
  });

  /// 占位数据，用于骨架阶段展示。
  factory DetailState.placeholder(String type, String id) {
    switch (type) {
      case 'person':
        return DetailState(
          entityType: type,
          entityId: id,
          person: Person(
            id: int.tryParse(id) ?? 0,
            name: '张仲景',
            dynastyId: 2,
            birthYear: 150,
            deathYear: 219,
            title: '医圣',
            biography: '东汉医学家，被后世尊为「医圣」。广泛学习医书，结合临床经验著成《伤寒杂病论》。',
            achievements: '确立六经辨证体系，奠定中医临床医学基础。',
          ),
        );
      case 'book':
        return DetailState(
          entityType: type,
          entityId: id,
          book: Book(
            id: int.tryParse(id) ?? 0,
            title: '伤寒杂病论',
            dynastyId: 2,
            publishedYear: 200,
            category: '临床',
            summary: '张仲景著，确立了辨证论治体系，分《伤寒论》与《金匮要略》两部分。',
            volumeCount: 16,
            isExtant: true,
          ),
        );
      case 'school':
        return DetailState(
          entityType: type,
          entityId: id,
          school: School(
            id: int.tryParse(id) ?? 0,
            name: '伤寒学派',
            dynastyId: 2,
            founderPersonId: 1,
            summary: '以研究、阐发《伤寒论》为中心的医学流派，代表人物包括张仲景、成无己等。',
            establishedYear: 200,
          ),
        );
      default:
        return DetailState(entityType: type, entityId: id);
    }
  }
}
