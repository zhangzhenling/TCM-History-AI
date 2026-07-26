// 统一响应包络与分页结构，对齐 frontend/packages/api/src/types.ts。
//
// 后端统一响应体：{ code, message, data?, trace_id? }，code=0 表示成功。
// 分页列表响应：{ page, page_size, total, total_page, items }。

/// 后端统一响应体。
class ApiEnvelope<T> {
  final int code;
  final String message;
  final T? data;
  final String? traceId;

  const ApiEnvelope({
    required this.code,
    required this.message,
    this.data,
    this.traceId,
  });

  bool get isOk => code == 0;

  factory ApiEnvelope.fromJson(
    Map<String, dynamic> json,
    T Function(Object? data) fromData,
  ) {
    return ApiEnvelope<T>(
      code: json['code'] as int? ?? -1,
      message: json['message'] as String? ?? '',
      data: json.containsKey('data') ? fromData(json['data']) : null,
      traceId: json['trace_id'] as String?,
    );
  }

  Map<String, dynamic> toJson(Object? Function(T value) toData) {
    return {
      'code': code,
      'message': message,
      if (data != null) 'data': toData(data as T),
      if (traceId != null) 'trace_id': traceId,
    };
  }
}

/// 分页列表响应（后端 dto.ListResponse）。
class ListResponse<T> {
  final int page;
  final int pageSize;
  final int total;
  final int totalPage;
  final List<T> items;

  const ListResponse({
    required this.page,
    required this.pageSize,
    required this.total,
    required this.totalPage,
    required this.items,
  });

  factory ListResponse.fromJson(
    Map<String, dynamic> json,
    T Function(Map<String, dynamic>) fromJsonItem,
  ) {
    return ListResponse<T>(
      page: json['page'] as int? ?? 1,
      pageSize: json['page_size'] as int? ?? 20,
      total: json['total'] as int? ?? 0,
      totalPage: json['total_page'] as int? ?? 0,
      items: (json['items'] as List? ?? const [])
          .map((e) => fromJsonItem(e as Map<String, dynamic>))
          .toList(),
    );
  }
}
