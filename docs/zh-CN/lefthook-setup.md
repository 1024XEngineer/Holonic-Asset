# Lefthook 初始化与使用指南

本文档为开发者提供在本项目中配置与初始化 [Lefthook](https://github.com/evilmartians/lefthook) Git Hooks 工具的指南。

---

## 目录

1. [简介](#简介)
2. [前置依赖](#前置依赖)
3. [安装 Lefthook](#安装-lefthook)
4. [初始化 Git Hooks](#初始化-git-hooks)
5. [常用命令与操作](#常用命令与操作)
6. [配置文件说明](#配置文件说明)

---

## 简介

本项目使用 **Lefthook** 管理 Git Hooks（如 `pre-commit`）。在代码提交前，Lefthook 会自动触发代码格式化（Format）与静态检查（Lint），以确保提交到仓库的代码质量符合标准。

---

## 前置依赖

运行预设的 Hook 命令需要本地安装以下工具：

- **Frontend 依赖**：Node.js & `pnpm`（已在 `frontend/` 中配置 `oxlint` 和 `oxfmt`）
- **Backend 依赖**：Go 语言环境 & `golangci-lint`

如未安装 `golangci-lint`，可以通过以下命令安装：

```bash
# macOS (Homebrew)
brew install golangci-lint

# Go install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

---

## 安装 Lefthook

请选择适合您开发环境的安装方式：

### 方式一：Homebrew (macOS / Linux 推荐)

```bash
brew install lefthook
```

### 方式二：pnpm / npm 全局或本地安装

```bash
# 根目录或全局安装
pnpm add -D lefthook
```

### 方式三：Go install

```bash
go install github.com/evilmartians/lefthook@latest
```

### 方式四：Scoop / Winget (Windows)

```bash
# Scoop
scoop install lefthook

# Winget
winget install evilmartians.lefthook
```

安装完成后，可通过以下命令验证是否安装成功：

```bash
lefthook version
```

---

## 初始化 Git Hooks

在克隆仓库或拉取最新代码后，**必须在项目根目录运行以下命令**，将 Hook 脚本写入 `.git/hooks/` 目录：

```bash
lefthook install
```

执行成功后，终端将输出类似以下内容：

```text
Lefthook v1.x.x
SYNC  hooks installed
```

此时，Git Hook 已成功挂载，后续执行 `git commit` 时将自动触发检查。

---

## 常用命令与操作

### 1. 手动测试预提交检查

无需触发实际提交，即可手动运行配置的所有 `pre-commit` 检查：

```bash
lefthook run pre-commit
```

如果只想测试特定的子命令：

```bash
lefthook run pre-commit --commands backend-lint
lefthook run pre-commit --commands frontend-format
```

### 2. 临时跳过 Hooks 检查

在极其特殊或紧急情况下，如果需要跳过检查进行提交：

```bash
# 使用环境变量跳过
LEFTHOOK=0 git commit -m "chore: temporary commit"

# 或使用 Git 自带的 --no-verify 选项
git commit -m "chore: temporary commit" --no-verify
```

### 3. 更新或重新同步 Hooks

当项目根目录下的 `lefthook.yml` 发生变更时，运行以下命令同步生效：

```bash
lefthook install
```

---
