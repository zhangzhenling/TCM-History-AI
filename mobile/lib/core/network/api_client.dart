// 类型化 API 客户端，对齐 frontend/packages/api/src/modules/ 下的 history.ts 与 auth.ts。
//
// 当前为手写封装（基于 Dio）；后续可切换为 retrofit + retrofit_generator 代码生成：
//   1. 将类改为 abstract class，加 @RestApi() 注解与 part 'api_client.g.dart';
//   2. 方法加 @GET/@POST 注解；
//   3. 运行 flutter pub run build_runner build 生成实现类。
//
// 端点对齐 backend history-service / user-service 路由（/api/v1 前缀已由 baseUrl 注入）。

import 'package:dio/dio.dart';

import '../../shared/models/book.dart';
import '../../shared/models/dynasty.dart';
import '../../shared/models/history_event.dart';
import '../../shared/models/person.dart';
import '../../shared/models/school.dart';
import '../../shared/models/search_hit.dart';
import '../error/app_exception.dart';

/// History Service 类型化客户端。
class HistoryApiClient {
  HistoryApiClient(this._dio);
  final Dio _dio;

  // ---- Dynasties ----
  Future<List<Dynasty>> listDynasties({int page = 1, int pageSize = 20}) async {
    final res = await _dio.get(
      '/history/dynasties',
      queryParameters: {'page': page, 'page_size': pageSize},
    );
    return _unwrapList(res.data, Dynasty.fromJson);
  }

  Future<Dynasty> getDynasty(String id) async {
    final res = await _dio.get('/history/dynasties/$id');
    return _unwrap(res.data, Dynasty.fromJson);
  }

  // ---- Persons ----
  Future<List<Person>> listPersons({int? dynastyId, String? keyword}) async {
    final res = await _dio.get(
      '/history/persons',
      queryParameters: {
        if (dynastyId != null) 'dynasty_id': dynastyId,
        if (keyword != null && keyword.isNotEmpty) 'keyword': keyword,
      },
    );
    return _unwrapList(res.data, Person.fromJson);
  }

  Future<Person> getPerson(String id) async {
    final res = await _dio.get('/history/persons/$id');
    return _unwrap(res.data, Person.fromJson);
  }

  // ---- Books ----
  Future<List<Book>> listBooks({int? dynastyId, String? category}) async {
    final res = await _dio.get(
      '/history/books',
      queryParameters: {
        if (dynastyId != null) 'dynasty_id': dynastyId,
        if (category != null && category.isNotEmpty) 'category': category,
      },
    );
    return _unwrapList(res.data, Book.fromJson);
  }

  Future<Book> getBook(String id) async {
    final res = await _dio.get('/history/books/$id');
    return _unwrap(res.data, Book.fromJson);
  }

  // ---- Schools ----
  Future<List<School>> listSchools() async {
    final res = await _dio.get('/history/schools');
    return _unwrapList(res.data, School.fromJson);
  }

  Future<School> getSchool(String id) async {
    final res = await _dio.get('/history/schools/$id');
    return _unwrap(res.data, School.fromJson);
  }

  // ---- Events ----
  Future<List<HistoryEvent>> listEvents({int? dynastyId}) async {
    final res = await _dio.get(
      '/history/events',
      queryParameters: {if (dynastyId != null) 'dynasty_id': dynastyId},
    );
    return _unwrapList(res.data, HistoryEvent.fromJson);
  }

  // ---- Search ----
  Future<SearchResponse> search(SearchParams params) async {
    final res = await _dio.get('/history/search', queryParameters: params.toQuery());
    final data = _unwrapRaw(res.data) as Map<String, dynamic>;
    return SearchResponse.fromJson(data);
  }

  // ---- 包络解包辅助 ----
  dynamic _unwrapRaw(dynamic body) {
    if (body is! Map<String, dynamic>) {
      throw const ServerException('响应格式异常');
    }
    final code = body['code'] as int? ?? -1;
    if (code != 0) {
      throw BusinessException(
        code: code,
        message: body['message'] as String? ?? '请求失败',
      );
    }
    return body['data'];
  }

  T _unwrap<T>(dynamic body, T Function(Map<String, dynamic>) fromJson) {
    final data = _unwrapRaw(body);
    return fromJson(data as Map<String, dynamic>);
  }

  List<T> _unwrapList<T>(dynamic body, T Function(Map<String, dynamic>) fromJson) {
    final data = _unwrapRaw(body) as Map<String, dynamic>;
    final items = (data['items'] as List? ?? const [])
        .cast<Map<String, dynamic>>()
        .map(fromJson)
        .toList();
    return items;
  }
}

/// Auth Service 类型化客户端（登录 / 注册）。
class AuthApiClient {
  AuthApiClient(this._dio);
  final Dio _dio;

  Future<TokenPair> login({
    required String username,
    required String password,
  }) async {
    final res = await _dio.post('/auth/login', data: {
      'username': username,
      'password': password,
    });
    return _unwrap(res.data, TokenPair.fromJson);
  }

  Future<TokenPair> register({
    required String username,
    required String password,
    String? email,
    String? phone,
  }) async {
    final res = await _dio.post('/auth/register', data: {
      'username': username,
      'password': password,
      if (email != null) 'email': email,
      if (phone != null) 'phone': phone,
    });
    return _unwrap(res.data, TokenPair.fromJson);
  }

  static T _unwrap<T>(dynamic body, T Function(Map<String, dynamic>) fromJson) {
    if (body is! Map<String, dynamic>) {
      throw const ServerException('响应格式异常');
    }
    final code = body['code'] as int? ?? -1;
    if (code != 0) {
      throw BusinessException(
        code: code,
        message: body['message'] as String? ?? '请求失败',
      );
    }
    return fromJson(body['data'] as Map<String, dynamic>);
  }
}

/// 登录/注册返回的 token 对，对齐 TS TokenResponse。
class TokenPair {
  final String accessToken;
  final String refreshToken;
  final int expiresIn;
  final int userId;
  final String username;

  const TokenPair({
    required this.accessToken,
    required this.refreshToken,
    required this.expiresIn,
    required this.userId,
    required this.username,
  });

  factory TokenPair.fromJson(Map<String, dynamic> json) {
    return TokenPair(
      accessToken: json['access_token'] as String? ?? '',
      refreshToken: json['refresh_token'] as String? ?? '',
      expiresIn: json['expires_in'] as int? ?? 0,
      userId: json['user_id'] as int? ?? 0,
      username: json['username'] as String? ?? '',
    );
  }
}
