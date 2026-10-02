# 文件解析与特征数据规范

本文件确定数据应如何读取、保留和验证。它不确定家族难度模型、权重或段位边界。

## 一、身份、版本与单位

每份文件记录：原文件名、内部 Name、获取时间、来源 URL/本地来源、原始字节 SHA-256、GameVersion、文件大小、对应的 leaderboard ID 及匹配证据。名称相同仅是候选匹配；已确认匹配才可与分数门槛相连。

benchmark 归属另存 benchmark ID、版本/难度组、原生分类、rank 门槛、门槛快照 hash、采集时间。禁止以 leaderboard ID 单独充当场景内容版本。

每个提取值必须带：`raw_value`、`parsed_value`、`unit`、`source_section`、`profile_name`、`line`、`status`。status 至少区分 explicit / default_verified / missing / invalid / unresolved_reference / unsupported。缺省值不能直接填 0。

本次 15 个文件的提取事实见 `data/structure-probes.json`；字段清单见 `data/field-inventory.json`。它们是格式探查，不是可直接用于模型的特征矩阵。

## 二、结构与活动引用

`.sce` 包含无节名的根属性，以及可重复的 `[Bot Profile]`、`[Character Profile]`、`[Dodge Profile]`、`[Weapon Profile]` 等节。每节通过自身 `Name` 区分；使用普通 INI 解析器覆盖同名节会丢失数据。

解析顺序：

1. 保留原始节顺序、Name、字段位置和重复键；编码/BOM、数字小数点、CRLF/LF、空分号槽位分别处理。
2. 从实际根配置 `PlayerProfile` 和 `AddedBots` 建立引用图；不要把 `BotCharacters` 的候选配置列表当作当前目标数量。
3. `.rot` 进入 Bot Rotation Profile，解析 `ProfileNames`、权重和顺序，再进入具体 Bot Profile。保留循环引用和不完整引用的诊断信息。
4. 解析 bot→character/dodge、character→weapon/ability，以及会影响位置、状态、命中和计分的其他引用。武器槽位空项不可丢掉其索引。
5. 单独识别玩家、可计分目标和环境辅助 bot。`DisableScoring`、队伍、无敌、伤害和计分条件共同核验；辅助 bot 虽不计入可击杀目标数，仍可能通过击退/墙面排斥改变目标轨迹。
6. `[Map Data]` 保留成地图内容，不能当作普通 key=value 字段。解析 spawn、区域、障碍、边界、平台与变换，记录外部地图依赖和不支持的实体。

本轮探查只验证 root→bot/rotation→character/dodge→weapon 的结构链接，15 份文件在这个有限范围内没有未解析引用。**这不表示完整的 ability、aim、spawn、物理图已经解析或实机验证。**

## 三、特征分层

| 数据层 | 要收集的字段或量 | 可直接得到什么 | 尚需什么 |
|---|---|---|---|
| 命中几何 | MainBBType/Radius/Height、ProjBB*、头部配置、武器 Type、HitscanRadius、弹丸半径 | 原始碰撞形状与射击方式 | 活动武器、视角/目标姿态、地图位置、缩放语义核验 |
| 目标生成 | AddedBots、rotation、respawn、spawn offsets、生成约束、分阶段数量 | 根 bot 槽位数、原始生成规则 | 实际同时可见/可命中目标分布；阻挡、随机生成及阶段切换 |
| 移动动力学 | MaxSpeed、Acceleration、Friction、Gravity、jump、dodge timing、profile weights、反应触发 | 配置范围和轨迹机制候选 | 各字段在当前游戏版本的语义；加速/碰撞/外力后的实际轨迹 |
| 武器与消灭 | Type、Category、DamagePerShot、TimeBetweenShots、血量、恢复、无敌、弹匣、reload、命中奖励弹药 | 原始开火及生存规则 | 击杀/命中时间的有效模型；burst/charge/projectile、死亡触发、伤害倍率 |
| 计分与结束 | ScorePer*、各类 accuracy multiplier、ScoreToWin、BotMaxLives、Timelimit、补时 | 计分规则候选与时间上限 | 确认实际结束条件和计分单位；轮换/kill/time 机制 |
| 场景变化 | Timescale、动态尺寸/速度、阶段轮换、表现触发变化 | 是否含动态参数与范围 | 状态依赖轨迹；变化范围不能压成单个固定值 |
| 练习条件 | FOV 锁定/范围/尺度、视角、作者说明、灵敏度要求 | 明确约束及未约束部分 | 玩家实际设置；文本条件可能未写入文件 |

