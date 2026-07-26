// AppException 体系单元测试：验证 sealed 异常类的构造与 toString。

import 'package:flutter_test/flutter_test.dart';

import 'package:tcm_history_ai/core/error/app_exception.dart';

void main() {
  test('NetworkException 应承载 message 并在 toString 中输出', () {
    const e = NetworkException('网络异常');
    expect(e.message, '网络异常');
    expect(e.toString(), contains('NetworkException'));
    expect(e.toString(), contains('网络异常'));
  });

  test('UnauthorizedException 应承载 message', () {
    const e = UnauthorizedException('未授权');
    expect(e.message, '未授权');
    expect(e, isA<AppException>());
  });

  test('ServerException 应承载 message', () {
    const e = ServerException('服务器异常');
    expect(e.message, '服务器异常');
    expect(e, isA<AppException>());
  });

  test('BusinessException 应承载 code 与 message', () {
    const e = BusinessException(code: 40001, message: '参数错误');
    expect(e.code, 40001);
    expect(e.message, '参数错误');
    expect(e.toString(), contains('40001'));
  });

  test('所有具体异常都应是 AppException 的子类型', () {
    expect(const NetworkException('a'), isA<AppException>());
    expect(const UnauthorizedException('b'), isA<AppException>());
    expect(const ServerException('c'), isA<AppException>());
    const BusinessException(code: 1, message: 'd');
  });
}
