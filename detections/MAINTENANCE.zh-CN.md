# 防御检测规则：中文维护提示

这里保留源韩语本地化项目附带的防御检测材料，其用途是识别、分析并缓解自动化攻击行为，不应以此作为对未授权目标主动探测的依据。

## 目录与职责

- `sigma/`：日志检测规则与兼容性说明。
- `suricata/`：网络 IDS 规则和测试样例。
- `attack/`：MITRE ATT&CK 映射与规则覆盖资料。
- `indicators/`：指标及其出处说明。
- `triage/`：被调查主机上的本地分类、核查辅助资料。
- `tests/`：检测规则校验与回归测试，入口参见 [tests/README.md](tests/README.md)。

## 修改原则

1. 每条检测规则都应有可追溯的来源和可复现的样例，区分防御线索与独立定罪证据。
2. 变更规则时同步核对对应测试、误报边界、平台字段映射及 ATT&CK 覆盖关系。
3. 请优先在隔离环境中运行 `detections/tests/run-all.sh`；完整测试需要 Docker 及测试镜像。
4. 不要把生产日志、实际客户环境流量、IP 资产清单、密钥或真实身份数据提交仓库。

仓库的检测规则工作流位于 `.github/workflows/detections.yml`（相对仓库根目录）。上游详细文档继续保留在 [README.md](README.md) 和 [README.ko.md](README.ko.md)。

回到 [中文仓库维护指南](../docs/REPOSITORY_GUIDE.zh-CN.md)。
