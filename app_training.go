package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	appsettings "refleks/internal/settings"
	"refleks/internal/training"
)

//go:embed ue4ss-mod/RefleksBridge/Scripts/main.lua
var refleksBridgeLua string

// InstallTrainingBridge installs only our Lua observer into an existing UE4SS
// installation. It never deploys an injector or changes game executables.
func (a *App) InstallTrainingBridge() (string, error) {
	if err := a.trainingReady(); err != nil { return "", err }
	base := a.settingsSvc.Get().KovaaksInstallDir
	if base == "" { return "", fmt.Errorf("请先在设置中指定 KovaaK's 安装目录") }
	bin := filepath.Join(base, "FPSAimTrainer", "Binaries", "Win64")
	if _, err := os.Stat(filepath.Join(bin, "UE4SS.dll")); err != nil {
		return "", fmt.Errorf("未找到现有 UE4SS.dll；请先按 UE4SS 官方说明安装，并确认 KovaaK's 的 Binaries/Win64 路径")
	}
	mods := filepath.Join(bin, "Mods")
	list := filepath.Join(mods, "mods.txt")
	data, err := os.ReadFile(list)
	if err != nil { return "", fmt.Errorf("未找到 UE4SS Mods/mods.txt：%w", err) }
	script := filepath.Join(mods, "RefleksBridge", "Scripts", "main.lua")
	if current, err := os.ReadFile(script); err == nil && string(current) != refleksBridgeLua {
		return "", fmt.Errorf("已有自定义 RefleksBridge 脚本；请先自行备份后处理")
	} else if err != nil && !os.IsNotExist(err) { return "", err }
	if err := os.MkdirAll(filepath.Dir(script), 0755); err != nil { return "", err }
	if err := os.WriteFile(script, []byte(refleksBridgeLua), 0644); err != nil { return "", err }
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "RefleksBridge") {
			lines[i] = "RefleksBridge : 1"
			found = true
		}
	}
	if !found { lines = append(lines, "RefleksBridge : 1") }
	updated := strings.Join(lines, "\n")
	if !strings.HasSuffix(updated, "\n") { updated += "\n" }
	backup := list + ".refleks.bak"
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		if err := os.WriteFile(backup, data, 0644); err != nil { return "", err }
	}
	if err := os.WriteFile(list, []byte(updated), 0644); err != nil { return "", err }
	return script, nil
}

func (a *App) startTraining() {
	dir, err := appsettings.EnsureConfigDir()
	if err == nil {
		a.trainingSvc, err = training.New(dir)
	}
	if err != nil {
		a.trainingErr = err
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.trainingCancel = cancel
	events := training.NewEventReader(filepath.Join(dir, training.GameEventsFile))
	// Start after the existing file's EOF. A prior game session is not replayed.
	_, _ = events.Read()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if observed, err := events.Read(); err == nil {
					for _, event := range observed {
						_ = a.trainingSvc.ApplyGameEvent(event, now)
					}
				}
				if name := a.trainingSvc.Tick(now, a.GetRecentRuns(0)); name != "" {
					if err := a.LaunchKovaaksScenario(name, "challenge"); err != nil {
						a.trainingSvc.LaunchFailed(err)
					}
				}
				if reminder := a.trainingSvc.TakeReminder(); reminder != "" {
					showTrainingReminder(reminder)
				}
				if a.trainingSvc.DiscoveryDue(now) {
					go func() { _, _ = a.trainingSvc.Discover(ctx) }()
				}
			}
		}
	}()
}

func (a *App) trainingReady() error {
	if a.trainingErr != nil {
		return a.trainingErr
	}
	if a.trainingSvc == nil {
		return fmt.Errorf("训练模块尚未就绪")
	}
	return nil
}

func (a *App) GetTrainingState() (string, error) {
	if err := a.trainingReady(); err != nil {
		return "", err
	}
	b, err := json.Marshal(a.trainingSvc.Snapshot(a.GetRecentRuns(0)))
	return string(b), err
}

func (a *App) GenerateTrainingPlan(request string) (string, error) {
	if err := a.trainingReady(); err != nil {
		return "", err
	}
	var p training.Preferences
	if err := json.Unmarshal([]byte(request), &p); err != nil {
		return "", err
	}
	plan, err := a.trainingSvc.Generate(p, a.GetRecentRuns(0))
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(plan)
	return string(b), err
}

func (a *App) TrainingAction(action string) error {
	if err := a.trainingReady(); err != nil {
		return err
	}
	name, err := a.trainingSvc.Action(action, time.Now(), a.GetRecentRuns(0))
	if err != nil {
		return err
	}
	if reminder := a.trainingSvc.TakeReminder(); reminder != "" {
		showTrainingReminder(reminder)
	}
	if name != "" {
		if err = a.LaunchKovaaksScenario(name, "challenge"); err != nil {
			a.trainingSvc.LaunchFailed(err)
			return err
		}
	}
	return nil
}

// TestTrainingReminder allows a real in-game check of borderless overlay visibility.
func (a *App) TestTrainingReminder() {
	go func() {
		time.Sleep(10 * time.Second)
		showTrainingReminder("Refleks 测试提醒：如果游戏中能看到这条消息，浮层已正常显示。")
	}()
}

