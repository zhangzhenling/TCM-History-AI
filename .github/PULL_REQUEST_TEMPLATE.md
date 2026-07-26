## 变更目的

<!-- 简述本次 PR 解决的问题或实现的功能，1–3 句话。 -->

## 影响范围

- 涉及模块：<!-- 例如 backend/history-service、frontend/web、deploy/docker -->
- 涉及接口：<!-- 列出新增/修改/删除的 API，无则填「无」 -->
- 数据库变更：<!-- 是否含 migration，无则填「无」 -->
- 兼容性影响：<!-- 是否破坏向后兼容，如无填「无」 -->

## 关联 Issue

<!-- 关联的 issue 编号，例如 Closes #123 / Refs #456 -->

## 自测清单

请在提交前确认以下事项全部完成：

- [ ] 本地 `make backend-test` / `make frontend-build` 通过
- [ ] CI 流水线全绿（lint / test / build）
- [ ] 至少一名 Reviewer 已批准
- [ ] 新增/修改代码的单元测试覆盖率达标（后端 ≥ 70%，关键路径 ≥ 90%）
- [ ] 已移除调试残留（`fmt.Println` / `console.log` / 注释掉的代码）
- [ ] 已更新相关文档（OpenAPI / README / 设计说明书章节）
- [ ] 如含 migration，已在本地验证 up/down 可逆

## 风险点与回滚方案

<!-- 说明本次变更可能引入的风险，以及出问题时的回滚策略。无风险可填「无」。 -->
