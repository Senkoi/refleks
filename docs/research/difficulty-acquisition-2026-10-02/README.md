# Benchmark 优先的真实文件采集结果

日期：2026-10-02。接续 `difficulty-preparation-2026-10-02`，基线研究提交 `35360682ea71a2188e84ee482d09949d297a59c5`。

## 已完成与实际阻塞

已经自主下载并校验 **25 份来源文件、24 个不同 SHA-256 内容**，复现了前一轮全部 15 个格式探查样本，另补充历史同名候选。三个公开仓库的固定 commit 树均完整检查。7 个 benchmark 候选名称有历史文件，其中仅 2 个属于原先的 81 张中间组首轮清单。

**当前版本 benchmark 文件确认数仍为 0；拟合准入样本仍为 0；没有拟合权重或难度预测。**同名旧文件不冒充当前文件。下载、解析工具、事实表、身份清单及 VDIM 标签规则已经落地，应用训练逻辑未改。

| 本轮产物 | 实测结果 | 证据文件 |
|---|---:|---|
| 真实 .sce 获取 | 25 份来源，24 个不同内容 | `data/download-manifest.json` |
| 历史文件活动引用提取 | 25 份；有限引用范围内无未解引用 | `data/structure-extractions.json` |
| 旧同名候选 | 7 / 286 | `data/candidate-file-coverage.json` |
| 首个能力组的采集目标 | 30 个文件、32 条原生 benchmark 归属 | `data/first-family-targets.json` |
| Workshop 精确标题 + KovaaK AppID 核验 | 30 / 30 | `data/workshop-candidates.json`、`data/workshop-file-metadata.json` |
| Steam 元数据中可直接下载的 file_url | 0 / 30 | `data/workshop-file-metadata.json` |
| 当前文件核验／拟合 | 0／未进行 | `data/validation-results.json` |
| 官方 VDIM S5 每日入口 | 六天 × 六档 = 36 | `data/vdim-official-playlist-catalog.json` |
| VDIM 第三方列表线索 | 145 条位置记录、143 个唯一名称 | `data/vdim-secondary-scenario-leads.json` |
| Python 边界测试 | 7 项通过 | `tools/test_sce_extract.py` |

第三方完整 .sce 只进入忽略的 `.cache/`，未重分发到 Git 仓库；固定来源、Git blob SHA-1、SHA-256、大小、内部名称与字段位置可供复现。原生门槛与分类继承准备包证据，不把它们改写成跨体系统一段位。

## 独立下载路径的实测

