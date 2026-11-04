package handlers_test

import (
	"bytes"
	"net/http/httptest"
	"regexp"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/tests/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCreateProjectGroupValidation(t *testing.T) {
	app := setupApp()
	app.Post("/project-groups", handlers.CreateProjectGroup)

	for name, body := range map[string]string{
		"missing nama": `{}`,
		"blank nama":   `{"nama":"   "}`,
		"bad json":     `{`,
	} {
		t.Run(name, func(t *testing.T) {
			testutil.SetupMockDB(t)
			req := httptest.NewRequest("POST", "/project-groups", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			resp, _ := app.Test(req)
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestCreateProjectGroupDuplicateName(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Post("/project-groups", handlers.CreateProjectGroup)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "project_groups" WHERE LOWER(nama) = LOWER($1)`)).
		WithArgs("Pengadaan TAD").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	req := httptest.NewRequest("POST", "/project-groups", bytes.NewBufferString(`{"nama":"Pengadaan TAD"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestDeleteProjectGroupWithProjectsIsRejected(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Delete("/project-groups/:id", handlers.DeleteProjectGroup)

	id := uuid.New().String()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "projects" WHERE project_group_id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	// No DELETE is expected: any further statement fails the mock.

	resp, _ := app.Test(httptest.NewRequest("DELETE", "/project-groups/"+id, nil))
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestDeleteProjectGroupNotFound(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Delete("/project-groups/:id", handlers.DeleteProjectGroup)

	id := uuid.New().String()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "projects" WHERE project_group_id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "project_groups" WHERE id = $1`)).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	resp, _ := app.Test(httptest.NewRequest("DELETE", "/project-groups/"+id, nil))
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestUpdateProjectGroupNotFound(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Put("/project-groups/:id", handlers.UpdateProjectGroup)

	id := uuid.New().String()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "project_groups" WHERE id = $1`)).
		WithArgs(id).
		WillReturnError(gorm.ErrRecordNotFound)

	req := httptest.NewRequest("PUT", "/project-groups/"+id, bytes.NewBufferString(`{"nama":"X"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
