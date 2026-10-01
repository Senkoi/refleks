# KovaaK's 游戏事件桥接（试验性）

默认训练无需 UE4SS：生成并运行本地列表、归集完成成绩、按模块和总时长弹出提醒都由 RefleK's 自身完成。游戏未完成的重开不会写出完整成绩，默认流程因此只能按墙钟时间提醒，无法准确辨别每一次重开。

这部分代码用于验证 UE4SS 能否在当前 KovaaK's 版本稳定读取挑战状态。它尚未通过真实 Windows 游戏测试。只有已经使用 UE4SS 且愿意试验更细致的重开提醒时，才需要在训练工作台展开可选入口安装桥接脚本。RefleK's 不会自动安装 UE4SS；可参考 [UE4SS 官方说明](https://docs.ue4ss.com/installation-guide)。安装脚本只写入 `Mods/RefleksBridge/Scripts/main.lua` 和 `Mods/mods.txt`，第一次修改 `mods.txt` 时保留 `mods.txt.refleks.bak`。

脚本读取 `ScenarioManager:IsInChallenge()`、当前场景对象和挑战剩余时间，每秒观察一次；出现挑战开始、重开、结束边界时向 `%USERPROFILE%\.refleks\kovaaks-events.jsonl` 追加一行 JSON。RefleK's 启动时跳过旧事件，每两秒读取新增的完整行。格式示例：

```json
{"type":"challenge_restart","scenario":"Example Scenario","at":1790870400000}
```

支持的类型为 `challenge_start`、`challenge_restart`、`challenge_complete`、`challenge_canceled`、`challenge_quit`。`at` 是 Unix 毫秒。事件只更新局内状态与反复重开的提醒；**完成次数和分数仍以 KovaaK's 的落盘记录为准**。若场景名无法读取，脚本不会猜测场景或输出事件。发现当前游戏版本的内部函数或属性不同，应根据 UE4SS 日志核对后调整脚本，不能把无事件当成没有重开。

验证时先运行 RefleK's 和 KovaaK's，生成列表并开始计时，再练习一局、重开几次、换关。检查工作台「游戏事件」中的关卡名称、重开次数和局间状态；对照最终成绩记录。若安装 AimMod 等其他 UE4SS 模组，先检查其现有运行环境和加载顺序，避免安装多个不同版本的 UE4SS 运行时。脚本目前不提供游戏内绘制或直接切关，视觉提醒仍由现有 Windows 浮层显示。独占全屏以及关卡名称读取需要实际机器验证。
