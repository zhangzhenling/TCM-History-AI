// ApiResult<T>：统一返回包装，将异步数据操作结果表达为成功/失败两态。
//
// Repository 层返回 Future<ApiResult<T>>，UI 层通过 pattern matching 处理。
// 对齐 doc/13-移动端设计.md §六的 ApiResult 设计。

import '../error/app_exception.dart';

sealed class ApiResult<T> {
  const ApiResult();

  /// 模式匹配辅助：成功返回 data，失败抛出异常。
  T getOrThrow();
}

class ApiSuccess<T> extends ApiResult<T> {
  final T data;
  const ApiSuccess(this.data);

  @override
  T getOrThrow() => data;
}

class ApiFailure<T> extends ApiResult<T> {
  final AppException error;
  const ApiFailure(this.error);

  @override
  T getOrThrow() => throw error;
}
