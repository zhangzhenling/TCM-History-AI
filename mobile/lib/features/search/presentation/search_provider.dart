// 检索 Provider：
// - searchQueryProvider 持有当前查询词（绑定搜索框）
// - searchProvider 根据 query 触发检索（当前返回占位结果）
//
// 接入真实后端时，searchProvider 改为：
//   final repo = ref.read(searchRepositoryProvider);
//   return repo.search(query);
//
// 后续可引入防抖（debounce）避免高频请求。

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../domain/search_state.dart';

/// 当前查询词。
final searchQueryProvider = StateProvider<String>((ref) => '');

/// 检索结果：依赖 searchQueryProvider，query 为空时返回空态。
final searchProvider = FutureProvider<SearchState>((ref) async {
  final query = ref.watch(searchQueryProvider);
  if (query.trim().isEmpty) {
    return const SearchState();
  }
  // TODO: 接入 searchRepository 拉取真实结果
  // final repo = ref.read(searchRepositoryProvider);
  // return repo.search(query);
  await Future<void>.delayed(const Duration(milliseconds: 200));
  return SearchState.placeholder(query);
});
