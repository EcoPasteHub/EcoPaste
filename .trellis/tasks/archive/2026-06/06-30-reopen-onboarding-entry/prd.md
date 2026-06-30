# 重新打开引导入口

## Goal

在偏好设置中新增一个手动 action，让用户可以再次打开首次使用引导窗口，方便回看权限、导入和基础使用说明。

## What I Already Know

- 前端已有 `/onboarding` 路由。
- 命令层已有 `openOnboarding()` 包装，Rust 端已有 `open_onboarding` 命令。
- 偏好页已有 action 控件模式，适合一次性动作。
- 本需求不需要新增持久化设置字段。

## Requirements

- 在偏好设置的界面/控制相关区域新增“重新打开引导” action 项。
- 点击 action 后复用现有 `openOnboarding()` 命令打开 onboarding 窗口。
- 点击不修改 `onboarding.completed`，不重置设置或历史数据。
- 补齐 zh-CN 和 en-US 文案、搜索关键词、图标映射。

## Acceptance Criteria

- [ ] 偏好页显示“重新打开引导” action。
- [ ] 点击后打开引导窗口。
- [ ] 点击不会把引导状态改回未完成。
- [ ] 中英文文案、搜索和图标正常。
- [ ] 前端 lint/typecheck 通过。

## Out of Scope

- 不重做 onboarding 流程。
- 不新增 settings 字段。
- 不改变首次启动自动弹出引导逻辑。
