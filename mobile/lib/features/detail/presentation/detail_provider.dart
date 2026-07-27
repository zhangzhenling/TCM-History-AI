import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../data/detail_repository.dart';
import '../domain/detail_state.dart';

class DetailParams {
  final String entityType;
  final String entityId;
  const DetailParams({required this.entityType, required this.entityId});

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DetailParams &&
          entityType == other.entityType &&
          entityId == other.entityId;

  @override
  int get hashCode => Object.hash(entityType, entityId);
}

final detailProvider =
    FutureProvider.family<DetailState, DetailParams>((ref, params) async {
  final repo = ref.read(detailRepositoryProvider);
  return repo.fetchDetail(entityType: params.entityType, entityId: params.entityId);
});
