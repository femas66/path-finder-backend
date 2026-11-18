package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"

	"haji-koding/path-finder/internal/dataset"
	"haji-koding/path-finder/internal/entity"
)

// SeedJobs memastikan katalog lowongan pada dataset sudah ada di database.
//
// Proses ini idempoten: lowongan yang ID-nya sudah ada tidak ditimpa, sehingga
// perubahan data manual di database tidak hilang saat server dijalankan ulang.
func (u *RecommendationUseCase) SeedJobs(ctx context.Context) error {
	jobs, err := u.parseJobDataset(dataset.JobsCSV)
	if err != nil {
		return err
	}

	db := u.DB.WithContext(ctx)

	ids := make([]string, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.ID)
	}

	existing, err := u.JobRepository.FindExistingIDs(db, ids)
	if err != nil {
		return fmt.Errorf("gagal memeriksa katalog lowongan: %w", err)
	}

	missing := make([]entity.Job, 0, len(jobs))
	for _, job := range jobs {
		if existing[job.ID] {
			continue
		}
		missing = append(missing, job)
	}

	if len(missing) == 0 {
		return nil
	}

	if err := u.JobRepository.CreateBatch(db, missing); err != nil {
		return fmt.Errorf("gagal menyimpan katalog lowongan: %w", err)
	}

	u.Log.Infof("Katalog lowongan: %d lowongan baru ditambahkan", len(missing))
	return nil
}

// parseJobDataset membaca jobs.csv menjadi entity.Job siap simpan.
func (u *RecommendationUseCase) parseJobDataset(csvData []byte) ([]entity.Job, error) {
	reader := csv.NewReader(bytes.NewReader(csvData))
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("gagal membaca dataset lowongan: %w", err)
	}

	jobs := make([]entity.Job, 0, len(records))
	for index, record := range records {
		// Baris pertama adalah header.
		if index == 0 {
			continue
		}
		if len(record) < 11 {
			u.Log.Warnf("Lowongan pada baris %d dilewati: jumlah kolom kurang", index+1)
			continue
		}

		category := strings.TrimSpace(record[2])
		if !entity.IsValidPosition(category) {
			u.Log.Warnf("Lowongan pada baris %d dilewati: kategori %q bukan posisi resmi", index+1, category)
			continue
		}

		jobs = append(jobs, entity.Job{
			ID:             strings.TrimSpace(record[0]),
			Title:          strings.TrimSpace(record[1]),
			Category:       category,
			Company:        strings.TrimSpace(record[3]),
			Description:    strings.TrimSpace(record[4]),
			Location:       strings.TrimSpace(record[5]),
			WorkType:       strings.TrimSpace(record[6]),
			Source:         strings.TrimSpace(record[7]),
			SourceURL:      strings.TrimSpace(record[8]),
			RequiredSkills: u.Lexicon.NormalizeSkillList(splitSkills(record[9])),
			OptionalSkills: u.Lexicon.NormalizeSkillList(splitSkills(record[10])),
		})
	}

	return jobs, nil
}

// splitSkills memecah kolom skill yang dipisahkan titik koma.
func splitSkills(value string) []string {
	parts := strings.Split(value, ";")
	skills := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		skills = append(skills, trimmed)
	}
	return skills
}
