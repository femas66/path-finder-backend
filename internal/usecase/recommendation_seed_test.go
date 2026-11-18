package usecase

import (
	"testing"

	"haji-koding/path-finder/internal/dataset"
	"haji-koding/path-finder/internal/entity"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestUseCase membuat usecase minimal untuk menguji parsing dataset.
// Hanya Log dan Lexicon yang dipakai, sehingga tidak perlu koneksi database.
func newTestUseCase(t *testing.T) *RecommendationUseCase {
	t.Helper()

	logger := logrus.New()
	lexicon, err := DefaultSkillLexicon()
	require.NoError(t, err)

	return &RecommendationUseCase{
		Log:     logger,
		Lexicon: lexicon,
	}
}

func TestParseJobDataset(t *testing.T) {
	useCase := newTestUseCase(t)

	jobs, err := useCase.parseJobDataset(dataset.JobsCSV)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	t.Run("kategori selalu posisi resmi", func(t *testing.T) {
		for _, job := range jobs {
			assert.Truef(
				t,
				entity.IsValidPosition(job.Category),
				"lowongan %s memiliki kategori tidak resmi: %q",
				job.ID,
				job.Category,
			)
		}
	})

	t.Run("delapan posisi resmi tercakup", func(t *testing.T) {
		covered := make(map[string]bool, len(entity.Positions))
		for _, job := range jobs {
			covered[job.Category] = true
		}

		for _, position := range entity.Positions {
			assert.Truef(t, covered[position], "belum ada lowongan untuk posisi %q", position)
		}
	})

	t.Run("setiap lowongan punya syarat dan identitas", func(t *testing.T) {
		seenIDs := make(map[string]bool, len(jobs))

		for _, job := range jobs {
			assert.NotEmpty(t, job.ID)
			assert.NotEmpty(t, job.Title)
			assert.NotEmpty(t, job.Company)
			assert.NotEmpty(t, job.RequiredSkills, "lowongan %s tanpa skill wajib", job.ID)
			assert.Falsef(t, seenIDs[job.ID], "ID lowongan duplikat: %s", job.ID)
			seenIDs[job.ID] = true
		}
	})

	t.Run("skill dinormalkan ke nama kanonik", func(t *testing.T) {
		for _, job := range jobs {
			for _, skill := range job.RequiredSkills {
				assert.Equalf(
					t,
					skill,
					useCase.Lexicon.NormalizeSkillName(skill),
					"skill %q pada lowongan %s belum kanonik",
					skill,
					job.ID,
				)
			}
		}
	})
}
