package test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newJobStatusTestFixture(t *testing.T) *jobTestFixture {
	t.Helper()
	f := newJobTestFixture(t)
	// Exercise PostgreSQL's microsecond rounding rather than relying on clock precision.
	f.db.NowFunc = func() time.Time { return time.Now().UTC().Truncate(time.Microsecond).Add(789 * time.Nanosecond) }
	return f
}

func TestJobsStatusCreateDefaultsAndValidation(t *testing.T) {
	f := newJobStatusTestFixture(t)
	for _, tc := range []struct {
		name  string
		value any
		omit  bool
		want  string
	}{
		{"omitted", nil, true, entity.JobStatusAktif},
		{"empty", "", false, entity.JobStatusAktif},
		{"whitespace", " \t ", false, entity.JobStatusAktif},
		{"null", nil, false, entity.JobStatusAktif},
		{"active", entity.JobStatusAktif, false, entity.JobStatusAktif},
		{"closed", entity.JobStatusDitutup, false, entity.JobStatusDitutup},
		{"trimmed", " ditutup ", false, entity.JobStatusDitutup},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := jobTestPayload()
			delete(payload, "optional_skills")
			if !tc.omit {
				payload["status"] = tc.value
			}
			created := f.create(t, f.ownerToken, payload)
			require.Equal(t, tc.want, created.Status)
			require.Equal(t, []string{"Go", "PostgreSQL"}, created.RequiredSkills)
			require.NotNil(t, created.OptionalSkills)
			require.Empty(t, created.OptionalSkills)
			stored := f.stored(t, created.ID)
			require.Equal(t, tc.want, stored.Status)
			require.Equal(t, created.RequiredSkills, stored.RequiredSkills)
			var detail model.WebResponse[model.JobResponse]
			f.request(t, "GET", "/api/jobs/"+created.ID, f.studentToken, nil, 200, &detail)
			requirePersistedJobEqual(t, created, detail.Data)
		})
	}
	for _, invalid := range []any{"closed", "AKTIF", "Ditutup", "aktif ditutup", 1, true, []string{"aktif"}, map[string]string{"value": "aktif"}} {
		t.Run(fmt.Sprintf("invalid_%v", invalid), func(t *testing.T) {
			var before int64
			require.NoError(t, f.db.Model(&entity.Job{}).Count(&before).Error)
			payload := jobTestPayload()
			payload["status"] = invalid
			f.request(t, "POST", "/api/mitra/jobs", f.ownerToken, payload, 400, nil)
			var after int64
			require.NoError(t, f.db.Model(&entity.Job{}).Count(&after).Error)
			require.Equal(t, before, after)
		})
	}
}

func TestJobsStatusLifecycleAndAuthorization(t *testing.T) {
	f := newJobStatusTestFixture(t)
	for _, method := range []string{"PATCH", "PUT"} {
		t.Run(method, func(t *testing.T) {
			created := f.create(t, f.ownerToken, jobTestPayload())
			path := "/api/mitra/jobs/" + created.ID
			for _, status := range []string{entity.JobStatusDitutup, entity.JobStatusAktif} {
				before := f.stored(t, created.ID)
				body := map[string]any{"status": status}
				f.request(t, method, path, "", body, 401, nil)
				f.request(t, method, path, f.otherToken, body, 404, nil)
				for _, token := range []string{f.studentToken, f.adminToken} {
					f.request(t, method, path, token, body, 403, nil)
				}
				require.Equal(t, before, f.stored(t, created.ID))
				var changed model.WebResponse[model.JobResponse]
				f.request(t, method, path, f.ownerToken, body, 200, &changed)
				require.Equal(t, status, changed.Data.Status)
				require.Equal(t, created.ID, changed.Data.ID)
				require.Equal(t, created.MitraID, changed.Data.MitraID)
				require.Equal(t, created.Title, changed.Data.Title)
				require.Equal(t, created.RequiredSkills, changed.Data.RequiredSkills)
				require.Equal(t, created.OptionalSkills, changed.Data.OptionalSkills)
				require.WithinDuration(t, created.CreatedAt, changed.Data.CreatedAt, time.Microsecond)
				require.False(t, changed.Data.UpdatedAt.Before(before.UpdatedAt))
				require.Equal(t, status, f.stored(t, created.ID).Status)
				for _, token := range []string{f.studentToken, f.otherToken, f.adminToken} {
					var detail model.WebResponse[model.JobResponse]
					f.request(t, "GET", "/api/jobs/"+created.ID, token, nil, 200, &detail)
					requirePersistedJobEqual(t, changed.Data, detail.Data)
				}
				f.request(t, "GET", "/api/jobs/"+created.ID, "", nil, 401, nil)
				var own model.WebResponse[model.JobResponse]
				f.request(t, "GET", path, f.ownerToken, nil, 200, &own)
				requirePersistedJobEqual(t, changed.Data, own.Data)
				f.request(t, "GET", path, f.otherToken, nil, 404, nil)
				listed := f.list(t, "/api/jobs?search="+url.QueryEscape(created.Title)+"&size=100", f.studentToken)
				found := false
				for _, job := range listed.Data {
					found = found || job.ID == created.ID
				}
				require.Equal(t, status == entity.JobStatusAktif, found)
			}
		})
	}
}

