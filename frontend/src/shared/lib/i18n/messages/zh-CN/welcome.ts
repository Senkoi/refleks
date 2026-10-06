import type { WelcomeMessages } from "../en/welcome";

/**
 * 欢迎功能的简体中文文本。
 */
export const welcome: WelcomeMessages = {
  content: {
    titleFirstLaunch: "喵，瞄瞄来陪你练啦 · v{version}",
    titleUpgrade: "回来啦，继续一起练喵 · v{version}",
    introFirstLaunch:
      "我是瞄瞄，你的瞄准训练搭子。时间和安排交给我，你专心练就好喵。",
    introUpgrade: "这次我又学会了一些新本领，点开版本记录看看喵。",
    details:
      "先确认游戏目录，再告诉我这次能练多久。想了解具体变化，可以看看下面的版本记录。",
    highlightsTitle: "快速开始",
    highlights: {
      changelog: "确认设置里的 KovaaK’s 游戏目录。",
      docs: "到训练工作台选择时间，生成并安装列表。",
      customize: "在游戏的 Local Playlists 打开本次列表，开始工作台计时。",
      community: "完整对局会自动记录；点击练过的场景名称回看成绩。",
    },
    linksTitle: "资源",
    ctaFirstLaunch: "开始一起练",
    ctaUpgrade: "继续陪我练",
    links: {
      docsLabel: "浏览文档",
      docsDescription: "我把设置步骤和常见问题都放在这里啦。",
      changelogLabel: "阅读更新日志",
      changelogDescription: "看看瞄瞄一路学会了哪些新本领。",
    },
  },
  modal: {
    moreDetails: "更多介绍",
    optionalRecording: "轨迹与录像设置（可选）",
    syncStatusEnabled: "训练同步当前已启用。你可以稍后在隐私设置中更改此项。",
    syncStatusDisabled:
      "训练同步当前已在设置中关闭。如果之后启用，将使用此选择。",
    sectionFirstTime: "首次设置",
    sectionProfile: "个人资料设置",
    sectionReview: "设置",
    sectionFirstTimeDescription:
      "先决定要不要记录轨迹和录像。以后随时能改，按你舒服的方式来喵。",
    sectionProfileDescription:
      "选择训练在 你配置的同步服务 上的显示方式。你可以稍后在隐私设置中更改此项。",
    sectionReviewDescription:
      "查看当前设置。你可以随时在设置面板中更改这些选项。",
    recommended: "推荐",
    later: "稍后",
    private: "私密",
    index: {
      label: "你配置的同步服务",
      description: "只有配置了同步服务并开启上传，我才会把完成的训练发送过去。",
    },
    publicProfile: {
      label: "公开个人资料",
      subtitle: "在 Index 上显示我的 Steam 名称。",
      description: "如果你希望上传的训练显示 Steam 名称，此选项最适合你。",
      bullets: [
        "你的 Steam 名称会显示在上传到 Index 的训练上。",
        "稍后可以在隐私设置中切换为匿名。",
      ],
    },
    anonymous: {
      label: "匿名",
      subtitle: "隐藏身份，分享贡献。",
      description:
        "如果你希望贡献数据，同时避免在上传内容中包含可识别信息，此选项最适合你。",
      bullets: [
        "上传前会清除 Steam ID 和用户名称。",
        "数据将发送到你选择的服务，请先了解它的用途。",
        "稍后可以在隐私设置中切换回公开。",
      ],
    },
    mouseTraces: {
      label: "鼠标轨迹",
      description:
        "我可以记下你练习时的鼠标移动，方便之后回放和比较。你也可以随时关掉。",
      helper: "这只是你的初始选择——稍后可以在常规设置中更改。",
      enabled: {
        label: "启用鼠标轨迹",
        subtitle: "记录受支持训练中的移动。",
        description: "想从第一局就能回看操作，让我帮你记下来喵。",
        bullets: [
          "记录会使用少量资源；如果影响流畅度，可以随时关闭。",
          "可以在历史记录视图中回放和比较训练。",
          "随时可以在常规设置中关闭。",
        ],
      },
      disabled: {
        label: "暂时不要",
        subtitle: "先不记录轨迹，随时可以启用。",
        description: "先轻松练几局，熟悉我以后再决定也行喵。",
        bullets: [
          "让首次设置保持简单。",
          "稍后随时可以在常规设置中启用轨迹。",
          "应用的其他功能不受影响。",
        ],
      },
    },
    screenReplay: {
      label: "屏幕回放",
      description: "我可以录下你的练习，之后一起看看准星移动和操作细节。",
      helper: "需要 FFmpeg。稍后可以在常规设置中更改。",
      enabled: {
        label: "启用回放录制",
        subtitle: "录制训练过程中的屏幕（硬件加速）。",
        description: "想把分数、轨迹和画面一起复盘，可以让我录下来。",
        bullets: [
          "默认以 30 fps 录制；优先使用 GPU 编码，不可用时可能改用软件编码。",
          "每场比赛后，回放会作为运行检查器中的新标签页显示。",
          "随时可以在常规设置中关闭。",
        ],
      },
      disabled: {
        label: "暂不录制",
        subtitle: "先仅使用鼠标跟踪，随时添加屏幕回放。",
        description:
          "这是一个轻量的开始方式。熟悉应用后，仍可稍后启用回放录制。",
        bullets: [
          "让首次设置保持简单。",
          "鼠标轨迹和其他功能仍可正常使用。",
          "随时可以在常规设置中启用屏幕捕获。",
        ],
      },
    },
    resourcesDescription:
      "如果你想完整了解版本历程，更新日志和文档始终只需点击一下即可访问。",
  },
};
