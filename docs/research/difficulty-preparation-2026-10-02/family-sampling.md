# 具体场景家族采样表

所有名称来自本次官方 API 快照。下表是**比较对象清单**，不是已经证明同难度的配对。以 VT 九个子类组织阅读，同时保留 rA 和 Viscose 原生类别，不强制改写其语义。

## 首轮中间组：81 张

选择中间组是为了先建立跨体系连接；不能把三个体系的 Intermediate/Medium 标签认定为相同水平。完整名称、原生门槛和排行榜 ID 位于 `data/benchmark-snapshot.json`。

| 待比较能力 | VT S5 Intermediate 基准家族 | 跨体系具体候选 | 必须先解决的问题 |
|---|---|---|---|
| Static clicking | VT 1w3ts Intermediate S5；VT ww5t Intermediate S5 | EvoClick Revosect Int；StrawberryClick Revosect Int；1w5ts vbr Revosect Int；1w4t Pressure Revosect Int；ww4t Voltaic；1w6ts reload v2；1w2ts reload smallflicks slightly larger | 数量随时间变化、大小切换、微振动、向玩家接近分别保留，不能简化成“半径＋目标数”；多个 wall 变体不能当独立家族 |
| Dynamic clicking | VT Pasu Intermediate S5；VT Popcorn Intermediate S5 | Pasu Revosect Int；SpringClick Revosect Int；SnowClick Revosect Int；Pasu Voltaic Reload Easy；VT Bounceshot Viscose Medium；CatClick Medium；Popcorn MV Medium；skyClick Goated Medium | Pasu 与其 micro/TS 变体需审查同源；Popcorn MV 的平台标签为 Tracking，必须核对武器和击杀方式 |
| Linear clicking | VT Frogtagon Intermediate S5；VT Floating Heads Intermediate S5 | Floating Heads Revosect Int；VT Floating Heads Viscose Medium；VoxTS Click Medium；psalmTS Viscose Click | Viscose 将后三者放在 Click Timing/Stability；轨迹线性程度仍需文件确认；Floating Heads 跨体系属于同源候选，不能重复增加独立家族数 |
| Precise tracking | VT PGT Intermediate S5；VT Snake Track Intermediate S5 | FloodTrack Revosect Int；VTP Revosect Int；Tornadosphere Revosect Int；Leapstrafes Revosect Int；SmoothBot Perfected；VT PreciseOrb Intermediate 15% Slower | rA precise 覆盖多种轨迹；Viscose 的 arm/blending 不是等价子类；要分别记录形状、深度变化、轴向要求 |
| Reactive tracking | VT Aether Intermediate S5；VT Ground Intermediate S5 | Plaza Pure Revosect Int；Plaza Smooth Revosect Int；Air Pure Revosect Int；Air Smooth Revosect Int；Ground Plaza Sparky V3；Air Spectral Easy；Plink Palace Easy；Air Pure Medium；Air Voltaic Invincible 4 Medium | 多张 Air/Plaza 图可能共享大量配置；按更大的同源组划分验证集。Plink 的变向节奏还会随过程变化 |
| Control tracking | VT Raw Control Intermediate S5；VT Controlsphere Intermediate S5 | Smoothsphere Viscose；Whisphere Viscose；Controlsphere SuperbAim Viscose；VT Controlsphere Viscose；cloverRawControl Viscose 50cm；Flower 50cm；RawControlSphere；Leapstrafes Control wobin | 控制范围、轨迹方向和灵敏度约束分别保留；50cm 场景带作者练习条件，不能只靠文件几何解释个人适配 |
| Speed switching | VT DotTS Intermediate S5；VT EddieTS Intermediate S5 | AngleSwitch Revosect Int；DotSwitch Revosect Int；NovaSwitch Revosect Int；WaveSwitch Revosect Int；voxTargetSwitch 2；aimerz+ Static Switching 5 Bot Slightly Larger；StaticSwitchingVox Small | DotSwitch 带振动和快速恢复；不能把同处 speed 组理解为同一机制；Flick Tech 中的静态点击排除在 TS 直接配对之外 |
| Evasive switching | VT DriftTS Intermediate S5；VT FlyTS Intermediate S5 | SpringTS Revosect Int；WhisphereTS Revosect Int；PasuTS Revosect Int；SmoothTS Revosect Int；kinTS Voltaic Easy；domiSwitch Easy | rA Evasive 内还有 stability/control 差异；domiSwitch 的平台标签与训练分组不一致，先核武器、血量、恢复机制 |
| Stability switching | VT ControlTS Intermediate S5；VT Penta Bounce Intermediate S5 | SmoothTS Revosect Int；beanTS；B180T Voltaic Easy；waldoTS Intermediate | beanTS 标签缺失；waldoTS 标签为 Tracking；需要验证后再决定归属。不可用纯点击场景直接校准持续命中要求 |

其余首轮目标也保留在 81 张清单中，避免为满足这张表而删掉原生子类。一个场景可进入多个待研究视角，但只能按其身份计数一次。

## 同家族难度梯度

| 基准体系 | 已收集难度组 | 用途与限制 |
|---|---|---|
| VT S5 | Novice / Intermediate / Advanced，共 54 条归属 | 18 个场景位置的三级梯度；1w4ts→1w3ts→1w2ts 目标数也改变，不能只比较尺寸 |
| VT S5.5 | 本次明确取得 Advanced，共 21 条 | 与 S5 单独保存门槛；含 domiClick、Air Angelic、voxTS、AvasiveTS 等新增/替换成员。没有推造未核验的 Novice/Intermediate S5.5 清单 |
| Viscose S2 | Easier / Medium / Hard / Expert，共 156 条 | 39 个位置并不保证 39 条纯粹同家族梯度。例如 Easier 的 PGTI 与 Medium 的 PreciseOrb 不能按行号自动当作同文件变体 |
| rA S5 | Entry / Intermediate / Advanced，共 72 条 | 每组 24 张；只保留原生分级，当前没有可信逐段位 VT 换算 |

## 家族确认规则

候选名称与正式家族分开：数据中的 `family_candidate` 只是人工维护的搜索组织方式。确认家族至少需要当前文件中的活动引用图、地图/生成区域、主要轨迹配置和武器/计分规则相互印证，必要时加入作者的衍生说明。

例如 `1w6ts reload v2`、其 30s 变体和缩小变体应优先放在同一同源组；两张同名 Controlsphere 也不能因此判定同文件版本。反过来，名字不同但复用同一地图、bot 和 dodge 参数的场景，应共享验证分组。

已准备 75 个候选家族名和 46 个更保守的同源分组。它们不是独立样本数量的证明。未解析当前文件之前，家族独立性一律未确认。

## 采样覆盖要求

- 每个拟上线能力子类要同时具有同家族梯度和不同家族的相近表现区间。
- 初始采集目标为每个待校准难度区间至少 5 个不同家族；这是工作量规划目标，不是统计上足够的保证。
- “同一社区难度组”只能先给出弱区间；同难度场景配对还需指定目标分数，以及相近水平玩家在两图上的表现证据。
- 如某类只有 2～3 个独立家族，不用 sibling 变体凑数；继续补数据或暂不发布该类难度推断。
- 先补齐这 81 张当前文件，再扩展 286 张的梯度；没有必要先盲目下载所有网上场景。