### 命中盒的重要语义

官方 Character Profiles 说明：Main bounding box 参与世界碰撞，可被 hitscan 和 projectile 命中；Projectile bounding box 不参与世界和 hitscan 碰撞，但可被 projectile 命中。因此 **hitscan 不应一律读取 ProjBBRadius**。投射物场景还需确认有效命中区域组合及弹丸碰撞半径。

历史 Controlsphere 文件中目标 MainBBRadius=20、ProjBBRadius=55，玩家武器 Type=Hitscan。这正好说明直接拿一个“radius”字段会得到完全不同结果；当前 benchmark 变体仍需重新核验。

### 几何归一化

世界单位的大小、速度不能跨地图直接比较。后续应提取目标的角宽/角高、角速度、角加速度及转移角度分布。

- 已知平面目标有效宽度 W 和垂直视线距离 d 时，可使用 `2 atan(W / (2d))` 表示该平面几何下的角宽。
- 已知球心距离 d 和球半径 r 时，球的视角直径为 `2 asin(r/d)`（d > r）。不能把这两个公式无条件混用。
- 一般形状应投影有效轮廓/命中盒；柱体、头部、多段形状需保留朝向信息。
- 相对位置 r(t) 与相对速度 v(t) 已知时，视线方向角速度大小可由 `|r × v| / |r|²` 得到。`MaxSpeed` 只是配置速度，不能直接代入并称其为实际角速度。
- FOV 与分辨率用于像素/可见性解释；相同世界几何的视角大小不能仅靠改 FOV 当作目标物理难度变化。

这些只是候选特征的物理定义，**没有映射成难度、没有选权重**。本次嵌入地图仍未完成统一坐标和缩放解释，因此所有真实角度特征保持 unknown。

### 时间与击杀机制

`Timelimit` 分开记录为上限；`expected_run_duration` 需要结束条件或实测历史。尤其 Air/Ground Plaza 轮换图不能把 100000 秒传入训练计划。目标生命衰减、固定轮换时长、击杀触发轮换、补时和提前结束需要分别处理。

有效 TTK 不能只用 HP/DPS：攻击间隔、首发时点、命中部位、恢复延迟、invincible、弹匣和击杀补弹均会改变结果。场景和目标阶段要同时保留，不在当前阶段推导未经验证的统一 TTK。

## 四、已发现的格式反例

| 实际样本 | 文件事实 | 需要防止的错误 |
|---|---|---|
| 1wall6targets TE | 7 个 Bot Profile；AddedBots 为 6 次 target.bot；其他配置未通过根路径加入 | 扫描所有 bot 得到错误的数量、尺寸或反应时间 |
| 1w6ts reload v2 | 活动 target 的 MaxSpeed=0，同时 NoDodging=false 且配置了 Mimic | 用“有 Dodge Profile”就判定移动；零速度也不能排除其他外力，需完整检查 |
| 1w6ts reload v2 的缩小、30s 版本 | 源仓库存在真实同源变体 | 将 sibling 同时放进训练集与测试集，导致泛化效果虚高 |
| Air / Ground Plaza | `.rot`、多个 bot 阶段、Timelimit=100000 | 固定 60 秒或直接用上限代表局长；只读第一个 bot |
| SmoothBot Invincible Goated | InvincibleBots 与连续射击配置 | 仍计算“清空目标需要多久” |
| Controlsphere | MainBB 与 ProjBB 尺寸不同 | 忽略武器 Type，读错实际命中盒 |
| 1wall5targets_pasu 与 Reload | 真实文件分别存在，根计分与武器机制需分开核对 | 仅删除 Reload 后缀就认为成绩可以直接比较 |

## 五、无法确认时的结果

不支持的地图、未解引用、缺失字段、动态机制和版本冲突都以显式状态进入数据集。它们不自动继承同名场景难度，也不把名称里的 easy/hard 变成机制标签。对于当前还不支持的类别，输出应保留 unknown。
