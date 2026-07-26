# TCM-History-AI 移动端

TCM-History-AI 中医发展史 AI 学习平台的 Flutter 移动端，定位为碎片化学习场景的补充入口，与 PC 端共用同一套后端 API（`/api/v1` 前缀，JWT 鉴权）。本目录为 P6 阶段移动端 MVP 工程骨架。

## 技术栈

| 维度 | 选型 |
| ---- | ---- |
| 框架 | Flutter 3.x + Dart 3 |
| 状态管理 | Riverpod（`flutter_riverpod`） |
| 路由 | `go_router`（声明式路由 + `ShellRoute` 持久化底部导航） |
| 网络 | `dio` + `retrofit`（类型化 API 客户端） |
| 本地存储 | `shared_preferences`（JWT token 持久化） |
| 序列化 | `json_annotation` + `json_serializable`（代码生成） |
| 主题 | Material 3，中医风格配色，中文优先 |

> 设计依据：`doc/13-移动端设计.md`、`doc/21-开发路线图.md`（P6 阶段）。

## 目录结构

```
mobile/
├── lib/
│   ├── main.dart                       # 入口：初始化 SharedPreferences + ProviderScope
│   ├── app.dart                        # MaterialApp.router 装配
│   ├── core/
│   │   ├── config/env_config.dart      # 环境配置（api_base_url 等）
│   │   ├── network/                    # Dio 客户端、拦截器、API 客户端、token 存储
│   │   │   ├── dio_client.dart
│   │   │   ├── api_client.dart         # 类型化 API 客户端（对齐 PC 端 packages/api）
│   │   │   ├── api_result.dart         # ApiResult<T> 统一返回包装
│   │   │   ├── token_storage.dart      # 基于 SharedPreferences 的 JWT 持久化
│   │   │   └── interceptors/
│   │   │       ├── auth_interceptor.dart     # JWT 注入 + 401 自动跳转登录
│   │   │       └── error_interceptor.dart    # 统一错误映射
│   │   ├── router/                     # go_router 路由表 + 认证守卫
│   │   ├── theme/                      # Material 3 主题 + 中医配色
│   │   └── error/app_exception.dart    # 业务异常体系
│   ├── features/                       # feature-first 分层（data/domain/presentation）
│   │   ├── home/                       # 首页
│   │   ├── search/                     # 跨实体检索
│   │   ├── detail/                     # 实体详情（人物/著作/学派，按 type 路由参数区分）
│   │   ├── timeline/                   # 朝代时间线
│   │   ├── profile/                    # 我的（登录入口、学习记录）
│   │   └── auth/                       # 登录/注册
│   └── shared/
│       ├── models/                     # 共享实体（Person/Book/School/Dynasty/Event 等）
│       └── widgets/tcm_scaffold.dart   # 统一页面骨架
├── assets/images/                      # 图片资源占位
├── test/                               # 单元测试与 widget 测试骨架
├── android/                            # Android 平台占位
├── ios/                                # iOS 平台占位
├── pubspec.yaml
├── analysis_options.yaml
└── README.md
```

每个 feature 模块采用三层分层：
- `data/`：repository，封装业务逻辑与数据来源（远程 API / 本地缓存）
- `domain/`：feature 级 state / entity
- `presentation/`：page、widget、provider

## MVP 五个核心页面（P6 范围）

1. **首页** `/home` — 朝代时间线入口、推荐人物/著作、搜索入口
2. **检索** `/search` — 跨实体检索（人物/著作/学派/事件）
3. **详情** `/detail/:type/:id` — 实体详情（type=person/book/school）
4. **时间线** `/timeline` — 朝代时间线
5. **我的** `/profile` — 登录/注册入口、学习记录

另有 `/login`、`/register` 两个认证页面，路由表共 7 条。

## 网络层设计

- `Dio` 单例装配 `ErrorInterceptor` + `AuthInterceptor`。
- `AuthInterceptor` 从 `SharedPreferences` 读取 `access_token`，注入 `Authorization: Bearer {token}` 头；收到 401 时清空 token 并跳转 `/login`。
- `ErrorInterceptor` 拆解统一响应包络（`{ code, message, data }`），业务码非 0 视为错误；将 HTTP 异常映射为 `AppException` 体系。
- `HistoryApiClient` 为类型化 API 客户端，端点对齐 `frontend/packages/api/src/modules/history.ts`（`/history/dynasties`、`/history/persons/:id`、`/history/search` 等）。

> 后续可启用 `retrofit` + `retrofit_generator` 代码生成替换手写封装：运行 `flutter pub run build_runner build` 生成 `*.g.dart`。

## 运行方式

```bash
# 1. 安装依赖
flutter pub get

# 2.（可选）启用 retrofit / json_serializable 代码生成
flutter pub run build_runner build --delete-conflicting-outputs

# 3. 运行（默认指向 Android 模拟机的宿主机 10.0.2.2:8080）
flutter run

# 4. 指定后端地址
flutter run --dart-define=API_BASE_URL=http://192.168.1.10:8080/api/v1

# 5. 运行测试
flutter test

# 6. 静态分析
flutter analyze
```

## 与 PC 端的对齐

- 路由对齐 `frontend/apps/learner/src/router/routes.ts`：home / timeline / search / detail / login / register。
- API 端点与字段对齐 `frontend/packages/api/src/modules/history-types.ts`（snake_case 字段）。
- 统一响应包络对齐 `frontend/packages/api/src/types.ts`（`ApiEnvelope`、`ListResponse`）。

## 当前状态

本目录为 P6 阶段 MVP 工程骨架：页面以占位 UI + 注释形式呈现，数据层（repository / api client / provider）已搭好结构并返回占位数据，待后续接入真实后端联调。