func TestJobsStatusUpdateValidationAndNull(t *testing.T) {
	f := newJobStatusTestFixture(t)
	for _, method := range []string{"PATCH", "PUT"} {
		for _, initial := range []string{entity.JobStatusAktif, entity.JobStatusDitutup} {
			t.Run(method+"/"+initial, func(t *testing.T) {
				payload := jobTestPayload()
				payload["status"] = initial
				created := f.create(t, f.ownerToken, payload)
				path := "/api/mitra/jobs/" + created.ID
				for _, invalid := range []any{"", " \t ", "invalid", "AKTIF", "Ditutup", 7, false, []string{}, map[string]string{}} {
					before := f.stored(t, created.ID)
					f.request(t, method, path, f.ownerToken, map[string]any{"status": invalid, "title": "Must Not Persist"}, 400, nil)
					require.Equal(t, before, f.stored(t, created.ID), "invalid status %v must not partially update", invalid)
				}
				for _, body := range []map[string]any{{}, {"status": nil}, {"status": nil, "location": "Bandung"}} {
					var updated model.WebResponse[model.JobResponse]
					f.request(t, method, path, f.ownerToken, body, 200, &updated)
					require.Equal(t, initial, updated.Data.Status)
					require.Equal(t, initial, f.stored(t, created.ID).Status)
					require.Equal(t, created.RequiredSkills, updated.Data.RequiredSkills)
					if body["location"] != nil {
						require.Equal(t, "Bandung", updated.Data.Location)
					}
				}
				var trimmed model.WebResponse[model.JobResponse]
				f.request(t, method, path, f.ownerToken, map[string]any{"status": " ditutup "}, 200, &trimmed)
				require.Equal(t, entity.JobStatusDitutup, trimmed.Data.Status)
			})
		}
	}
}

