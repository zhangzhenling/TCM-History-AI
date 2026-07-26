// 共享实体 fromJson / toJson 往返测试：验证 snake_case 字段映射。

import 'package:flutter_test/flutter_test.dart';

import 'package:tcm_history_ai/shared/models/person.dart';
import 'package:tcm_history_ai/shared/models/book.dart';
import 'package:tcm_history_ai/shared/models/dynasty.dart';
import 'package:tcm_history_ai/shared/models/school.dart';
import 'package:tcm_history_ai/shared/models/history_event.dart';
import 'package:tcm_history_ai/shared/models/search_hit.dart';

void main() {
  group('Person', () {
    test('fromJson 应正确映射 snake_case 字段', () {
      final json = {
        'id': 1,
        'name': '张仲景',
        'courtesy_name': '机',
        'alias_name': '医圣',
        'dynasty_id': 2,
        'birth_year': 150,
        'death_year': 219,
        'gender': '男',
        'title': '医圣',
        'biography': '东汉医学家',
        'achievements': '确立辨证论治',
        'portrait_url': 'http://example.com/p.png',
      };
      final p = Person.fromJson(json);
      expect(p.id, 1);
      expect(p.name, '张仲景');
      expect(p.courtesyName, '机');
      expect(p.aliasName, '医圣');
      expect(p.dynastyId, 2);
      expect(p.birthYear, 150);
      expect(p.deathYear, 219);
      expect(p.title, '医圣');
      expect(p.portraitUrl, 'http://example.com/p.png');
    });

    test('toJson 应输出 snake_case 字段', () {
      const p = Person(
        id: 2,
        name: '华佗',
        courtesyName: '元化',
        dynastyId: 2,
        birthYear: 145,
        deathYear: 208,
        title: '神医',
      );
      final json = p.toJson();
      expect(json['id'], 2);
      expect(json['name'], '华佗');
      expect(json['courtesy_name'], '元化');
      expect(json['dynasty_id'], 2);
      expect(json['title'], '神医');
    });
  });

  group('Book', () {
    test('fromJson 应正确映射字段，缺失字段走默认值', () {
      final b = Book.fromJson({'id': 1, 'title': '黄帝内经'});
      expect(b.id, 1);
      expect(b.title, '黄帝内经');
      expect(b.dynastyId, 0);
      expect(b.isExtant, isTrue);
    });
  });

  group('Dynasty', () {
    test('fromJson / toJson 往返一致', () {
      final json = {
        'id': 1,
        'name': '汉',
        'start_year': -206,
        'end_year': 220,
        'sort_order': 2,
        'description': '汉代医学',
      };
      final d = Dynasty.fromJson(json);
      expect(d.startYear, -206);
      expect(d.sortOrder, 2);
      expect(d.toJson(), json);
    });
  });

  group('School', () {
    test('fromJson 应正确映射 founder_person_id', () {
      final s = School.fromJson({
        'id': 1,
        'name': '伤寒学派',
        'dynasty_id': 2,
        'founder_person_id': 1,
        'summary': '...',
        'established_year': 200,
      });
      expect(s.founderPersonId, 1);
      expect(s.establishedYear, 200);
    });
  });

  group('HistoryEvent', () {
    test('fromJson 应正确映射 occurred_year / event_type', () {
      final e = HistoryEvent.fromJson({
        'id': 1,
        'title': '《伤寒杂病论》成书',
        'dynasty_id': 2,
        'occurred_year': 200,
        'event_type': 'publication',
        'description': '...',
      });
      expect(e.occurredYear, 200);
      expect(e.eventType, 'publication');
    });
  });

  group('SearchHit', () {
    test('fromJson 应保留 source 原始字段', () {
      final hit = SearchHit.fromJson({
        'type': 'person',
        'id': 1,
        'score': 0.92,
        'source': {'name': '张仲景', 'title': '医圣'},
      });
      expect(hit.type, 'person');
      expect(hit.id, 1);
      expect(hit.score, 0.92);
      expect(hit.source['name'], '张仲景');
    });

    test('SearchParams.toQuery 应输出 types 数组为逗号串', () {
      const p = SearchParams(
        q: '伤寒',
        types: ['person', 'book'],
        page: 1,
        pageSize: 20,
      );
      final q = p.toQuery();
      expect(q['q'], '伤寒');
      expect(q['types'], 'person,book');
      expect(q['page'], 1);
      expect(q['page_size'], 20);
    });
  });
}
