# HeadscaleClient · macOS 统一安装包预览

本次每种芯片只提供一个 PKG，无需再选择独立安装版或 GUI-only 版。

| Mac | 下载 |
| --- | --- |
| Apple Silicon（M 系列） | `headscaleclient-macos-arm64-installer.pkg` |
| Intel | `headscaleclient-macos-amd64-installer.pkg` |

安装器会自动处理：

- 没有 Tailscale 服务：安装并启动自带服务。
- 已有外部服务：提醒并复用，不覆盖、停止或卸载原服务，也不启动第二套。
- 已有 HeadscaleClient 管理的服务：更新，保留账号与配置。
- 同时存在两套服务：中止并提示先处理冲突，不擅自选择或删除其中一套。

安装向导增加中英文说明、安装类型提示及完成说明；安装日志记录实际选择。
客户端设置页显示服务来源，并区分不可达、权限不足、接口不兼容的提示。
统一包仍包含备用内核文件，但复用模式不会将它们注册为服务。

复用意味着共享原 Tailscale 的账号及网络设置，不是第二套独立会话。
请勿同时从两个界面修改网络设置。若需转为独立模式，先自行卸载原服务，
再运行同一个 PKG；可能需要重新登录。升级前请从菜单栏退出 HeadscaleClient。

此发布仅更新 macOS 安装方式，Wails 保持 beta.23，网络内核保持 1.102.2。
Windows 用户不需要为这次 Mac 安装流程变更重新安装。

验证：65 项前端测试、7 项安装提示策略测试、三平台 CI 全部通过。Mac 双架构
通过首次安装、升级、外部服务运行/停止场景、冲突保护、保留原服务进程/文件/
网络偏好、卸载与旧版 PKG 升级检查。构建源提交：`f654741`。

仍为预览：无 Apple Developer ID 签名和公证。Mac 官方 App Store/独立版
网络扩展的实际登录、权限与长期网络使用仍需实机验收。不要全局关闭系统
安全保护。详细验证范围见仓库记录，不将模拟目录检测等同于官方客户端验证。

- [安装说明](https://github.com/bimcc/headscaleclient/blob/main/docs/product/MACOS.md)
- [验证记录](https://github.com/bimcc/headscaleclient/blob/main/docs/verification/MACOS-UNIFIED-INSTALLER-2026-09-20.md)
