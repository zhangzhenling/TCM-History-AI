import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/token_storage.dart';
import '../data/profile_repository.dart';
import '../domain/profile_state.dart';

final profileProvider = FutureProvider<ProfileState>((ref) async {
  final tokenStorage = ref.watch(tokenStorageProvider);
  if (!tokenStorage.isLoggedIn) {
    return const ProfileState.guest();
  }
  final repo = ref.read(profileRepositoryProvider);
  return repo.fetchProfile();
});
