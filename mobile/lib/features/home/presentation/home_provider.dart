// 首页 Provider：暴露异步 HomeState，UI 层通过 ref.watch(homeProvider) 消费。
//
// 当前骨架返回 HomeState.placeholder() 以便在无后端时展示占位内容。
// 接入真实后端时，替换 build() 为：
//   final repo = ref.read(homeRepositoryProvider);
//   return repo.fetchHomeState();

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../domain/home_state.dart';

final homeProvider = FutureProvider<HomeState>((ref) async {
  // TODO: 接入 homeRepository 拉取真实数据
  // final repo = ref.read(homeRepositoryProvider);
  // return repo.fetchHomeState();
  return HomeState.placeholder();
});
