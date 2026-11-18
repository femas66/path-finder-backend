package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLexicon(t *testing.T) *SkillLexicon {
	t.Helper()

	lexicon, err := DefaultSkillLexicon()
	require.NoError(t, err)
	return lexicon
}

func TestSkillLexiconExtract(t *testing.T) {
	lexicon := newTestLexicon(t)

	t.Run("mengenali skill teknis dari narasi", func(t *testing.T) {
		profile := lexicon.Extract(
			"Saya membangun backend dengan Go dan PostgreSQL, memakai REST API, Git, serta Docker.",
		)

		assert.Contains(t, profile.HardSkills, "Go")
		assert.Contains(t, profile.HardSkills, "PostgreSQL")
		assert.Contains(t, profile.HardSkills, "REST API")
		assert.Contains(t, profile.HardSkills, "Git")
		assert.Contains(t, profile.HardSkills, "Docker")
	})

	t.Run("tanda baca akhir kalimat tidak menghalangi", func(t *testing.T) {
		profile := lexicon.Extract("Saya sering memakai PostgreSQL. Dan juga Docker.")

		assert.Contains(t, profile.HardSkills, "PostgreSQL")
		assert.Contains(t, profile.HardSkills, "Docker")
	})

	t.Run("Java tidak ikut terdeteksi dari JavaScript", func(t *testing.T) {
		profile := lexicon.Extract("Saya menulis JavaScript setiap hari.")

		assert.Contains(t, profile.HardSkills, "JavaScript")
		assert.NotContains(t, profile.HardSkills, "Java")
	})

	t.Run("alias bahasa Indonesia dikenali", func(t *testing.T) {
		profile := lexicon.Extract("Saya memakai golang untuk layanan backend.")

		assert.Contains(t, profile.HardSkills, "Go")
	})

	t.Run("simbol teknis pada nama skill dipertahankan", func(t *testing.T) {
		profile := lexicon.Extract("Saya terbiasa dengan C++ dan C# serta .NET.")

		assert.Contains(t, profile.HardSkills, "C++")
		assert.Contains(t, profile.HardSkills, "C#")
		assert.Contains(t, profile.HardSkills, ".NET")
	})

	t.Run("soft skill frasa Indonesia dikenali", func(t *testing.T) {
		profile := lexicon.Extract("Saya senang bekerja dalam tim dan mempelajari hal baru.")

		assert.Contains(t, profile.SoftSkills, "Teamwork")
		assert.Contains(t, profile.SoftSkills, "Continuous Learning")
	})

	t.Run("narasi tanpa skill teknis", func(t *testing.T) {
		profile := lexicon.Extract("Setiap pagi saya menyusuri sawah dan menikmati udara dingin.")

		assert.False(t, profile.HasSkills())
		assert.Empty(t, profile.HardSkills)
	})

	t.Run("skill tidak duplikat", func(t *testing.T) {
		profile := lexicon.Extract("Go, go, GOLANG, dan golang.")

		assert.Equal(t, []string{"Go"}, profile.HardSkills)
	})
}

func TestSkillLexiconNormalize(t *testing.T) {
	lexicon := newTestLexicon(t)

	t.Run("nama kanonik", func(t *testing.T) {
		assert.Equal(t, "Go", lexicon.NormalizeSkillName("golang"))
		assert.Equal(t, "PostgreSQL", lexicon.NormalizeSkillName("PostgreSQL"))
	})

	t.Run("nama tidak dikenal dikembalikan apa adanya", func(t *testing.T) {
		assert.Equal(t, "Skill Aneh", lexicon.NormalizeSkillName("Skill Aneh"))
	})

	t.Run("daftar dinormalkan dan dideduplikasi", func(t *testing.T) {
		result := lexicon.NormalizeSkillList([]string{"golang", "Go", "postgresql", " "})

		assert.Equal(t, []string{"Go", "PostgreSQL"}, result)
	})
}

func TestComputeSkillGap(t *testing.T) {
	t.Run("menghitung cocok dan kurang", func(t *testing.T) {
		gap := computeSkillGap(
			[]string{"Go", "Git"},
			[]string{"Go", "PostgreSQL", "Git", "Docker"},
			[]string{"Fiber", "Gin"},
		)

		assert.Equal(t, []string{"Go", "Git"}, gap.Matched)
		assert.Equal(t, []string{"PostgreSQL", "Docker"}, gap.Missing)
		assert.Equal(t, []string{}, gap.OptionalMatched)
		assert.Equal(t, 4, gap.RequiredTotal)
		assert.InDelta(t, 0.5, gap.HardMatch, 0.0001)
	})

	t.Run("tidak peka huruf besar kecil", func(t *testing.T) {
		gap := computeSkillGap([]string{"go"}, []string{"Go"}, nil)

		assert.Equal(t, []string{"Go"}, gap.Matched)
		assert.Equal(t, []string{}, gap.Missing)
	})

	t.Run("lowongan tanpa syarat", func(t *testing.T) {
		gap := computeSkillGap([]string{"Go"}, nil, nil)

		assert.Equal(t, 0, gap.RequiredTotal)
		assert.Equal(t, []string{}, gap.Missing)
	})
}

func TestMatchingAndReadinessScore(t *testing.T) {
	t.Run("cocok penuh tanpa skill opsional", func(t *testing.T) {
		gap := computeSkillGap([]string{"Go", "Git"}, []string{"Go", "Git"}, nil)

		assert.Equal(t, 100, matchingScore(gap))
	})

	t.Run("cocok sebagian", func(t *testing.T) {
		gap := computeSkillGap([]string{"Go"}, []string{"Go", "Git"}, nil)

		assert.Equal(t, 50, matchingScore(gap))
	})

	t.Run("skill opsional menambah skor", func(t *testing.T) {
		gap := computeSkillGap([]string{"Go", "Fiber"}, []string{"Go", "Git"}, []string{"Fiber", "Gin"})

		// 0.8 * (1/2) + 0.2 * (1/2) = 0.5
		assert.Equal(t, 50, matchingScore(gap))
	})

	t.Run("kesiapan memakai bobot 70/10", func(t *testing.T) {
		gap := computeSkillGap([]string{"Go"}, []string{"Go"}, nil)

		// (0.7 * 1.0 + 0.1 * 0.25) / 0.8 = 0.90625 -> 91
		assert.Equal(t, 91, readinessScore(gap, 0.25))
	})

	t.Run("lowongan tanpa syarat tidak diberi skor", func(t *testing.T) {
		gap := computeSkillGap([]string{"Go"}, nil, nil)

		assert.Equal(t, 0, matchingScore(gap))
		assert.Equal(t, 0, readinessScore(gap, 0.25))
	})

	t.Run("skor selalu dalam rentang 0-100", func(t *testing.T) {
		assert.Equal(t, 0, clampScore(-10))
		assert.Equal(t, 100, clampScore(120))
	})
}
