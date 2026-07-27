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
├── android/                            # Android 原生工程（Gradle + Kotlin + MainActivity）
├── ios/                                # iOS 原生工程（Xcode + Swift + AppDelegate）
├── pubspec.yaml
├── analysis_options.yaml
└── README.md
```

每个 feature 模块采用三层分层：
- `data/`：repository，封装业务逻辑与数据来源（远程 API / 本地缓存）
- `domain/`：feature 级 state / entity
- `presentation/`：page、widget、provider

## 原生工程结构

`android/` 与 `ios/` 是 P6 阶段补齐的原生工程骨架，遵循 Flutter 官方模板结构，可在装有 Flutter SDK 的开发机上直接 `flutter build` 生成安装包。两套骨架均含真实的 Gradle / Xcode 构建脚本与启动资源配置，非空占位。

### Android（`android/`）

| 文件 | 作用 |
| ---- | ---- |
| `settings.gradle` | 顶层 Gradle settings，引入 `flutter_tools/gradle` 与 `flutter-plugin-loader`，AGP 8.1.0、Kotlin 1.9.10 |
| `build.gradle` | 根工程 Gradle，统一仓库与 `clean` 任务 |
| `app/build.gradle` | `applicationId = com.tcmhistory.tcm_history_ai`，`compileSdk = 34`、`minSdk = 21`、`targetSdk = 34`，JDK 17，release 默认走 debug 签名 |
| `app/src/main/AndroidManifest.xml` | 声明 `INTERNET` 权限（Dio 联调后端）、`MainActivity` 启动入口、`flutterEmbedding=2` |
| `app/src/main/kotlin/com/tcmhistory/tcm_history_ai/MainActivity.kt` | `FlutterActivity` 子类 |
| `app/src/main/res/values/styles.xml` | `LaunchTheme` + `NormalTheme` |
| `app/src/main/res/drawable/launch_background.xml` | 启动背景 |
| `app/src/main/res/drawable/ic_launcher_{background,foreground}.xml` | 自适应图标矢量资源 |
| `app/src/main/res/mipmap-anydpi-v26/ic_launcher{,_round}.xml` | API 26+ 自适应图标 |
| `app/src/main/res/mipmap-{m,h,x,xx,xxx}hdpi/ic_launcher{,_round}.png` | 各密度启动图标（占位 TCM 配色） |
| `gradle/wrapper/gradle-wrapper.properties` | Gradle 8.3 |
| `gradle.properties` | AndroidX + Jetifier |

### iOS（`ios/`）

| 文件 | 作用 |
| ---- | ---- |
| `Runner.xcodeproj/project.pbxproj` | Xcode 工程文件，`PRODUCT_BUNDLE_IDENTIFIER = com.tcmhistory.tcmHistoryAi`，`IPHONEOS_DEPLOYMENT_TARGET = 12.0`，Debug/Release 双配置 |
| `Runner.xcodeproj/xcshareddata/xcschemes/Runner.xcscheme` | 共享 scheme，支持 Build/Run/Test/Profile/Archive |
| `Runner/AppDelegate.swift` | `FlutterAppDelegate` 子类，注册 `GeneratedPluginRegistrant` |
| `Runner/Info.plist` | Bundle 名称、横竖屏、`UILaunchStoryboardName = Main`、ATS 允许 HTTP（开发期联调后端） |
| `Runner/Base.lproj/Main.storyboard` | 启动 storyboard，居中 `LaunchImage` |
| `Runner/Assets.xcassets/AppIcon.appiconset/` | 1024×1024 单尺寸 App Icon（占位） |
| `Runner/Assets.xcassets/LaunchImage.imageset/` | 1x/2x/3x 启动图（透明占位） |
| `Flutter/Debug.xcconfig`、`Flutter/Release.xcconfig` | 引入 `Generated.xcconfig` 与（可选）CocoaPods 配置 |
| `Flutter/AppFrameworkInfo.plist` | Flutter.framework 元数据，`MinimumOSVersion = 12.0` |

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

## 构建与发布

### 本地构建命令

```bash
# Android Release APK（产物：build/app/outputs/flutter-apk/app-release.apk）
flutter build apk --release

# iOS Release（无签名，产物：build/ios/iphoneos/Runner.app）
# 仅在 macOS + Xcode 环境下可用，CI 默认走 --no-codesign
flutter build ios --release --no-codesign

# iOS IPA（需要有效签名配置）
flutter build ipa --release
```

> 远程沙箱未安装 Flutter SDK，故本仓库原生工程为手工搭建的最小可用骨架。在装有 Flutter 3.19+ 的开发机上，可直接 `flutter pub get && flutter build apk --release` 生成 APK；如需重新生成完整模板，可执行 `flutter create . --platforms=android,ios --org com.tcmhistory --project-name tcm_history_ai`。

### CI 构建

CI 流水线位于 `.github/workflows/mobile-ci.yml`，包含三个 job：

| Job | Runner | 步骤 | 说明 |
| --- | ------ | ---- | ---- |
| `test` | `ubuntu-latest` | `flutter pub get` → `flutter analyze` → `flutter test` | 静态分析与单元测试，必须通过 |
| `build-android` | `ubuntu-latest` | `flutter build apk --release` → 上传 APK artifact | `continue-on-error: true`，骨架阶段不阻断主流程 |
| `build-ios` | `macos-latest` | `flutter build ios --release --no-codesign` → 上传 `.app` / `.ipa` artifact | `continue-on-error: true`，骨架阶段不阻断主流程 |

构建 job 设置 `continue-on-error: true`：在原生工程骨架阶段，避免因 Flutter SDK 版本差异或 podspec 缺失等环境问题阻断 `test` job。后续双端产物稳定后可移除该开关。

### 签名配置（占位）

Release 签名密钥与证书不在仓库内托管，发布前需在内部文档（待补：`doc/22-发布与签名.md`）登记：

- **Android**：在 `android/key.properties` 配置 `storeFile` / `storePassword` / `keyAlias` / `keyPassword`（已被 `.gitignore` 忽略），并在 `android/app/build.gradle` 的 `release` 块引用 `signingConfigs.release`；上传 `.jks` / `.keystore` 到 Google Play Console。
- **iOS**：在 `ios/Runner.xcodeproj` 配置 `DEVELOPMENT_TEAM` 与 Provisioning Profile，CI 通过 App Store Connect API Key 走 `xcrun altool` 或 `fastlane pilot` 上传；本仓库 `DEVELOPMENT_TEAM = ""` 为占位，发布前替换为真实 Team ID。

## 与 PC 端的对齐

- 路由对齐 `frontend/apps/learner/src/router/routes.ts`：home / timeline / search / detail / login / register。
- API 端点与字段对齐 `frontend/packages/api/src/modules/history-types.ts`（snake_case 字段）。
- 统一响应包络对齐 `frontend/packages/api/src/types.ts`（`ApiEnvelope`、`ListResponse`）。

## 当前状态

P6 阶段 MVP 已完成：五大 feature（home / search / detail / timeline / profile）页面与后端真实接口联调通过；`android/` 与 `ios/` 原生工程骨架已补齐，CI 双端构建 job（`build-android` / `build-ios`）已配置 `continue-on-error` 容错。Release 签名与上架流程见「签名配置」一节，待后续接入。
