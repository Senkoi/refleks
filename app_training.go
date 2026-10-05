package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"aimmeow/internal/constants"
	"aimmeow/internal/models"
	"aimmeow/internal/sceneanalysis"
	appsettings "aimmeow/internal/settings"
	"aimmeow/internal/training"
	"aimmeow/internal/training/orchestration"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) startTraining() {
	dir, err := appsettings.EnsureConfigDir()
	if err == nil {
		var svc *training.Service
		svc, err = training.New(dir)
		if err == nil {
			svc.BeginStartup()
			a.trainingSvc = svc
		}
	}
	if err != nil {
		a.trainingErr = err
		return
	}
	if err = a.trainingSvc.InitializeTraining(); err != nil {
		a.trainingErr = err
		return
	}
	// Keep the existing UI events and mirror subsequent score refreshes into
	// training evidence; never rewrite the already generated playlist here.
	a.benchmarkSvc.SetOnProgressUpdated(func(id int, p models.BenchmarkProgress) {
		runtime.EventsEmit(a.ctx, fmt.Sprintf("%s%d", constants.EventBenchmarkProgressPrefix, id), p)
		runtime.EventsEmit(a.ctx, constants.EventBenchmarkProgressUpdated, map[string]interface{}{"id": id, "progress": p})
		if catalog, e := a.GetBenchmarks(); e == nil {
			if _, e = a.trainingSvc.Add(training.BenchmarkScenarios(catalog, map[int]models.BenchmarkProgress{id: p}, time.Now())); e != nil {
				runtime.LogWarningf(a.ctx, "training benchmark cache update failed: %v", e)
			}
		}
	})
	a.pollTrainingLocal(time.Now())
	a.trainingRunner = orchestration.Start(a.ctx, orchestration.Hooks{
		Startup: func(ctx context.Context) {
			if catalog, e := a.GetBenchmarks(); e == nil {
				a.sampleInitialBenchmarkScores(ctx, catalog)
				if ctx.Err() != nil {
					return
				}
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
			// Catch-up imports existing CSVs asynchronously. Do not infer Novice
			// merely because those historical runs have not arrived yet.
			if !a.runsRuntimeSvc.WaitInitialHistory(ctx) {
				return
			}
			history, historyErr := a.runsRuntimeSvc.TrainingRuns()
			if historyErr != nil {
				a.trainingSvc.StartupFailed(fmt.Errorf("读取训练历史失败：%w", historyErr))
			} else {
				regenerated, prepareErr := a.trainingSvc.PrepareStartup(history)
				if prepareErr != nil {
					a.trainingSvc.StartupFailed(fmt.Errorf("启动训练列表生成失败：%w", prepareErr))
				} else if regenerated {
					// Update only the owned fixed slot; active sessions and foreign
					// playlist files keep the existing ownership protection.
					base := a.settingsSvc.Get().KovaaksInstallDir
					if base != "" {
						if _, installErr := a.InstallTrainingPlaylist(); installErr != nil {
							a.trainingSvc.StartupFailed(fmt.Errorf("启动列表已生成，但安装失败：%w", installErr))
						}
					}
					a.trainingSvc.FinishStartup(nil)
				}
			}
		},
		PollLocal: a.pollTrainingLocal,
		Tick: func(now time.Time) {
			if name := a.trainingSvc.Tick(now, a.trainingExecutionHistory(now)); name != "" {
				if err := a.LaunchKovaaksScenario(name, "challenge"); err != nil {
					a.trainingSvc.LaunchFailed(err)
				}
			}
			if reminder := a.trainingSvc.TakeReminder(); reminder != "" {
				showTrainingReminder(reminder)
			}
		},
		DiscoveryDue: a.trainingSvc.DiscoveryDue,
		Discover:     func(ctx context.Context) { _, _ = a.trainingSvc.Discover(ctx) },
	})
}

// Startup samples the relevant live rosters in addition to existing caches.
// Offline/rate-limited endpoints cannot hold initialization indefinitely; the
// existing deduplicated requests may finish as ordinary background refreshes.
func (a *App) sampleInitialBenchmarkScores(ctx context.Context, catalog []models.Benchmark) {
	ids := training.InitialLevelReferenceIDs(catalog)
	var pending sync.WaitGroup
	for _, id := range ids {
		pending.Add(1)
		go func(id int) { defer pending.Done(); _, _, _ = a.benchmarkSvc.GetBenchmarkProgress(id, false) }(id)
	}
	done := make(chan struct{})
	go func() { pending.Wait(); close(done) }()
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
	case <-ctx.Done():
	}
}

