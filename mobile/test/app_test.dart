// App 根 Widget 冒烟测试：验证 TcmHistoryApp 能在 ProviderScope 下挂载并渲染首页。
//
// 通过 override sharedPreferencesProvider 注入内存版 SharedPreferencesFake，
// 避免 main.dart 中的异步初始化依赖。

import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:tcm_history_ai/app.dart';
import 'package:tcm_history_ai/core/network/token_storage.dart';

void main() {
  testWidgets('App 启动后应渲染首页 AppBar', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final prefs = await SharedPreferences.getInstance();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sharedPreferencesProvider.overrideWithValue(prefs),
        ],
        child: const TcmHistoryApp(),
      ),
    );
    // 仅 pump 一帧验证 widget 挂载，不使用 pumpAndSettle（会等待网络请求超时）
    await tester.pump(const Duration(milliseconds: 500));

    expect(find.text('首页'), findsOneWidget);
  });
}
