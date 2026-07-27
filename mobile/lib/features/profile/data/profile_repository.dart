import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';
import '../domain/profile_state.dart';

final profileRepositoryProvider = Provider<ProfileRepository>((ref) {
  return ProfileRepository(ref.read(historyApiClientProvider));
});

class ProfileRepository {
  ProfileRepository(this._api);
  final HistoryApiClient _api;

  Future<ProfileState> fetchProfile() async {
    final user = await _api.getUserProfile();
    return ProfileState(
      isLoggedIn: true,
      userId: user.id,
      username: user.username,
      nickname: user.nickname,
      avatarUrl: user.avatarUrl,
      bio: user.bio,
    );
  }
}