func TestJobsStatusListFiltersAndCounts(t *testing.T) {
	f := newJobStatusTestFixture(t)

	var ownedActive, ownedClosed, otherActive, otherClosed []string
	for _, token := range []string{f.ownerToken, f.otherToken} {
		for _, status := range []string{entity.JobStatusAktif, entity.JobStatusDitutup} {
			for i := 0; i < 3; i++ {
				payload := jobTestPayload()
				payload["title"] = fmt.Sprintf("StatusCatalog Backend %d", i)
				payload["status"] = status
				created := f.create(t, token, payload)
				switch {
				case token == f.ownerToken && status == entity.JobStatusAktif:
					ownedActive = append(ownedActive, created.ID)
				case token == f.ownerToken:
					ownedClosed = append(ownedClosed, created.ID)
				case status == entity.JobStatusAktif:
					otherActive = append(otherActive, created.ID)
				default:
					otherClosed = append(otherClosed, created.ID)
				}
			}
		}
	}
	allActive := append(append([]string{}, ownedActive...), otherActive...)
	allClosed := append(append([]string{}, ownedClosed...), otherClosed...)
	for index, tc := range []struct {
		path, token string
		ids         []string
	}{
		{"/api/jobs?search=StatusCatalog", f.studentToken, allActive},
		{"/api/jobs?search=StatusCatalog&status=aktif", f.studentToken, allActive},
		{"/api/jobs?search=StatusCatalog&status=ditutup", f.studentToken, allClosed},
		{"/api/jobs?search=StatusCatalog&status=", f.studentToken, allActive},
		{"/api/jobs?search=StatusCatalog&status=%20ditutup%20", f.adminToken, allClosed},
		{"/api/mitra/jobs?search=StatusCatalog", f.ownerToken, append(append([]string{}, ownedActive...), ownedClosed...)},
		{"/api/mitra/jobs?search=StatusCatalog&status=", f.ownerToken, append(append([]string{}, ownedActive...), ownedClosed...)},
		{"/api/mitra/jobs?search=StatusCatalog&status=aktif", f.ownerToken, ownedActive},
		{"/api/mitra/jobs?search=StatusCatalog&status=ditutup", f.ownerToken, ownedClosed},
		{"/api/mitra/jobs?search=StatusCatalog&status=ditutup", f.otherToken, otherClosed},
	} {
		t.Run(fmt.Sprintf("%02d/%s", index, tc.path), func(t *testing.T) {
			result := f.list(t, tc.path+"&size=100", tc.token)
			require.EqualValues(t, len(tc.ids), result.Paging.TotalItem)
			require.EqualValues(t, 1, result.Paging.TotalPage)
			ids := make([]string, 0, len(result.Data))
			for _, job := range result.Data {
				ids = append(ids, job.ID)
			}
			require.ElementsMatch(t, tc.ids, ids)
			var pagedIDs []string
			for page := 1; page <= (len(tc.ids)+1)/2; page++ {
				paged := f.list(t, tc.path+fmt.Sprintf("&size=2&page=%d", page), tc.token)
				require.EqualValues(t, len(tc.ids), paged.Paging.TotalItem)
				require.EqualValues(t, (len(tc.ids)+1)/2, paged.Paging.TotalPage)
				for _, job := range paged.Data {
					pagedIDs = append(pagedIDs, job.ID)
				}
			}
			require.Equal(t, ids, pagedIDs)
			beyond := f.list(t, tc.path+"&size=2&page=99", tc.token)
			require.Empty(t, beyond.Data)
			require.EqualValues(t, len(tc.ids), beyond.Paging.TotalItem)
		})
	}
	catalog := f.list(t, "/api/jobs?size=100", f.studentToken)
	require.EqualValues(t, len(allActive), catalog.Paging.TotalItem)
	for _, path := range []string{"/api/jobs", "/api/mitra/jobs"} {
		for _, status := range []string{"invalid", "AKTIF", "null", "aktif,ditutup", "%' OR 1=1 --"} {
			f.request(t, "GET", path+"?status="+url.QueryEscape(status), f.ownerToken, nil, 400, nil)
		}
		for _, status := range []string{entity.JobStatusAktif, entity.JobStatusDitutup} {
			query := "?status=" + status + "&search=StatusCatalog&category=" + url.QueryEscape("Frontend Developer Intern")
			empty := f.list(t, path+query, f.ownerToken)
			require.Empty(t, empty.Data)
			require.Zero(t, empty.Paging.TotalItem)
			require.Zero(t, empty.Paging.TotalPage)
			matched := f.list(t, path+"?status="+status+"&search=StatusCatalog&category="+url.QueryEscape("Backend Developer Intern"), f.ownerToken)
			want := 6
			if path == "/api/mitra/jobs" {
				want = 3
			}
			require.EqualValues(t, want, matched.Paging.TotalItem)
		}
		f.request(t, "GET", path+"?status=ditutup", "", nil, 401, nil)
	}
}

func TestJobsStatusActiveCatalogSelection(t *testing.T) {
	f := newJobStatusTestFixture(t)
	active := f.create(t, f.ownerToken, jobTestPayload())
	closedPayload := jobTestPayload()
	closedPayload["status"] = entity.JobStatusDitutup
	closed := f.create(t, f.otherToken, closedPayload)
	repo := repository.NewJobRepository(f.log)
	assertCatalog := func(activeIDs []string) {
		t.Helper()
		var activeJobs []entity.Job
		require.NoError(t, repo.FindActive(f.db, &activeJobs))
		ids := make([]string, 0, len(activeJobs))
		for _, job := range activeJobs {
			require.Equal(t, entity.JobStatusAktif, job.Status)
			ids = append(ids, job.ID)
		}
		require.ElementsMatch(t, activeIDs, ids)
		var allJobs []entity.Job
		require.NoError(t, repo.FindAll(f.db, &allJobs))
		allIDs := make([]string, 0, len(allJobs))
		for _, job := range allJobs {
			allIDs = append(allIDs, job.ID)
			require.Equal(t, f.stored(t, job.ID), job)
		}
		require.ElementsMatch(t, []string{active.ID, closed.ID}, allIDs)
	}
	assertCatalog([]string{active.ID})
	f.request(t, "PATCH", "/api/mitra/jobs/"+active.ID, f.ownerToken, map[string]any{"status": entity.JobStatusDitutup}, 200, nil)
	require.Equal(t, entity.JobStatusDitutup, f.stored(t, active.ID).Status)
	assertCatalog(nil)
	f.request(t, "PUT", "/api/mitra/jobs/"+closed.ID, f.otherToken, map[string]any{"status": entity.JobStatusAktif}, 200, nil)
	require.Equal(t, entity.JobStatusAktif, f.stored(t, closed.ID).Status)
	assertCatalog([]string{closed.ID})
	f.request(t, "PATCH", "/api/mitra/jobs/"+active.ID, f.ownerToken, map[string]any{"status": entity.JobStatusAktif}, 200, nil)
	assertCatalog([]string{active.ID, closed.ID})
}

