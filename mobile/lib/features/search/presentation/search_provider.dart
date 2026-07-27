import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../data/search_repository.dart';
import '../domain/search_state.dart';

final searchQueryProvider = StateProvider<String>((ref) => '');

final searchProvider = FutureProvider<SearchState>((ref) async {
  final query = ref.watch(searchQueryProvider);
  if (query.trim().isEmpty) {
    return const SearchState();
  }
  final repo = ref.read(searchRepositoryProvider);
  return repo.search(query);
});
