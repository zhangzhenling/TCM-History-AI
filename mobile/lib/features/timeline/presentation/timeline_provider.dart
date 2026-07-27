import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../data/timeline_repository.dart';
import '../domain/timeline_state.dart';

final timelineProvider = FutureProvider<TimelineState>((ref) async {
  final repo = ref.read(timelineRepositoryProvider);
  return repo.fetchTimeline();
});