func (a *App) trainingHistory() []models.RunRecord {
	rows, err := a.runsRuntimeSvc.TrainingRuns()
	if err != nil {
		return a.GetRecentRuns(0)
	}
	return rows
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
	return a.trainingSvc.WorkbenchJSON(a.trainingHistory())
}

// Typed endpoints are the frontend's source of truth. Legacy JSON endpoints
// remain readable for existing clients during migration.
func (a *App) GetTrainingWorkbench() (*training.WorkbenchDTO, error) {
	if err := a.trainingReady(); err != nil {
		return nil, err
	}
	return a.trainingSvc.Workbench(a.trainingHistory())
}
func (a *App) GetTrainingExecution() (training.LiveState, error) {
	if err := a.trainingReady(); err != nil {
		return training.LiveState{}, err
	}
	return a.trainingSvc.Live(), nil
}
func (a *App) CreateTrainingPlan(request training.GenerateRequest) (*training.Plan, error) {
	if err := a.trainingReady(); err != nil {
		return nil, err
	}
	a.pollTrainingLocal(time.Now())
	return a.trainingSvc.Generate(request.Preferences, a.trainingHistory())
}
func (a *App) CompareTrainingScenes(anchor, candidate string) (training.ScenarioComparison, error) {
	if err := a.trainingReady(); err != nil {
		return training.ScenarioComparison{}, err
	}
	return a.trainingSvc.CompareScenes(anchor, candidate)
}
func (a *App) GetTrainingSceneRequirements(name, hash string) (*sceneanalysis.Descriptor, error) {
	if err := a.trainingReady(); err != nil {
		return nil, err
	}
	return a.trainingSvc.SceneRequirements(name, hash)
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
	plan, err := a.trainingSvc.Generate(p, a.trainingHistory())
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
	name, err := a.trainingSvc.Action(action, time.Now(), a.trainingHistory())
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
		showTrainingReminder("喵，提醒送到啦！游戏中能看到我，就说明浮层正常。")
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

func (a *App) RecordTrainingTrialFeedback(id, feedback string) error {
	if err := a.trainingReady(); err != nil {
		return err
	}
	return a.trainingSvc.TrialFeedback(id, feedback)
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
		return 0, fmt.Errorf("我还没拿到测试关卡喵，先去基准训练页面同步一下。")
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
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "导出瞄瞄训练列表", DefaultFilename: "AimMeow-Training.json", Filters: []runtime.FileFilter{{DisplayName: "KovaaK's playlist", Pattern: "*.json"}}})
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
		return "", fmt.Errorf("先在设置里告诉我 KovaaK's 装在哪里喵。")
	}
	if info, err := os.Stat(gameDir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("未找到 KovaaK's 游戏目录：%s", gameDir)
	}
	dir := filepath.Join(gameDir, "Saved", "SaveGames", "Playlists")
	return a.trainingSvc.Install(dir)
}

func (a *App) pollTrainingLocal(now time.Time) {
	settings := a.settingsSvc.Get()
	a.trainingSvc.SetSessionGap(time.Duration(settings.SessionGapMinutes) * time.Minute)
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

// Poll only changing execution fields; catalog and ability use a separate refresh.
func (a *App) GetTrainingLiveState() (string, error) {
	if err := a.trainingReady(); err != nil {
		return "", err
	}
	return a.trainingSvc.LiveJSON()
}

// Tick needs recent summaries only, not replay/mouse traces or all 45 days.
func (a *App) trainingExecutionHistory(now time.Time) []models.RunRecord {
	rows := a.trainingHistory()
	out := make([]models.RunRecord, 0)
	for _, r := range rows {
		at, err := time.Parse(time.RFC3339, r.Stats.Summary.DatePlayed)
		if err == nil && !at.Before(now.Add(-3*time.Hour)) {
			out = append(out, r)
		}
	}
	return out
}
