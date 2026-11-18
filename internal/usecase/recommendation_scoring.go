package usecase

import (
	"math"
	"sort"
	"strings"

	"haji-koding/path-finder/internal/entity"
)

const (
	// schemaVersion mengikuti kontrak ML data-contract v1.2.
	schemaVersion = "1.0"
	// modelVersion wajib memuat segmen ner/encoder/ranker sesuai kontrak ML.
	modelVersion = "ner:lexicon-rule-1.0.0;encoder:none;ranker:rule-1.0.0"

	defaultTopK = 3
	maxTopK     = 8

	// Bobot mengikuti configs/pipeline.yaml pada pathfinder-ml.
	hardSkillWeight    = 0.70
	softSkillWeight    = 0.20
	completenessWeight = 0.10

	// Bobot skor peringkat: skill wajib lebih menentukan daripada opsional.
	requiredScoreWeight = 0.80
	optionalScoreWeight = 0.20
)

// skillGap adalah hasil perbandingan keahlian profil dengan syarat lowongan.
type skillGap struct {
	Matched         []string
	Missing         []string
	OptionalMatched []string
	RequiredTotal   int
	OptionalTotal   int
	HardMatch       float64
}

// computeSkillGap membandingkan skill profil dengan skill wajib dan opsional
// pada satu lowongan. Perbandingan tidak peka huruf besar/kecil.
func computeSkillGap(profileSkills, requiredSkills, optionalSkills []string) skillGap {
	owned := make(map[string]bool, len(profileSkills))
	for _, skill := range profileSkills {
		owned[strings.ToLower(skill)] = true
	}

	gap := skillGap{
		RequiredTotal:   len(requiredSkills),
		OptionalTotal:   len(optionalSkills),
		Matched:         []string{},
		Missing:         []string{},
		OptionalMatched: []string{},
	}

	for _, skill := range requiredSkills {
		if owned[strings.ToLower(skill)] {
			gap.Matched = append(gap.Matched, skill)
			continue
		}
		gap.Missing = append(gap.Missing, skill)
	}

	for _, skill := range optionalSkills {
		if owned[strings.ToLower(skill)] {
			gap.OptionalMatched = append(gap.OptionalMatched, skill)
		}
	}

	if gap.RequiredTotal > 0 {
		gap.HardMatch = float64(len(gap.Matched)) / float64(gap.RequiredTotal)
	}

	return gap
}

// matchingScore menghitung skor kecocokan 0-100.
func matchingScore(gap skillGap) int {
	if gap.RequiredTotal == 0 {
		return 0
	}

	if gap.OptionalTotal == 0 {
		return clampScore(int(math.Round(gap.HardMatch * 100)))
	}

	optionalRatio := float64(len(gap.OptionalMatched)) / float64(gap.OptionalTotal)
	value := requiredScoreWeight*gap.HardMatch + optionalScoreWeight*optionalRatio
	return clampScore(int(math.Round(value * 100)))
}

// readinessScore menghitung kesiapan kerja 0-100 memakai aturan yang sama
// dengan src/scoring/readiness_score.py. Komponen soft skill lowongan belum
// tersedia pada katalog, sehingga bobotnya dialokasikan proporsional.
func readinessScore(gap skillGap, completeness float64) int {
	if gap.RequiredTotal == 0 {
		return 0
	}

	totalWeight := hardSkillWeight + completenessWeight
	value := (hardSkillWeight*gap.HardMatch + completenessWeight*completeness) / totalWeight
	return clampScore(int(math.Round(value * 100)))
}

// profileCompleteness menghitung rasio kelompok profil yang terisi.
func profileCompleteness(profile SkillProfile) float64 {
	groups := [][]string{
		profile.HardSkills,
		profile.SoftSkills,
		profile.Interests,
		// Kelompok experience belum diekstraksi oleh fallback ini.
		nil,
	}

	filled := 0
	for _, group := range groups {
		if len(group) > 0 {
			filled++
		}
	}

	return float64(filled) / float64(len(groups))
}

func clampScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

// buildItem menyusun satu item rekomendasi dari profil dan lowongan.
func buildItem(profile SkillProfile, job entity.Job, completeness float64) entity.RecommendationItem {
	gap := computeSkillGap(profile.HardSkills, job.RequiredSkills, job.OptionalSkills)

	item := entity.RecommendationItem{
		JobID:         job.ID,
		Title:         job.Title,
		Category:      job.Category,
		Company:       job.Company,
		Location:      job.Location,
		WorkType:      job.WorkType,
		Source:        job.Source,
		SourceURL:     job.SourceURL,
		MatchedSkills: gap.Matched,
		MissingSkills: gap.Missing,
	}

	switch {
	case gap.RequiredTotal == 0:
		// Lowongan tanpa syarat tidak bisa diberi skor.
		item.MissingSkills = []string{}
		item.ScoreStatus = entity.ScoreStatusPersyaratanBelumTersedia
	case !profile.HasSkills():
		item.ScoreStatus = entity.ScoreStatusProfilTanpaKeterampilan
	default:
		score := matchingScore(gap)
		readiness := readinessScore(gap, completeness)
		item.MatchingScore = &score
		item.ReadinessScore = &readiness
		item.ScoreStatus = entity.ScoreStatusOK
	}

	return item
}

// rankItems mengurutkan item lalu memberi nomor peringkat.
//
// Urutan: matching_score menurun, lalu readiness_score menurun, lalu job_id
// menaik agar hasil selalu sama untuk masukan yang sama.
func rankItems(items []entity.RecommendationItem) []entity.RecommendationItem {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]

		leftScore := scoreOrZero(left.MatchingScore)
		rightScore := scoreOrZero(right.MatchingScore)
		if leftScore != rightScore {
			return leftScore > rightScore
		}

		leftReadiness := scoreOrZero(left.ReadinessScore)
		rightReadiness := scoreOrZero(right.ReadinessScore)
		if leftReadiness != rightReadiness {
			return leftReadiness > rightReadiness
		}

		return left.JobID < right.JobID
	})

	rank := 0
	for index := range items {
		if items[index].MatchingScore == nil {
			continue
		}
		rank++
		current := rank
		items[index].Rank = &current
	}

	return items
}

func scoreOrZero(value *int) int {
	if value == nil {
		return -1
	}
	return *value
}