func (a *App) DiscoverTrainingContent() (string, error) {
	if err := a.trainingReady(); err != nil {
		return "", err
	}
	d, err := a.trainingSvc.Discover(a.ctx)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(d)
	return string(b), err
}

func (a *App) ImportTrainingSource(source string) (int, error) {
	if err := a.trainingReady(); err != nil {
		return 0, err
	}
	return a.trainingSvc.ImportURL(a.ctx, source)
}

func (a *App) ImportTrainingPlaylist() (int, error) {
	if err := a.trainingReady(); err != nil {
		return 0, err
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "导入 KovaaK's 训练列表", Filters: []runtime.FileFilter{{DisplayName: "KovaaK's playlist", Pattern: "*.json;*.plo"}}})
	if err != nil || path == "" {
		return 0, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if info.Size() > 2<<20 {
		return 0, fmt.Errorf("列表文件超过 2 MB")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return a.trainingSvc.Import(b, training.Source{URL: "local-import", Title: "本地列表"})
}

func (a *App) UpdateTrainingScenario(payload string) error {
	if err := a.trainingReady(); err != nil {
		return err
	}
	var s training.Scenario
	if err := json.Unmarshal([]byte(payload), &s); err != nil {
		return err
	}
	return a.trainingSvc.UpdateScenario(s)
}

func (a *App) ImportTrainingBenchmarks() (int, error) {
	if err := a.trainingReady(); err != nil {
		return 0, err
	}
	catalog, err := a.GetBenchmarks()
	if err != nil {
		return 0, err
	}
	progress, err := a.GetAllBenchmarkProgresses()
	if err != nil {
		return 0, err
	}
	items := []training.Scenario{}
	for _, b := range catalog {
		for _, d := range b.Difficulties {
			p, ok := progress[d.KovaaksBenchmarkID]
			if !ok {
				continue
			}
			for _, c := range p.Categories {
				for _, g := range c.Groups {
					for _, s := range g.Scenarios {
						skill := "unknown"
						hint := strings.ToLower(c.Name + " " + g.Name)
						switch {
						case strings.Contains(hint, "switch"):
							skill = "switching"
						case strings.Contains(hint, "static"):
							skill = "static"
						case strings.Contains(hint, "dynamic"), strings.Contains(hint, "linear"), strings.Contains(hint, "timing"):
							skill = "dynamic"
						case strings.Contains(hint, "reactiv"), strings.Contains(hint, "control"):
							skill = "reactive"
						case strings.Contains(hint, "smooth"), strings.Contains(hint, "precis"), strings.Contains(hint, "stability"):
							skill = "smooth"
						}
						diff := "unknown"
						for _, v := range []string{"novice", "intermediate", "advanced"} {
							if strings.Contains(strings.ToLower(d.DifficultyName), v) {
								diff = v
							}
						}
						items = append(items, training.Scenario{Name: s.Name, Skill: skill, Family: strings.ToLower(s.Name), Difficulty: diff, Seconds: 60, Benchmarks: []training.BenchmarkMembership{{Name: b.BenchmarkName + " / " + d.DifficultyName, Thresholds: s.Thresholds}}, Classification: "benchmark", Enabled: skill != "unknown", Sources: []training.Source{{URL: b.SpreadsheetURL, Title: b.BenchmarkName + " / " + g.Name, Retrieved: time.Now().UTC().Format(time.RFC3339)}}})
					}
				}
			}
		}
	}
	if len(items) == 0 {
		return 0, fmt.Errorf("尚无可用 benchmark 关卡定义。请先在 Benchmarks 页面完成同步")
	}
	return a.trainingSvc.Add(items)
}

func (a *App) ExportTrainingPlaylist() (string, error) {
	if err := a.trainingReady(); err != nil {
		return "", err
	}
	b, err := a.trainingSvc.Export()
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "导出训练列表（次数按模块时长生成）", DefaultFilename: "Refleks-Adaptive.json", Filters: []runtime.FileFilter{{DisplayName: "KovaaK's playlist", Pattern: "*.json"}}})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, b, 0600)
}

// InstallTrainingPlaylist places the generated local playlist in KovaaK's
// SaveGames directory. The game loads it as a Local Playlist on next launch.
func (a *App) InstallTrainingPlaylist() (string, error) {
	if err := a.trainingReady(); err != nil {
		return "", err
	}
	base := a.settingsSvc.Get().KovaaksInstallDir
	gameDir := filepath.Join(base, "FPSAimTrainer")
	if base == "" {
		return "", fmt.Errorf("请先在设置中指定 KovaaK's 安装目录")
	}
	if info, err := os.Stat(gameDir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("未找到 KovaaK's 游戏目录：%s", gameDir)
	}
	b, err := a.trainingSvc.Export()
	if err != nil {
		return "", err
	}
	var playlist struct {
		Name string `json:"playlistName"`
	}
	if err = json.Unmarshal(b, &playlist); err != nil || !strings.HasPrefix(playlist.Name, "Refleks Adaptive ") {
		return "", fmt.Errorf("生成的列表无效")
	}
	id := strings.TrimPrefix(playlist.Name, "Refleks Adaptive ")
	if id == "" || strings.ContainsAny(id, "\\/.:\x00") {
		return "", fmt.Errorf("列表标识无效")
	}
	dir := filepath.Join(gameDir, "Saved", "SaveGames", "Playlists")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "Refleks-Adaptive-"+id+".json")
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return "", err
	}
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return path, nil
}
