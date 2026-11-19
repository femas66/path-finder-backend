package converter

import (
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
)

func ApplicationToResponse(app *entity.Application) *model.ApplicationResponse {
	if app == nil {
		return nil
	}

	res := &model.ApplicationResponse{
		ID:          app.ID,
		JobID:       app.JobID,
		MahasiswaID: app.MahasiswaID,
		Status:      app.Status,
		CreatedAt:   app.CreatedAt,
		UpdatedAt:   app.UpdatedAt,
	}

	if app.Job != nil {
		res.JobTitle = app.Job.Title
		res.Company = app.Job.Company
	}

	if app.Mahasiswa != nil {
		res.Mahasiswa = &model.MahasiswaApplicant{
			ID:             app.Mahasiswa.ID,
			Name:           app.Mahasiswa.Name,
			NIM:            app.Mahasiswa.NIM,
			Email:          app.Mahasiswa.Email,
			ProfilePicture: app.Mahasiswa.ProfilePicture,
			CV:             app.Mahasiswa.CV,
			PortfolioURL:   app.Mahasiswa.PortfolioURL,
			Location:       app.Mahasiswa.Location,
		}
	}

	return res
}

func ApplicationsToResponses(apps []entity.Application) []model.ApplicationResponse {
	responses := make([]model.ApplicationResponse, 0, len(apps))
	for i := range apps {
		if res := ApplicationToResponse(&apps[i]); res != nil {
			responses = append(responses, *res)
		}
	}
	return responses
}
