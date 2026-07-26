// 详情 Provider：按 family（entityType + entityId）暴露异步 DetailState。
//
// 使用 FutureProvider.family 接收路径参数 type 与 id，对齐路由
// /detail/:type/:id。UI 层通过 ref.watch(detailProvider((type, id))) 消费。
//
// 当前骨架返回 DetailState.placeholder() 以便在无后端时展示占位内容。
// 接入真实后端时，替换 build() 为：
//   final repo = ref.read(detailRepositoryProvider);
//   return repo.fetchDetail(entityType: type, entityId: id);

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../domain/detail_state.dart';

/// 详情参数：实体类型 + 实体 ID。
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

/// 详情 Provider：按 (type, id) 维度缓存与刷新。
final detailProvider =
    FutureProvider.family<DetailState, DetailParams>((ref, params) async {
  // TODO: 接入 detailRepository 拉取真实数据
  // final repo = ref.read(detailRepositoryProvider);
  // return repo.fetchDetail(entityType: params.entityType, entityId: params.entityId);
  await Future<void>.delayed(const Duration(milliseconds: 200));
  return DetailState.placeholder(params.entityType, params.entityId);
});
