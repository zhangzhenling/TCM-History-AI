// 历史事件实体，对齐 frontend/packages/api/src/modules/history-types.ts 的 HistoryEvent。
// 注意：TS 接口名为 HistoryEvent，此处沿用同名以保持一一对应。

import 'package:flutter/foundation.dart';

@immutable
class HistoryEvent {
  final int id;
  final String title;
  final int dynastyId;
  final int occurredYear;
  final String eventType;
  final String description;
  final String impact;
  final String location;

  const HistoryEvent({
    required this.id,
    required this.title,
    required this.dynastyId,
    required this.occurredYear,
    required this.eventType,
    this.description = '',
    this.impact = '',
    this.location = '',
  });

  factory HistoryEvent.fromJson(Map<String, dynamic> json) {
    return HistoryEvent(
      id: json['id'] as int,
      title: json['title'] as String? ?? '',
      dynastyId: json['dynasty_id'] as int? ?? 0,
      occurredYear: json['occurred_year'] as int? ?? 0,
      eventType: json['event_type'] as String? ?? '',
      description: json['description'] as String? ?? '',
      impact: json['impact'] as String? ?? '',
      location: json['location'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'title': title,
        'dynasty_id': dynastyId,
        'occurred_year': occurredYear,
        'event_type': eventType,
        'description': description,
        'impact': impact,
        'location': location,
      };
}