func TestJobsStatusLegacyMigrationAndDatabaseDefault(t *testing.T) {
	f := newJobStatusTestFixture(t)
	legacy := f.create(t, f.ownerToken, jobTestPayload())
	unowned := entity.Job{
		ID: uuid.NewString(), Title: "Legacy Unowned Job", Category: "Backend Developer Intern",
		RequiredSkills: []string{"Go"}, OptionalSkills: []string{"PostgreSQL"},
	}
	require.NoError(t, f.db.Create(&unowned).Error)
	var before []entity.Job
	require.NoError(t, f.db.Order("id ASC").Find(&before).Error)
	require.Len(t, before, 2, "include owned and unowned legacy jobs")
	// This schema belongs only to this test; simulate jobs created before status existed.
	require.NoError(t, f.db.Migrator().DropColumn(&entity.Job{}, "Status"))
	require.False(t, f.db.Migrator().HasColumn(&entity.Job{}, "Status"))
	require.NoError(t, f.db.AutoMigrate(&entity.Job{}))
	require.True(t, f.db.Migrator().HasIndex(&entity.Job{}, "Status"))
	// Recreated status changes SELECT * column order; discard pgx's pre-DDL plans.
	connection, err := f.db.DB()
	require.NoError(t, err)
	connection.SetMaxIdleConns(0)
	connection.SetMaxIdleConns(1)
	var after []entity.Job
	require.NoError(t, f.db.Order("id ASC").Find(&after).Error)
	require.Equal(t, before, after, "owned and unowned legacy rows must gain aktif without losing fields")
	require.Equal(t, entity.JobStatusAktif, f.stored(t, legacy.ID).Status)
	var column struct {
		IsNullable    string
		ColumnDefault string
	}
	require.NoError(t, f.db.Raw("SELECT is_nullable, column_default FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'jobs' AND column_name = 'status'").Scan(&column).Error)
	require.Equal(t, "NO", column.IsNullable)
	require.Contains(t, column.ColumnDefault, "aktif")
	rawID := uuid.NewString()
	require.NoError(t, f.db.Exec("INSERT INTO jobs (id, title, category) VALUES (?, ?, ?)", rawID, "Raw Default Job", "Backend Developer Intern").Error)
	require.Equal(t, entity.JobStatusAktif, f.stored(t, rawID).Status)
	ormJob := entity.Job{ID: uuid.NewString(), Title: "ORM Default Job", Category: "Backend Developer Intern", RequiredSkills: []string{"Go"}}
	require.NoError(t, f.db.Create(&ormJob).Error)
	require.Equal(t, entity.JobStatusAktif, ormJob.Status)
	require.Equal(t, entity.JobStatusAktif, f.stored(t, ormJob.ID).Status)
	err = f.db.Exec("INSERT INTO jobs (id, title, category, status) VALUES (?, ?, ?, NULL)", uuid.NewString(), "Invalid Null Job", "Backend Developer Intern").Error
	require.ErrorContains(t, err, "23502")
	// Repeated migrations must not reopen explicitly closed jobs.
	f.request(t, "PATCH", "/api/mitra/jobs/"+legacy.ID, f.ownerToken, map[string]any{"status": entity.JobStatusDitutup}, 200, nil)
	require.NoError(t, f.db.AutoMigrate(&entity.Job{}))
	require.Equal(t, entity.JobStatusDitutup, f.stored(t, legacy.ID).Status)
}
