<div align="center">

# mdtree

**自托管的服务器端 Markdown 浏览/编辑器。**

一棵*只含 Markdown* 的文件树、带实时预览的编辑器、即时文件名搜索 ——
全部由密码保护,全部打包进单个二进制文件。

[English](README.md) · **简体中文**

[![CI](https://github.com/SinlinLi/mdtree/actions/workflows/ci.yml/badge.svg)](https://github.com/SinlinLi/mdtree/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/SinlinLi/mdtree)](https://goreportcard.com/report/github.com/SinlinLi/mdtree)

</div>

![mdtree 编辑器](docs/screenshot-editor.png)

## 为什么做它

你的笔记、文档、README 散落在服务器各处,想快速编辑它们 —— 又不想
`ssh` + `vim`,也不想把整棵目录树同步到笔记本。mdtree 给你一个网址:一棵
只有 Markdown 的干净文件树、一个带实时预览的正经编辑器,以及对服务器上
每个 `.md` 文件的 `Ctrl-P` 搜索。

## 功能特性

- **只含 Markdown 的文件树** —— 目录用于导航,但只列出 Markdown 文件。
  按需懒加载,即使文件系统庞大也依旧流畅。
- **浏览与编辑** —— 基于 CodeMirror 6 的源码编辑器 + 实时、经过净化的
  预览。支持「仅编辑 / 分屏 / 仅预览」三种视图。
- **带索引的文件名搜索** —— 对每个 Markdown 文件建立内存索引,支持模糊
  匹配,以 `Ctrl`/`Cmd` + `P` 命令面板呈现。
- **完整的文件管理** —— 新建、保存、重命名、删除文件,以及新建目录。
  保存是原子操作(先写临时文件再重命名)。
- **鉴权** —— 密码登录(bcrypt 哈希)、HTTP-only 会话 cookie、登录限流。
  mdtree 能触及整个文件系统,所以密码就是那道关卡。
- **单一二进制** —— React 前端通过 `go:embed` 嵌入。往服务器丢一个文件、
  运行,即可。无任何运行时依赖。
- **可观测** —— 结构化分级日志(控制台 + 文件轮转)、`/healthz` 健康检查、
  `/api/stats` 指标端点。
- **明暗双主题**,并记住你的选择。

## 界面截图

| 文件名搜索(`Ctrl-P`)                   | 浅色主题                                |
| --------------------------------------- | --------------------------------------- |
| ![搜索](docs/screenshot-search.png)     | ![浅色主题](docs/screenshot-light.png)  |

## 快速开始

下载或自行构建二进制(见 [构建](#构建)),然后:

```bash
# 生成密码哈希。
./mdtree hash

# 创建配置文件。
cp config.example.yaml config.yaml
# ……把哈希粘贴到 config.yaml 的 auth.password_hash 下……

# 运行。
./mdtree --config config.yaml
```

然后打开 <http://localhost:8080>。

完全没有配置?`./mdtree` 也能直接跑 —— 它会浏览 `/`、绑定到 localhost,
并在控制台打印一次性随机密码。

常用参数:

```bash
./mdtree --root /srv/docs --port 9000 --log-level debug
```

## 构建

环境要求:**Go 1.25+** 和 **Node.js 20+**。

```bash
git clone https://github.com/SinlinLi/mdtree.git
cd mdtree
./scripts/build.sh        # 先构建前端,再生成 bin/mdtree
```

或使用 `make`:

```bash
make build                # 前端 + 二进制
make test                 # 运行测试套件
make dev                  # 后端 + Vite 开发服务器(热重载)
```

## 配置

mdtree 按优先级从低到高读取配置:内置默认值、YAML 配置文件、`MDTREE_*`
环境变量、命令行参数。每个选项都在
[`config.example.yaml`](config.example.yaml) 和
[`docs/configuration.md`](docs/configuration.md) 中有说明。

## 安全

mdtree 的设计目标就是**编辑服务进程能触及的任意文件** —— 这是工具的用途
所在,也是必须鉴权的原因。在把它暴露到 `localhost` 之外前,请先阅读
[`docs/security.md`](docs/security.md):用专用的最小权限用户运行、能收窄
`root` 就收窄、并放在 HTTPS 反向代理之后。

## 文档

- [架构](docs/architecture.md) —— 各部分如何协作
- [配置](docs/configuration.md) —— 每个选项、环境变量与参数
- [API 参考](docs/api.md) —— HTTP JSON 接口
- [安全模型](docs/security.md) —— 威胁模型与加固

## 贡献

欢迎贡献 —— 见 [`CONTRIBUTING.md`](CONTRIBUTING.md)。

## 许可证

[MIT](LICENSE)
