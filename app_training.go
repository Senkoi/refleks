package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	ctx, cancel := context.WithCancel(a.ctx)
	a.trainingCancel = cancel
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
	if name != "" {
		if err = a.LaunchKovaaksScenario(name, "challenge"); err != nil {
			a.trainingSvc.LaunchFailed(err)
			return err
		}
	}
	return nil
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
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "导出训练列表（每图一次；动态阈值由本应用管理）", DefaultFilename: "Refleks-Adaptive.json", Filters: []runtime.FileFilter{{DisplayName: "KovaaK's playlist", Pattern: "*.json"}}})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, b, 0600)
}
