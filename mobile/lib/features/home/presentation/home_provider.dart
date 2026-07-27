import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../data/home_repository.dart';
import '../domain/home_state.dart';

final homeProvider = FutureProvider<HomeState>((ref) async {
  final repo = ref.read(homeRepositoryProvider);
  return repo.fetchHomeState();
});
