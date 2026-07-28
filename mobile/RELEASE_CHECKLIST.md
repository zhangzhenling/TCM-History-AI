# 移动端发布 Checklist

本文档记录 TCM-History-AI 移动端（Flutter）的发布流程。

## 发布前检查

### 代码质量
- [ ] `flutter analyze` 无错误
- [ ] `flutter test` 全部通过
- [ ] 代码已合并到 `main` / `develop` 分支
- [ ] CHANGELOG 已更新

### 版本管理
- [ ] 版本号已更新（`pubspec.yaml` 中的 `version` 字段）
- [ ] Android `versionCode` 正确递增
- [ ] iOS `CFBundleVersion` 正确递增
- [ ] 使用 `./scripts/version.sh bump patch/minor/major` 统一管理版本

### 配置检查
- [ ] API 地址指向生产环境
- [ ] 调试开关已关闭（debug 日志、调试菜单等）
- [ ] 第三方服务 key 为生产版本（如统计、崩溃上报）

---

## Android 发布流程

### 1. 签名配置
- [ ] `android/key.properties` 已配置（从 `key.properties.example` 复制）
- [ ] 签名密钥文件（`.keystore` / `.jks`）在正确位置
- [ ] 密钥密码正确，可正常签名

### 2. 构建 APK
```bash
# 构建 prod flavor 的 release APK
./scripts/build.sh android --flavor prod

# 或直接使用 flutter 命令
flutter build apk --release --flavor prod
```

### 3. 构建 App Bundle（推荐，Google Play）
```bash
flutter build appbundle --release --flavor prod
```

### 4. 验证 APK
- [ ] APK 可正常安装
- [ ] 签名信息正确（`apksigner verify --print-certs app-release.apk`）
- [ ] 版本号正确
- [ ] 无调试权限（如 `READ_LOGS` 等）
- [ ] 网络安全配置正确

### 5. 上传
- [ ] 上传到 Google Play Console
- [ ] 填写商店信息（标题、描述、截图、图标）
- [ ] 选择发布渠道（内部测试 → 封闭测试 → 开放测试 → 生产）
- [ ] 配置内容分级
- [ ] 设置定价与分发范围

---

## iOS 发布流程

### 1. 签名配置
- [ ] `ios/Flutter/Signing.xcconfig` 已配置（从 `Signing.xcconfig.template` 复制）
- [ ] Apple Developer 账号有效
- [ ] 已创建 App ID（Bundle ID: `com.tcmhistory.tcmHistoryAi`）
- [ ] 已创建 Distribution Certificate
- [ ] 已创建 App Store Provisioning Profile

### 2. 构建 IPA
```bash
# 构建 IPA（需要签名配置）
./scripts/build.sh ios

# 或无签名构建（仅验证）
./scripts/build.sh ios --no-codesign
```

### 3. 验证
- [ ] Archive 成功
- [ ] 签名验证通过
- [ ] 无 ATS 违规（生产环境禁用 `NSAllowsArbitraryLoads`）
- [ ] 权限声明完整（Info.plist 中 `NS...UsageDescription`）

### 4. 上传 App Store Connect
```bash
# 使用 Xcode Organizer
# 或使用命令行
xcrun altool --upload-app --type ios -f build/ios/ipa/*.ipa \
  --apiKey YOUR_KEY_ID --apiIssuer YOUR_ISSUER_ID
```

### 5. 提交审核
- [ ] App Store Connect 中填写元数据
- [ ] 上传截图（6.5 英寸 / 5.5 英寸 / iPad）
- [ ] 填写应用描述、关键词、支持 URL
- [ ] 配置内容分级
- [ ] 设置定价与可用性
- [ ] 提交 App Review

---

## 版本号规范

采用语义化版本（Semantic Versioning）：`MAJOR.MINOR.PATCH+BUILD_NUMBER`

- **MAJOR**：不兼容的 API 变更 / 重大功能更新
- **MINOR**：向下兼容的功能性新增
- **PATCH**：向下兼容的问题修正
- **BUILD_NUMBER**：构建号，每次发布递增

示例：`1.0.0+1` → `1.0.1+2` → `1.1.0+3` → `2.0.0+10`

---

## 环境配置

| 环境 | Bundle ID (Android) | Bundle ID (iOS) | API 地址 | 用途 |
|------|---------------------|-----------------|----------|------|
| dev | `com.tcmhistory.tcm_history_ai.dev` | `com.tcmhistory.tcmHistoryAi.dev` | 开发服务器 | 日常开发调试 |
| staging | `com.tcmhistory.tcm_history_ai.staging` | `com.tcmhistory.tcmHistoryAi.staging` | 预发布服务器 | 发布前验证 |
| prod | `com.tcmhistory.tcm_history_ai` | `com.tcmhistory.tcmHistoryAi` | 生产服务器 | 正式发布 |

---

## CI/CD 自动化

### GitHub Actions
CI 配置位于 `.github/workflows/mobile-ci.yml`，包含：

- **test job**：静态分析 + 单元测试（必须通过）
- **build-android**：Android APK 构建 + 上传 artifact
- **build-ios**：iOS 构建 + 上传 artifact（无签名）

### 手动构建
```bash
# 进入移动端目录
cd mobile

# 安装依赖
flutter pub get

# 运行测试
flutter test

# 构建 Android APK
./scripts/build.sh android --flavor prod

# 构建 iOS
./scripts/build.sh ios
```

---

## 紧急发布 / 热修复

1. 从主分支创建 hotfix 分支：`hotfix/1.0.1`
2. 修复问题并提交
3. 升级补丁版本号
4. 构建验证
5. 合并回主分支并打 tag
6. 发布到应用商店（Android 可快速发布，iOS 需走审核）

---

## 发布后验证

- [ ] 应用商店状态正常（可搜索 / 可下载）
- [ ] 下载安装验证
- [ ] 首次启动正常
- [ ] 核心功能验证（登录、浏览、搜索等）
- [ ] 崩溃率监控
- [ ] 用户反馈收集
