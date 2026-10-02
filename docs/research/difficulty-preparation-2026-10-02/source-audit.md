# 来源、版本与证据审查

本轮证据分开记录，不按一个虚构的统一可靠性分数相加。

## 1. 官方定义和作者意图

- [VT S5 作者发布说明](https://blog.voltaic.gg/announcing-the-voltaic-season-5-aiming-benchmarks-beta-for-kovaaks/)：九个能力子类和场景设计解释。文章是 S5 初始说明，不能代表后来门槛不变。
- [VT 版本目录](https://evxl.app/groups/voltaic)：S5 与 S5.5 分开列出。
- [Revosect S5](https://revosect.com/benchmarks)：三组难度、六个子类、每类四图取最高两项的原生计算方式；不能直接套用 VT 的总段位。
- [Viscose S2 作者发布说明镜像](https://w.twstalker.com/ViscoseOCE/status/2048841432279191783)：Easier 低于 VT Iron～Platinum；Medium Gold～GM；Hard Jade～Astra+；Expert GM～Celestial+。这是发布时设计范围，存在交叠且当时仍需试玩调整；原始 X 直连未成功，镜像证据保留这一限制。

这些说明用于采样方向和弱区间，不能直接作为某张文件的机制难度标签。

## 2. 当前 benchmark 门槛：直接读取官方公开接口

已成功访问的只读端点：

```text
GET https://kovaaks.com/webapp-backend/benchmarks/player-progress-rank-benchmark?benchmarkId={id}&steamId={publicSteamId}
GET https://kovaaks.com/webapp-backend/scenario/popular?scenarioNameSearch={name}&page=0&max=25
GET https://kovaaks.com/webapp-backend/leaderboard/scores/global?leaderboardId={id}&page=0&max=100
```

这些是本次实际可读的网页后端接口，未找到稳定性承诺或完整公开 API 合同，不把它们称为长期稳定的官方开发者 SDK。请求失败必须记录，不能用零值替代。

benchmark 请求需要格式有效的公开 Steam ID。本轮使用公开成绩页面已展示的 ID 取得定义，仅在交付快照保留场景、门槛、分类和来源，不保留该玩家成绩。`steamId=0` 返回 400，不能当匿名空用户用。

| 体系 | benchmark ID |
|---|---|
| VT S5 Novice / Intermediate / Advanced | 459 / 458 / 460 |
| VT S5.5 Advanced | 2070 |
| Viscose S2 Easier / Medium / Hard / Expert | 2335 / 2336 / 2337 / 2338 |
| rA S5 Entry / Intermediate / Advanced | 823 / 822 / 821 |

每个快照保留实际采集时间、原始响应 SHA-256、原生 rank 列表和逐场景 `rank_maxes`。响应中的个人 score 与 rank_maxes 存在单位差异风险；本轮完全不使用该响应的个人 score 做标定。原始响应包含公开玩家数据，未纳入交付；保存的是提取后的基准定义。

目录中的分享码来自项目已有的 registry，仅用于采集线索，未启动游戏验证。Viscose S2 的 bundled spreadsheet URL 与其他项目条目出现重复，未将它当作已验证作者表格。本轮的具体门槛以直接 API 结果为证据。

## 3. 官方元数据也需要审查

81 张首轮图按“精确名称 + leaderboard ID”双匹配。原始标签保存，解释层与原始层分离：

- `ww4t Voltaic`、`1w6ts reload v2`、`1w2ts reload smallflicks slightly larger` 的平台类型是 Clicking，虽然位于 Viscose Flick Tech。
- `StaticSwitchingVox Small` 和 `aimerz+ Static Switching 5 Bot Slightly Larger` 是 Target Switching，不能与上面的单击任务合并为相同武器机制。
- `beanTS` 的 aimType 为空；`waldoTS Intermediate` 为 Tracking；`domiSwitch Easy` 为 Clicking；`Popcorn MV Medium` 为 Tracking，但它在 Viscose Click Timing 中。冲突一律进入文件核验队列。
- rA 的 Static 组中，EvoClick 有分阶段目标数量变化，StrawberryClick 有大小/数量变化，1w5ts vbr 有微振动，1w4t Pressure 有深度接近。因此“static”不能简单解释成世界坐标永远不动。

这说明模型需要同时保留 benchmark 的能力分类和由文件提取的实际动作机制，不应靠网站一个标签覆盖全部。

## 4. 社区跨体系配对数据

新发现的 [KovaaksCompare 配对报告](https://github.com/kvn1338/KovaaksCompare/blob/0fce2e25cfe8d24edc365d9488998cba61660d0d/example_reports/viscose-s2-medium-to-volatic-s5-intermediate-paired.md)，固定到 commit `0fce2e25cfe8d24edc365d9488998cba61660d0d`（2026-08-18）。

本包提取其 39 组配对的场景名、排行榜 ID、报告中的重叠人数和相关系数。报告称每对重叠约 7,115～22,627 人，相关系数约 0.641～0.799。**这些是外部报告数值，本轮未取得其原始成绩库、未重算，也未采纳报告的 high confidence 标记。**

主要问题：

1. 场景配对包括 VT ControlTS→1w6ts reload v2、VT DotTS→ww4t 等，存在持续命中任务与单击任务的差别。相关性不能证明机制同类。
2. VT static→Pasu micro 等存在静态与运动目标混配风险。
3. 配置中的 VT S5 Intermediate benchmarkId 为 431，而本轮直接核验的原生 ID 为 458；通过场景排行榜 ID 逐项比较，18 张场景的门槛一致。保留 ID 不一致，不能默默改成已确认同一 benchmark 对象。
4. 其 Viscose Easier 配置中 38 张场景仍是 1～9 占位门槛；不能使用它校准当前 Easier。本轮取得了真实门槛。其 Medium 的 39 张门槛与本次快照一致，这也不能证明文件内容版本相同。
5. 提交时间不是成绩采集时间。报告未提供可核验的逐条时间、场景 hash 及完整版本记录，不能证明两张图反映同一时段水平。

因此只保留为候选证据，当前全部 `calibration_eligible=false`。后续重算应使用同一玩家、相近时段、同类能力，并记录练习次数和采样人群的偏差。

先前社区体感资料亦保留为线索：[旧版段位对照讨论](https://www.reddit.com/r/Voltaic/comments/1nczq3n/viscose_benchmarks/)、[跨体系体验差异](https://www.reddit.com/r/FPSAimTrainer/comments/1q7chmd/vicosevolatic_rank_comparison/)。它们不是当前 S2 的逐场景金标准。

## 5. 真实 .sce 来源与状态

公开存档：

- [fvolpe83/Scenarios](https://github.com/fvolpe83/Scenarios/tree/6a4563d241e61a62020f76796762df5ae8817cc8)：核查递归树 6,015 项；选取 11 个文件作结构探查。
- [MBHuman/Scenarios](https://github.com/MBHuman/Scenarios/tree/1db6bfdec8cc42164ca9ff57dd9d3c82cfaf2137)：递归树 7,107 项；该 commit 日期为 2020-11-21，选取 4 个文件。

15 份文件均按 Git blob SHA-1 检查下载内容，并另记 SHA-256、字段行号和源 URL。未将完整第三方地图随本包重新分发，只提供来源及结构事实。**即使旧存档里有与当前相同的名称，如 1w6ts reload v2，也没有证明它就是当前排行榜对应文件。**

本轮 GitHub 文件搜索和公开仓库检查没有取得并核验 VT S5/S5.5、Viscose S2、rA S5 的当前完整 .sce。场景目录和成绩 API 响应也没有提供可确认的 .sce 下载链接；这不等于证明游戏完全没有其他下载接口。

## 6. 官方文件语义资料

- [场景创建入口](https://wiki.kovaaks.com/en/home/KovaaK%27s/ScenarioCreation/Intro)
- [Character Profiles](https://wiki.kovaaks.com/en/home/KovaaK%27s/ScenarioCreation/CharacterProfiles)
- [Weapon Profiles](https://wiki.kovaaks.com/en/home/KovaaK%27s/ScenarioCreation/WeaponProfiles)
- [Dodge Profiles](https://wiki.kovaaks.com/en/home/KovaaK%27s/ScenarioCreation/DodgeProfiles)
- [Map Creator](https://wiki.kovaaks.com/home/KovaaK%27sMapCreator)

社区 [场景分类指南](https://github.com/SalziCantAim/Documentation-for-Kovaaks-Scenarios) 可辅助查字段，但其中的阈值和名称优先规则不作为本项目难度算法。实际旧文件已表明“存在 dodge profile”本身不足以证明移动。
