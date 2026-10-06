package training

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) Import(data []byte, src Source) (int, error) {
	items, err := ParsePlaylist(data, src)
	if err != nil {
		return 0, err
	}
	return s.Add(items)
}

func (s *Service) Add(items []Scenario) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.state.Catalog)
	s.state.Catalog = mergeCatalog(s.state.Catalog, items)
	s.mergeCurricula(items)
	return len(s.state.Catalog) - n, s.save()
}

func (s *Service) ImportURL(ctx context.Context, raw string) (int, error) {
	items, c, err := readSource(ctx, raw, raw)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		s.mu.Lock()
		s.state.Discovery.Candidates = append(s.state.Discovery.Candidates, c)
		err = s.save()
		s.mu.Unlock()
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("已保存页面线索；未取得结构化关卡列表，请导入其 playlist JSON")
	}
	return s.Add(items)
}

func (s *Service) Discover(ctx context.Context) (Discovery, error) {
	s.mu.Lock()
	if s.discovering {
		s.mu.Unlock()
		return Discovery{}, fmt.Errorf("关卡发现正在进行")
	}
	s.discovering = true
	focus := s.state.Preferences.Focus
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.discovering = false; s.mu.Unlock() }()
	ctx, cancel := context.WithTimeout(ctx, 110*time.Second)
	defer cancel()
	items, d := discover(ctx, focus)
	s.mu.Lock()
	defer s.mu.Unlock()
	before := len(s.state.Catalog)
	s.state.Catalog = mergeCatalog(s.state.Catalog, items)
	s.mergeCurricula(items)
	d.Imported = len(s.state.Catalog) - before
	s.state.Discovery = d
	return d, s.save()
}

func (s *Service) DiscoveryDue(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.state.Preferences.AutoDiscover || s.discovering {
		return false
	}
	t, err := time.Parse(time.RFC3339, s.state.Discovery.Updated)
	return err != nil || now.Sub(t) > 7*24*time.Hour
}

func (s *Service) UpdateScenario(item Scenario) error {
	if !validSkill(item.Skill) && item.Skill != "unknown" {
		return fmt.Errorf("无效分类")
	}
	if item.Seconds < 10 || item.Seconds > 3600 {
		return fmt.Errorf("关卡时长需为 10–3600 秒")
	}
	if item.Difficulty != "unknown" && item.Difficulty != "novice" && item.Difficulty != "intermediate" && item.Difficulty != "advanced" {
		return fmt.Errorf("无效难度")
	}
	if item.Preference != "" && item.Preference != "liked" && item.Preference != "neutral" && item.Preference != "disliked" {
		return fmt.Errorf("无效喜好反馈")
	}
	if item.PersonalDifficulty != "" && item.PersonalDifficulty != "easy" && item.PersonalDifficulty != "suitable" && item.PersonalDifficulty != "hard" {
		return fmt.Errorf("无效体感难度")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.VariantOf != "" {
		found := false
		for _, s := range s.state.Catalog {
			if strings.EqualFold(s.Name, strings.TrimSpace(item.VariantOf)) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("原版关卡尚不在关卡库中")
		}
	}
	for i := range s.state.Catalog {
		if s.state.Catalog[i].Name == item.Name {
			if item.VariantOf != "" && strings.EqualFold(item.VariantOf, item.Name) {
				return fmt.Errorf("变体不能指向自身")
			}
			for _, name := range item.RelatedBenchmarks {
				if len(name) > 200 {
					return fmt.Errorf("关联 benchmark 名称过长")
				}
			}
			e := &s.state.Catalog[i]
			e.Skill = item.Skill
			e.Family = item.Family
			e.Difficulty = item.Difficulty
			e.DifficultySource = "manual"
			e.Technique = technique(item.Name, item.Skill)
			e.Seconds = item.Seconds
			e.VariantOf = strings.TrimSpace(item.VariantOf)
			e.RelatedBenchmarks = mergeStrings(nil, item.RelatedBenchmarks)
			e.Preference = item.Preference
			e.PersonalDifficulty = item.PersonalDifficulty
			e.Enabled = item.Enabled && validSkill(item.Skill)
			e.Classification = "manual"
			return s.save()
		}
	}
	return fmt.Errorf("关卡不存在")
}
