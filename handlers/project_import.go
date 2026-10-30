package handlers

import (
	"strconv"
	"strings"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ProjectImportRow is one row of the project Excel template. Every cell
// arrives as text; amounts and dates are parsed and validated here.
type ProjectImportRow struct {
	Row               int    `json:"row"`
	NoSP2K            string `json:"no_sp2k"`
	NoPerjanjian      string `json:"no_perjanjian"`
	NoAmandemen       string `json:"no_amandemen"`
	TanggalPerjanjian string `json:"tanggal_perjanjian"`
	JudulPekerjaan    string `json:"judul_pekerjaan"`
	TanggalMulai      string `json:"tanggal_mulai"`
	TanggalSelesai    string `json:"tanggal_selesai"`
	NilaiPekerjaan    string `json:"nilai_pekerjaan"`
	Client            string `json:"client"`
	ContractType      string `json:"contract_type"`
}

// optionalString returns nil for an empty cell.
func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ImportProjects godoc
// @Summary Import projects from Excel rows
// @Description Validates every row first; if any row is invalid nothing is saved and all errors are returned (422). A row whose no_sp2k already exists updates that project.
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "rows: list of project rows (see template)"
// @Success 200 {object} map[string]any "total, created, updated"
// @Failure 400 {object} map[string]any "Invalid body or too many rows"
// @Failure 422 {object} map[string]any "Validation errors per row"
// @Router /api/projects/import [post]
func ImportProjects(c *fiber.Ctx) error {
	rows, err := bindImportRows[ProjectImportRow](c)
	if err != nil {
		return respondImportError(c, err)
	}

	clients, err := loadNameIndex("clients", "name")
	if err != nil {
		return respondImportError(c, err)
	}
	contractTypes, err := loadNameIndex("contract_types", "name")
	if err != nil {
		return respondImportError(c, err)
	}

	existing, err := loadProjectIndex()
	if err != nil {
		return respondImportError(c, err)
	}

	errs := &importErrors{}
	seen := make(map[string]int, len(rows))
	requests := make([]*models.ProjectRequest, len(rows))

	for i := range rows {
		r := &rows[i]
		r.NoSP2K = strings.TrimSpace(r.NoSP2K)
		r.JudulPekerjaan = strings.TrimSpace(r.JudulPekerjaan)
		r.TanggalMulai = strings.TrimSpace(r.TanggalMulai)
		r.TanggalSelesai = strings.TrimSpace(r.TanggalSelesai)
		r.TanggalPerjanjian = strings.TrimSpace(r.TanggalPerjanjian)
		r.NilaiPekerjaan = strings.TrimSpace(r.NilaiPekerjaan)

		before := len(errs.list)

		errs.requireFields(r.Row,
			importField{"no_sp2k", r.NoSP2K},
			importField{"judul_pekerjaan", r.JudulPekerjaan},
			importField{"tanggal_mulai", r.TanggalMulai},
			importField{"tanggal_selesai", r.TanggalSelesai},
			importField{"nilai_pekerjaan", r.NilaiPekerjaan},
		)

		if r.NoSP2K != "" {
			key := helpers.NormalizeKey(r.NoSP2K)
			if first, dup := seen[key]; dup {
				errs.add(r.Row, "no_sp2k", "duplikat dengan baris "+strconv.Itoa(first)+" di file yang sama")
			} else {
				seen[key] = r.Row
			}
		}

		req := &models.ProjectRequest{
			NoSP2K:            r.NoSP2K,
			NoPerjanjian:      optionalString(strings.TrimSpace(r.NoPerjanjian)),
			NoAmandemen:       optionalString(strings.TrimSpace(r.NoAmandemen)),
			TanggalPerjanjian: optionalString(r.TanggalPerjanjian),
			JudulPekerjaan:    r.JudulPekerjaan,
			TanggalMulai:      r.TanggalMulai,
			TanggalSelesai:    r.TanggalSelesai,
		}

		if r.NilaiPekerjaan != "" {
			if v, ok := helpers.ParseImportAmount(r.NilaiPekerjaan); ok {
				req.NilaiPekerjaan = v
			} else {
				errs.add(r.Row, "nilai_pekerjaan", "harus berupa angka")
			}
		}

		if name := strings.TrimSpace(r.Client); name != "" {
			if id, ok := clients[helpers.NormalizeKey(name)]; ok {
				idStr := id.String()
				req.ClientID = &idStr
			} else {
				errs.add(r.Row, "client", "klien '"+name+"' tidak ditemukan")
			}
		}
		if name := strings.TrimSpace(r.ContractType); name != "" {
			if id, ok := contractTypes[helpers.NormalizeKey(name)]; ok {
				idStr := id.String()
				req.ContractTypeID = &idStr
			} else {
				errs.add(r.Row, "contract_type", "jenis kontrak '"+name+"' tidak ditemukan")
			}
		}

		// Cross-field rules (dates) only make sense once the row is complete
		// and numeric cells parsed.
		if len(errs.list) == before {
			if err := ValidateProjectRequest(req); err != nil {
				errs.addValidation(r.Row, err)
			} else if err := applyProjectRequest(&models.Project{}, req); err != nil {
				errs.addValidation(r.Row, err)
			}
		}
		requests[i] = req
	}

	if len(errs.list) > 0 {
		return respondImportInvalid(c, errs)
	}

	createdBy, _ := c.Locals("userEmail").(string)
	created, updated := 0, 0
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		for _, req := range requests {
			project, isUpdate := existing[helpers.NormalizeKey(req.NoSP2K)]
			if !isUpdate {
				project = &models.Project{CreatedBy: createdBy}
			}
			if err := applyProjectRequest(project, req); err != nil {
				return err
			}
			if isUpdate {
				updated++
			} else {
				created++
			}
			if err := tx.Save(project).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return respondImportError(c, err)
	}
	return respondImportDone(c, len(rows), created, updated)
}
