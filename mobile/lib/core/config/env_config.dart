// 环境配置：API 基址、超时时长。
//
// apiBaseUrl 通过 --dart-define=API_BASE_URL=... 覆盖，默认指向 Android 模拟机
// 访问宿主机服务的 10.0.2.2。真机调试时改为主机局域网 IP。

class EnvConfig {
  const EnvConfig._();

  /// 后端 API 基址，对齐 PC 端 baseURL（/api/v1 前缀）。
  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://10.0.2.2:8080/api/v1',
  );

  /// 连接超时（毫秒）。
  static const int connectTimeoutMs = 10000;

  /// 接收超时（毫秒）。
  static const int receiveTimeoutMs = 30000;
}
