package training

func scenarioFromLocal(name, path string, m *Mechanics, a *LocalAssessment) Scenario {
	skill := classify(name)
	if m != nil && validSkill(m.DeclaredSkill) {
		skill = m.DeclaredSkill
	}
	return enrichMechanics(Scenario{Name: name, Skill: skill, Technique: technique(name, skill), Family: family(name), Difficulty: "unknown", DifficultySource: "unknown", Seconds: 60, Enabled: validSkill(skill), Classification: "inferred", Mechanics: m, LocalAssessment: a, Sources: []Source{{URL: "local-sce:" + path, Title: "本地 SCE", Retrieved: a.ObservedAt}}})
}
