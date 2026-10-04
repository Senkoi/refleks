import type { ErrorMessages } from "../en/errors";

/**
 * Go 后端返回的用户可见错误信息的简体中文文本。
 */
export const errors: ErrorMessages = {
  replay: {
    processing: "我正在整理这局回放喵…",
    ready: "回放准备好啦喵。",
    noCaptureSession: "这局没有可用的录像喵。",
    noSessionCoverage: "没有捕获会话覆盖本次训练。",
    outsideSession: "本次训练发生在可用捕获会话之外。",
    captureStopped: "回放处理完成前，屏幕捕获已停止。",
    processingFailed: "回放处理失败。",
    segmentsMissing: "捕获片段未覆盖训练时间段。",
    trimTimedOut: "等待捕获片段超时。",
    storageUnavailable: "训练存储尚未初始化。",
    missing: "这局没有录到回放喵，可以去设置里看看录制选项。",
    exportFailed: "导出回放失败。",
  },
  screenCapture: {
    starting: "捕获尚未生成画面。",
    active: "正在捕获并接收画面。",
    uninitialized: "屏幕捕获运行时尚未初始化。",
  },
  update: {
    checkFailed: "我暂时没能查到新版本喵，稍后再试试。",
    downloadFailed: "这次更新没下载成功喵，检查网络后再试试。",
    unsupportedOS: "自动更新目前仅支持 Windows。",
  },
  autostart: {
    updateFailed: "更新开机自启设置失败。",
  },
  benchmark: {
    progressFetchFailed: "我暂时没拿到测试进度喵，稍后再试试。",
  },
  scenario: {
    scoresFetchFailed: "我暂时没拿到这张图的分数喵。检查一下设置中的 KovaaK's 用户名，再试一次。",
  },
};
