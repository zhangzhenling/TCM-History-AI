import 'package:flutter/foundation.dart';

@immutable
class UserProfile {
  final int id;
  final String username;
  final String nickname;
  final String avatarUrl;
  final String bio;

  const UserProfile({
    required this.id,
    this.username = '',
    this.nickname = '',
    this.avatarUrl = '',
    this.bio = '',
  });

  factory UserProfile.fromJson(Map<String, dynamic> json) {
    return UserProfile(
      id: json['user_id'] as int? ?? 0,
      username: json['username'] as String? ?? '',
      nickname: json['nickname'] as String? ?? '',
      avatarUrl: json['avatar_url'] as String? ?? '',
      bio: json['bio'] as String? ?? '',
    );
  }
}