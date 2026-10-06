import { plural } from "../../plural";
import type { SettingsMessages } from "../en/settings";

/**
 * 设置功能的简体中文文本。
 */
export const settings: SettingsMessages = {
  page: {
    title: "设置",
    description: "把瞄瞄调成你喜欢的样子喵。",
    loading: "让我看看你的设置…",
  },
  updates: {
    title: "更新",
    description: "看看我有没有学会新本领，也可以再读一次欢迎页。",
    currentVersion: "当前版本：",
    checkForUpdates: "检查更新",
    readWelcomeAgain: "再次查看欢迎页",
    failedToCheck: "检查更新失败",
    upToDate: "已经是最新版本啦喵！",
    versionAvailable: "版本 {version} 可用",
    installBannerPrefix: "你正在使用 {version}。点击",
    installBannerSuffix: "在后台下载更新——应用将关闭，安装程序会自动启动。",
    downloading: "我正在下载，稍等一下喵…",
    installUpdate: "安装更新",
    viewChangelog: "查看更新日志",
    failedToDownload: "下载更新失败",
  },
  general: {
    recordingAdvanced: "轨迹录制高级设置",
    title: "常规",
    description: "告诉我游戏装在哪里，剩下的记录交给我。",
    kovaaksInstallFolder: "KovaaK's 安装文件夹",
    kovaaksInstallFolderDescription:
      "选好游戏安装目录，我就能找到成绩和表现记录。",
    startWithKovaaks: "随 KovaaK's 启动",
    startWithKovaaksDescription:
      "你打开 KovaaK's 时，我就来陪你练；开启后也会随 Windows 启动。",
    mouseTracking: "鼠标跟踪",
    mouseTrackingDescription:
      "我会记下练习时的鼠标移动，方便回看；目前支持 Windows。",
    bufferDuration: "缓冲时长",
    bufferDurationDescription:
      "我会在内存里暂存这么多分钟的鼠标移动，方便保存完整对局。",
    screenCapture: "屏幕捕获",
    screenCaptureDescription:
      "让我帮你录下练习，之后慢慢复盘喵。需要 Windows 和 FFmpeg。",
    screenCaptureStatusActive: "屏幕捕获已启用",
    screenCaptureStatusError: "屏幕捕获错误",
    screenCaptureStatusUnavailable: "屏幕捕获不可用",
    screenCaptureStatusReady: "屏幕捕获已就绪",
    screenCaptureEncoder: "使用 {encoder}",
    screenCaptureHardware: "（硬件加速）",
    screenCaptureSoftware: "（软件）",
    ffmpegMissingTitle: "未检测到 FFmpeg",
    ffmpegMissingPrefix: "请将",
    ffmpegMissingSuffix: "放在",
    resolution: "分辨率",
    resolutionDescription:
      "我按这个清晰度录制。游戏中改动时，会立即开始新的一段录像。",
    resolutionNative: "原生（显示器分辨率）",
    resolution1080: "1080p (1920×1080)",
    resolution900: "900p (1600×900)",
    resolution720: "720p (1280×720)",
    captureFps: "捕获 FPS",
    captureFpsDescription:
      "我按这个帧率录制。帧率越高越流畅，也会用更多资源喵；游戏中修改会开始新的一段录像。",
    replayCleanup: "回放清理",
    replayCleanupDescription:
      "我会在启动和保存新回放后清理旧录像，按你设定的期限和空间上限来。",
    replayAgeLimit: "回放保留期限",
    replayAgeLimitDescription:
      "超过这段时间的回放，我会自动删除。选“不限”就不按年龄清理。",
    replayAge1d: "1 天",
    replayAge2d: "2 天",
    replayAge4d: "4 天",
    replayAge1w: "1 周",
    replayAge2w: "2 周",
    replayAge1m: "1 个月",
    storageLimit: "存储限制",
    storageLimitDescription:
      "空间超过上限时，我会先删除最旧的回放。选“不限”就不按大小清理。",
    storage1gb: "1 GB",
    storage2gb: "2 GB",
    storage5gb: "5 GB",
    storage10gb: "10 GB",
    storage25gb: "25 GB",
    sessionGap: "训练分段间隔",
    sessionGapDescription:
      "上一局结束后，休息达到这段时间，接下来的对局就记入新的训练时段。",
    sessionGapMinutes: plural({ one: "1 分钟", other: "{count} 分钟" }),
  },
  privacy: {
    localRecords: "成绩、轨迹与录像保存在本机。当前未配置同步服务。",
    title: "隐私",
    description:
      "训练记录默认留在本机。只有你明确配置同步服务并开启上传，我才会发送。",
    runSync: "训练同步",
    serviceUnavailable:
      "训练先安心留在本机喵。瞄瞄尚未配置云端同步服务，这里暂不开放上传。",
    runSyncDescription: "把已完成的训练发送到你明确配置的同步服务。",
    anonymousMode: "匿名模式",
    anonymousModeDescription:
      "在同步上传前，从训练环境数据中移除 Steam ID 和 Steam 用户名。",
  },
  appearance: {
    title: "外观",
    description: "配色、字体和大小，都按你的喜好来喵。",
    theme: "主题",
    themeDescription: "应用的配色主题",
    themeDark: "深色",
    themeLight: "浅色",
    themeCustom: "自定义",
    themeCustomDescription:
      "打开主题文件，就能给我换配色和字体。改好后重启，我就穿上新衣服啦。",
    openThemeFile: "打开主题文件",
    regenerateThemeFile: "重新生成",
    themeFileRegenerateConfirm:
      "要换回默认主题文件吗？这样会覆盖你自己写的主题，确认好再动爪喵。",
    themeFileWriteFailed:
      "我暂时没能保存主题文件喵，检查一下文件夹权限再试试。",
    themeFileOpenFailed: "我暂时没能打开主题文件喵。",
    font: "字体",
    fontDescription: "界面使用的字体系列",
    scale: "缩放",
    scaleDescription: "界面大小；较小的值可在大屏幕上容纳更多内容",
    language: "语言",
    languageDescription: "应用的界面语言",
  },
  advanced: {
    title: "高级",
    description: "集成和数据保留选项。",
    show: "显示高级设置",
    hide: "隐藏高级设置",
    steam: "Steam",
    steamInstallDirectory: "Steam 安装目录",
    steamId: "Steam ID",
    steamIdDescription: "从已登录的 Steam 账户自动检测。",
    personaName: "用户名称",
    personaNameDescription: "请输入你在 kovaaks.com 账户中使用的用户名。",
    displayNamePlaceholder: "显示名称",
    dataRetention: "历史加载范围",
    recentRunsWindow: "最近训练时间范围（天）",
    recentRunsWindowDescription:
      "控制历史页初次加载的记录范围，不删除历史。训练评估使用独立的近期窗口。",
    recentRunsMinCount: "最近训练最少数量",
    recentRunsMinCountDescription:
      "近期记录太少时，我会再找一些较早的训练，凑够这个数量。",
  },
  about: {
    title: "关于瞄瞄",
    description:
      "我是基于 Refleks 成长起来的训练搭子，由 Senkoi 独立维护。感谢原作者和训练社区喵。",
    notices: "来源、致谢与 GPL-3.0 许可证",
  },
  footer: {
    maintenance: "应用维护",
    clearCache: "清除缓存",
    saving: "我正在记下你的设置…",
    unsavedChanges: "有未保存的更改",
    allSaved: "都记好啦喵",
    quitApp: "退出应用",
  },
  errors: {
    failedToSaveSettings: "保存设置失败",
    failedToUpdateAutostart: "更新开机自启失败：{message}",
  },
  clearCache: {
    title: "清除缓存",
    description:
      "我会清掉临时统计和排名缓存，需要时再计算。你的设置和训练记录都会保留喵。",
    clearing: "清除中...",
  },
  resetSettings: {
    title: "重置设置",
    description: "选择要重置为默认值的数据：",
    settingsAndConfig: "设置和配置",
    favoriteScenarios: "收藏的场景",
    scenarioNotes: "场景备注",
    sessionNotes: "训练时段备注",
    resetting: "重置中...",
    resetSelected: "重置所选项",
  },
};