对 VT 1w3ts Intermediate S5（Workshop ID `3365695379`）测试了原版 [SteamRE/DepotDownloader](https://github.com/SteamRE/DepotDownloader)，release `DepotDownloader_3.4.0` 的 Linux x64 自包含发行版，无用户名、无凭据。

命令：

```sh
DepotDownloader -app 824270 -pubfile 3365695379 -dir <isolated-cache> -max-downloads 2
```

匿名连接及登录成功，但返回 `App 824270 (KovaaK's) is not available from this account.`，退出码 1。这个结果只能证明**本次账号与工具路径不能取得该文件**，不能证明所有独立下载方式都不存在。

此外，首个能力组 30 个场景的官方 Steam 文件元数据均成功读取、标题与 `consumer_app_id=824270` 匹配；它们都有内容句柄，但 `file_url` 都为空。句柄不是 HTTP 下载地址，预览图也不是 .sce。Workshop ID 与排行榜 ID 仍是不同身份系统，标题匹配只是关联候选。

公开仓库 commit 日期分别为 2020-09-23、2020-11-21、2026-07-22；第三个仓库本轮没有精确目标名称匹配。仓库页面的最近活动日期不能代替内容版本日期。未取得当前文件之前，不用历史参数加当前分数门槛训练模型。

## 先从哪些 benchmark 文件入手

采样起点改为用户建议的 **benchmark 原生分类 + 同家族跨档梯度 + 跨 benchmark 候选**，避免只收同档中间组。

第一组为 30 个静态点击／Flick Tech 候选：

- VT S5 两个位置 × 三档：6 个文件。S5.5 Advanced 同名文件另存门槛版本，不增加独立文件数。
- Viscose S2 三个点击候选位置 × 四档：12 个文件。没有把整个 Flick Tech 归为点击，因为其中还含 switching。
- rA S5 Static 四个位置 × 三档：12 个文件。动态数量、尺寸变化、微振动、深度接近必须保留，分类不等于世界坐标静止。

完整名称见 `first-family-names.txt`。若先交最小批次，可以先提供以下 6 张；它们只足以开始两个同源组的梯度核验，不足以声称跨家族泛化验证通过：

```text
VT 1w4ts Novice S5
VT 1w3ts Intermediate S5
VT 1w2ts Advanced S5
VT ww5t Novice S5
VT ww5t Intermediate S5
VT ww5t Advanced S5
```

## VDIM：用途与难度标签分开

从作者入口 `https://bit.ly/VDIMkvksS5` 取得的官方文档，正是用户之前提供的 [Google 文档](https://docs.google.com/document/d/1R4IyJqYmprRauaACt6bah7YzOOVuG6GqXlB03clNHeU/edit)。本轮保存文档内容 hash 和事实目录，不重分发作者完整正文。

当前入口可读到 Entry、Novice、Adept、Intermediate、Advanced、Elite 六档；六个每日类别为 Clicking I/II、Tracking I/II、Switching I/II。它们针对推荐玩家水平，混合多个子类，并包含 benchmark 练习。旧资料中的档数和每日分类不能覆盖这个快照。

每条未来的实际列表成员需要分别记录：推荐玩家档、训练子类、位置、重复次数、benchmark 锚点、训练角色及角色证据。

| 角色 | 能否继承列表等级当场景难度 | 如何使用 |
|---|---|---|
| benchmark | 否 | 与精确当前文件、benchmark 版本和目标分数绑定 |
| warmup | 否 | 保留用途；不作为监督难度标签 |
| skill_isolation | 否 | 保留专项技巧，按实际机制分类 |
| overload | 否 | 只在作者明确说明或文件差异核验后保留相对关系 |
| unknown | 否 | 待审查，不猜成热身或同级 |

开头位置、重复少、名称里的 Easy/Hard 都不构成角色或难度的独立证据。也不能假定整份列表按单一难度全局升序，因为中间可能切换能力子类并重启训练递进。

本轮发现 FpsAimForge 的 VDIM bundle，六份 Intermediate 列表共 145 个位置。但它是**另一个引擎的移植**，仅取场景名、顺序、次数作为待核验线索；没有导入其半径、速度、地图等定义来冒充 KovaaK .sce。只有 11 条精确名称能成为本准备包 benchmark 候选，其余不猜测映射。官方列表内容／角色尚未逐条确认；全部 `calibration_eligible=false`。

## 解析范围与发现

解析器支持重复 profile 节、重复字段诊断、空武器槽位、root→bot/rotation→character/dodge→weapon 引用图、循环及缺失引用报告、字段来源行号。Map Data 保持不透明，所有真实角尺寸、角速度、转移角度仍是 unknown。它还没有完整处理 ability、辅助 bot、动态阶段和地图物理，不是一个完成的特征估算器。

实测再次确认：

- 历史 1wall6targets TE 有 7 个 Bot Profile，但根实际只加入 6 个 target 槽位；未激活的 bot 不进入当前提取。
- 历史 1w6ts reload v2 主目标 MaxSpeed 为 0，同时仍有 dodge 配置；不能据此直接判断实际运动。
- 历史 Controlsphere 主目标 MainBBRadius=20、ProjBBRadius=55；武器类型核验前不能随意选一个 radius。

## 当前文件的最少交接

在游戏 Online Scenarios 里按 `first-family-names.txt` 的精确名称下载／刷新，然后提供对应 .sce。优先首个能力组 30 张，不需要先补齐整个 81 张中间组。文件不全也可以先交已有部分。

提供的 `tools/collect_local_sce.ps1` 会从用户指定游戏根目录的 Scenarios 和相邻 Workshop 目录读取文件，只打包精确内部 Name 命中的 .sce，并生成哈希、GameVersion、缺项清单。不改游戏文件，不收集 stats 或账户文件。它尚未在 Windows 实机执行验证，手动打包是等效路径。

```powershell
& .\tools\collect_local_sce.ps1 -GameRoot 'D:\SteamLibrary\steamapps\common\FPSAimTrainer' -DownloadNote 'refreshed in game on 2026-10-02; not manually edited'
```

生成 `sce-first-family.zip` 后上传。附上当前游戏版本、是否刚刷新、是否手动改过；仅文件 mtime 和 GameVersion 不能证明当前服务器内容。

收到文件后可继续：核验版本及身份 → 解析地图与活动机制 → 审查社区弱区间与具体目标分数 → 冻结家族分组／留出验证 → 比较候选模型。不满足准入条件的类别保持 unknown。

## 复现

在仓库根目录运行：

```sh
python docs/research/difficulty-acquisition-2026-10-02/tools/collect_public_sce.py
python docs/research/difficulty-acquisition-2026-10-02/tools/build_sampling_targets.py
python docs/research/difficulty-acquisition-2026-10-02/tools/capture_workshop.py
python docs/research/difficulty-acquisition-2026-10-02/tools/capture_vdim.py
python -m unittest discover -s docs/research/difficulty-acquisition-2026-10-02/tools -p 'test_*.py' -v
python docs/research/difficulty-acquisition-2026-10-02/tools/validate_acquisition.py
```

联网脚本使用有限并发和超时，捕获失败，不把失败返回当零值。VDIM 文档可能更新；内容 hash 变化时需重新审查。后续成功取得当前文件时应保存新快照，不能把这份历史采集结果整体改为已验证当前文件。
