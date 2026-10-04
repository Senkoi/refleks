package training

// Reuse the duration/exposure policy in the active VDIM path. The template is
// an order/goal source; its author repetitions are an upper bound, not five
// minutes assigned to every block. No history: at most two complete runs.
func curriculumRepetitions(row CurriculumRow, timing TimingEstimate) int {
	role := row.Role
	if role == "" {
		role = "practice"
	}
	count := plannedRepetitions(timing, role)
	if timing.Source != "history" && count > 2 {
		count = 2
	}
	if row.Count < count {
		count = row.Count
	}
	return max(1, count)
}

func originalRowCount(row CurriculumRow) int {
	if row.SourceCount > 0 {
		return row.SourceCount
	}
	return row.Count
}
