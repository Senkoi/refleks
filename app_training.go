package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"refleks/internal/models"
	appsettings "refleks/internal/settings"
	"refleks/internal/training"
)

func (a *App) startTraining() {
	dir, err := appsettings.EnsureConfigDir()
	if err == nil {
		a.trainingSvc, err = training.New(dir)
	}
	if err != nil {
		a.trainingErr = err
		return
	}
	if err = a.trainingSvc.InitializeTraining(); err != nil {
		a.trainingErr = err
		return
	}
	a.pollTrainingLocal(time.Now())
	ctx, cancel := context.WithCancel(a.ctx)
	a.trainingCancel = cancel
	go func() {
		if catalog, e := a.GetBenchmarks(); e == nil {
			progress := map[int]models.BenchmarkProgress{}
			for _, b := range catalog {
				for _, d := range b.Difficulties {
					if p, ok := a.benchmarkSvc.GetCachedBenchmarkProgress(d.KovaaksBenchmarkID); ok {
						progress[d.KovaaksBenchmarkID] = p
					}
				}
			}
			_, _ = a.trainingSvc.Add(training.BenchmarkScenarios(catalog, progress, time.Now()))
		}
		localTicker := time.NewTicker(15 * time.Second)
		defer localTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-localTicker.C:
				a.pollTrainingLocal(now)
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
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
	a.pollTrainingLocal(time.Now())
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
	return a.trainingSvc.Import(b, training.Source{URL: "local-import", Title: filepath.Base(path)})
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
	// Refresh definitions before an explicit training sync; cached data remains
	// usable offline. The catalog service uses ETag for unchanged responses.
	_ = a.benchmarkSvc.SyncBenchmarksCache()
	catalog, err := a.GetBenchmarks()
	if err != nil {
		return 0, err
	}
	progress, err := a.GetAllBenchmarkProgresses()
	if err != nil {
		return 0, err
	}
	items := training.BenchmarkScenarios(catalog, progress, time.Now())
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
	dir := filepath.Join(gameDir, "Saved", "SaveGames", "Playlists")
	return a.trainingSvc.Install(dir)
}

func (a *App) pollTrainingLocal(now time.Time) {
	settings := a.settingsSvc.Get()
	base := settings.KovaaksInstallDir
	playlists := ""
	if base != "" {
		playlists = filepath.Join(base, "FPSAimTrainer", "Saved", "SaveGames", "Playlists")
	}
	a.trainingSvc.PollLocal(training.LocalRoots(base, settings.SteamInstallDir), playlists, now)
}

// RefreshTrainingLocal reports real scan results; the poller retains its quiet
// period instead of pretending a partial Steam write is a finished download.
func (a *App) RefreshTrainingLocal() error {
	if err := a.trainingReady(); err != nil {
		return err
	}
	a.pollTrainingLocal(time.Now())
	return nil
}
