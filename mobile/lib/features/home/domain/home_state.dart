// 首页状态：聚合朝代时间线、推荐人物、推荐著作三路数据。
//
// 对齐 doc/13-移动端设计.md §四首页设计：
//   GET /api/v1/learning/today、/history/timeline/highlights、/learning/continue
// 三路并行合并为 HomeState。当前 MVP 仅取朝代与推荐实体两路。

import 'package:flutter/foundation.dart';

import '../../../shared/models/book.dart';
import '../../../shared/models/dynasty.dart';
import '../../../shared/models/person.dart';

@immutable
class HomeState {
  final List<Dynasty> dynasties;
  final List<Person> recommendedPersons;
  final List<Book> recommendedBooks;

  const HomeState({
    required this.dynasties,
    required this.recommendedPersons,
    required this.recommendedBooks,
  });

  /// 占位数据，用于骨架阶段展示。后续接入 homeRepository 后替换为真实数据。
  factory HomeState.placeholder() => const HomeState(
        dynasties: [
          Dynasty(
              id: 1,
              name: '先秦',
              startYear: -2070,
              endYear: -221,
              sortOrder: 1,
              description: '中医起源与原始积累'),
          Dynasty(
              id: 2,
              name: '汉',
              startYear: -206,
              endYear: 220,
              sortOrder: 2,
              description: '《黄帝内经》《伤寒论》成书'),
          Dynasty(
              id: 3,
              name: '魏晋',
              startYear: 220,
              endYear: 589,
              sortOrder: 3,
              description: '王叔和《脉经》、皇甫谧《针灸甲乙经》'),
          Dynasty(
              id: 4,
              name: '隋唐',
              startYear: 581,
              endYear: 907,
              sortOrder: 4,
              description: '《新修本草》、孙思邈《千金方》'),
          Dynasty(
              id: 5,
              name: '宋金元',
              startYear: 960,
              endYear: 1368,
              sortOrder: 5,
              description: '金元四大家争鸣'),
        ],
        recommendedPersons: [
          Person(
              id: 1,
              name: '张仲景',
              dynastyId: 2,
              birthYear: 150,
              deathYear: 219,
              title: '医圣',
              biography: '东汉医学家，《伤寒杂病论》作者，确立辨证论治体系'),
          Person(
              id: 2,
              name: '华佗',
              dynastyId: 2,
              birthYear: 145,
              deathYear: 208,
              title: '神医',
              biography: '外科鼻祖，发明麻沸散，创五禽戏'),
          Person(
              id: 3,
              name: '孙思邈',
              dynastyId: 4,
              birthYear: 581,
              deathYear: 682,
              title: '药王',
              biography: '唐代医学家，《千金要方》作者，倡导医德'),
        ],
        recommendedBooks: [
          Book(
              id: 1,
              title: '黄帝内经',
              dynastyId: 2,
              publishedYear: -100,
              category: '医经',
              summary: '中医理论奠基之作，分《素问》《灵枢》'),
          Book(
              id: 2,
              title: '伤寒杂病论',
              dynastyId: 2,
              publishedYear: 200,
              category: '临床',
              summary: '张仲景著，确立六经辨证体系'),
          Book(
              id: 3,
              title: '本草纲目',
              dynastyId: 6,
              publishedYear: 1578,
              category: '本草',
              summary: '李时珍著，集本草学大成'),
        ],
      );
}
