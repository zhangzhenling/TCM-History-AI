// 业务异常体系：将网络层与后端业务错误统一映射为 AppException。
//
// 对齐 PC 端 frontend/packages/api/src/http.ts 的错误处理：
// - HTTP 401 -> UnauthorizedException（触发刷新或登出）
// - HTTP 5xx / 超时 -> NetworkException / ServerException
// - 业务码非 0 -> BusinessException(code, message)

sealed class AppException implements Exception {
  final String message;
  const AppException(this.message);

  @override
  String toString() => '$runtimeType: $message';
}

/// 网络异常（超时、断网、DNS 失败等）。
class NetworkException extends AppException {
  const NetworkException(super.message);
}

/// 未授权（HTTP 401 或 access token 失效）。
class UnauthorizedException extends AppException {
  const UnauthorizedException(super.message);
}

/// 服务器异常（HTTP 5xx）。
class ServerException extends AppException {
  const ServerException(super.message);
}

/// 业务异常：后端返回 code 非 0。
class BusinessException extends AppException {
  final int code;
  const BusinessException({required this.code, required String message})
      : super(message);
}
