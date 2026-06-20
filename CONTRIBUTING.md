# Contributing to MediaHub

感谢你考虑为 MediaHub 做贡献！🎉

## 行为准则

本项目遵循 [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md)。参与即表示你同意遵守其条款。

## 提 Issue

提交 Issue 前请先搜索现有 Issue，避免重复。Bug Report 请包含：

- 复现步骤
- 预期行为 vs 实际行为
- 环境信息（OS、Docker 版本、NAS 型号）
- 关键日志

## 提 PR

1. Fork 仓库，从 `main` 创建特性分支（`feature/my-thing` 或 `fix/my-bug`）
2. 提交前跑测试：`make test` / `make lint`
3. PR 描述里关联 Issue（`Fixes #123`）
4. 等待 CI 通过 + 维护者 review

## 开发环境

```bash
# 后端
cd services/api
go mod tidy
go run ./cmd/server  # 需要本地 postgres + redis

# 前端
cd services/admin    # 或 web-player
npm install
npm run dev
```

## 代码风格

- **Go**：遵循 `gofmt` + `golangci-lint`（配置见 `.golangci.yml`）
- **TypeScript**：遵循 `eslint` + `prettier`（配置见 `.eslintrc` + `.prettierrc`）
- **提交信息**：用 [Conventional Commits](https://www.conventionalcommits.org/)（`feat:`, `fix:`, `docs:`）

## 项目结构

详见 [README.md](./README.md) 的"项目结构"部分。简要：

- `services/api` - Go 后端（domain → repository → service → handler 分层）
- `services/admin` - CMS 管理后台（Vue 3 + Element Plus）
- `services/web-player` - Web 播放端（Vue 3 + hls.js）
- `services/android-tv` - Android TV 客户端（计划中）
- `services/tvos` - tvOS 客户端（计划中）

## 测试

```bash
# 后端
cd services/api
go test ./...

# 前端
cd services/admin
npm run test
```

## 发布流程

1. 更新 `CHANGELOG.md`
2. 打 tag：`git tag v0.x.0`
3. 推 tag 触发 GitHub Actions release workflow
4. 自动构建 Docker 镜像并推送到 GHCR

## 安全漏洞

请 **不要** 在公开 Issue 里报告安全问题。发邮件到 [security@mediahub.dev]（占位，待维护者确定）。

## 许可证

提交 PR 即表示你同意按 [Apache License 2.0](LICENSE) 授权你的贡献。

---

再次感谢你的贡献 ❤️
