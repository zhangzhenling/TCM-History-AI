// 错误拦截器：
// 1. 响应方向：拆解统一响应包络 { code, message, data }，业务码非 0 视为错误，
//    转为 DioException 抛出（携带 BusinessException）。
// 2. 错误方向：将 HTTP 异常 / 网络异常映射为 AppException 体系，供 UI 层统一展示。
//
// 对齐 PC 端 frontend/packages/api/src/http.ts 的响应拦截器逻辑。

import 'package:dio/dio.dart';

import '../../error/app_exception.dart';

class ErrorInterceptor extends Interceptor {
  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    final body = response.data;
    // 仅处理 JSON Map 类型的统一包络；非 envelope 响应（如文件流）原样放行。
    if (body is Map<String, dynamic>) {
      final code = body['code'];
      if (code is int && code != 0) {
        final message = body['message'] as String? ?? '请求失败';
        handler.reject(
          DioException(
            requestOptions: response.requestOptions,
            response: response,
            type: DioExceptionType.badResponse,
            error: BusinessException(code: code, message: message),
          ),
        );
        return;
      }
    }
    handler.next(response);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    // 已经是 AppException 的错误直接放行，避免重复包装。
    if (err.error is AppException) {
      handler.next(err);
      return;
    }

    final status = err.response?.statusCode;
    final AppException mapped;
    if (status == 401) {
      mapped = const UnauthorizedException('未授权，请重新登录');
    } else if (status != null && status >= 500) {
      mapped = ServerException('服务器异常（$status）');
    } else if (err.type == DioExceptionType.connectionTimeout ||
        err.type == DioExceptionType.receiveTimeout ||
        err.type == DioExceptionType.connectionError) {
      mapped = const NetworkException('网络异常，请检查网络连接');
    } else {
      mapped = NetworkException(err.message);
    }

    handler.next(
      DioException(
        requestOptions: err.requestOptions,
        response: err.response,
        type: err.type,
        error: mapped,
      ),
    );
  }
}
