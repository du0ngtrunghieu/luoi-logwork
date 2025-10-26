package helper

import (
	"fmt"
	"strings"
)

func stringSimilarity(s1, s2 string) float64 {
	s1, s2 = strings.ToLower(strings.TrimSpace(s1)), strings.ToLower(strings.TrimSpace(s2))
	if s1 == s2 {
		return 1.0
	}

	r1, r2 := []rune(s1), []rune(s2)
	l1, l2 := len(r1), len(r2)
	if l1 == 0 || l2 == 0 {
		return 0.0
	}

	matchDistance := max(l1, l2)/2 - 1
	matches1 := make([]bool, l1)
	matches2 := make([]bool, l2)
	matches := 0
	transpositions := 0

	for i := 0; i < l1; i++ {
		start := max(0, i-matchDistance)
		end := min(i+matchDistance+1, l2)
		for j := start; j < end; j++ {
			if matches2[j] || r1[i] != r2[j] {
				continue
			}
			matches1[i] = true
			matches2[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	k := 0
	for i := 0; i < l1; i++ {
		if !matches1[i] {
			continue
		}
		for k < l2 && !matches2[k] {
			k++
		}
		if k < l2 && r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	jaro := (float64(matches)/float64(l1) +
		float64(matches)/float64(l2) +
		(float64(matches-transpositions/2) / float64(matches))) / 3.0

	prefix := 0
	for i := 0; i < min(4, min(l1, l2)); i++ {
		if r1[i] == r2[i] {
			prefix++
		} else {
			break
		}
	}
	return jaro + float64(prefix)*0.1*(1-jaro)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func FormatEstimate(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}

	const (
		secondsPerHour = 3600
		hoursPerDay    = 8
		daysPerWeek    = 5
	)

	totalHours := seconds / secondsPerHour
	weeks := totalHours / (hoursPerDay * daysPerWeek)
	days := (totalHours % (hoursPerDay * daysPerWeek)) / hoursPerDay
	hours := totalHours % hoursPerDay

	// Add minutes for precision
	minutes := (seconds % secondsPerHour) / 60

	result := ""
	if weeks > 0 {
		result += fmt.Sprintf("%dw ", weeks)
	}
	if days > 0 {
		result += fmt.Sprintf("%dd ", days)
	}
	if hours > 0 {
		result += fmt.Sprintf("%dh ", hours)
	}
	if minutes > 0 {
		result += fmt.Sprintf("%dm", minutes)
	}

	return result
}
