package seeds

import (
	"errors"
	"fmt"
	"math"
	"time"

	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DemoCreatedBy marks every project, transaction, invoice and operational
// expense written by DemoSeeder so DemoReset can remove exactly those rows.
const DemoCreatedBy = "demo-seed"

// DemoIDPrefix starts the fixed IDs of the demo groups and demo cost types.
const DemoIDPrefix = "40000000-"

// DemoInvoicePrefix starts the invoice number of every demo invoice.
const DemoInvoicePrefix = "DEMO-"

// demoNow is the simulated "today": invoices are only issued for periods that
// have ended, and a payment dated after this day is still pending (Submit).
var demoNow = time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)

// demoNamespace derives stable UUIDs for rows that have no hand-written ID, so
// re-running the seeder updates rows instead of duplicating them.
var demoNamespace = uuid.MustParse("40000000-0000-4000-8000-00000000ffff")

func demoID(parts ...any) uuid.UUID {
	return uuid.NewSHA1(demoNamespace, fmt.Append(nil, parts...))
}

func demoDate(y int, m time.Month, d int) models.Date {
	return models.Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}

func roundK(v float64) float64 { return math.Round(v/1000) * 1000 }

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

type demoCostType struct {
	id, name, code string
}

// Cost types are matched by name first; the two that already exist in a
// working database ("Gaji & Tunjangan", "Pelatihan") are reused, not duplicated.
var demoCostTypes = []demoCostType{
	{"40000000-0000-4000-8000-000000000101", "Gaji & Tunjangan", "PAY"},
	{"40000000-0000-4000-8000-000000000102", "BPJS Ketenagakerjaan", "BPJS-TK"},
	{"40000000-0000-4000-8000-000000000103", "BPJS Kesehatan", "BPJS-KS"},
	{"40000000-0000-4000-8000-000000000104", "THR & Bonus", "THR"},
	{"40000000-0000-4000-8000-000000000105", "Seragam & APD", "APD"},
	{"40000000-0000-4000-8000-000000000106", "Perlengkapan Kerja", "ALAT"},
	{"40000000-0000-4000-8000-000000000107", "Rekrutmen", "REKR"},
	{"40000000-0000-4000-8000-000000000108", "Transportasi Lapangan", "TRNS"},
	{"40000000-0000-4000-8000-000000000109", "Akomodasi", "AKOM"},
	{"40000000-0000-4000-8000-000000000110", "Pelatihan", "PLTH"},
}

type demoGroup struct {
	id, nama, deskripsi string
}

var demoGroups = []demoGroup{
	{"40000000-0000-4000-8000-000000000201", "Pengadaan TAD Sumbar", "Kontrak TAD Sumatera Barat 2026, dipecah per periode karena keterbatasan anggaran klien"},
	{"40000000-0000-4000-8000-000000000202", "Jasa EOS Sumut", "Kontrak EOS Sumatera Utara, berlanjut dari 2025 ke 2026"},
}

type demoProject struct {
	key            string // short code used in numbers
	id             string
	group          string // demo group ID, "" for none
	contractType   string // "TAD" or "EOS"
	sp2k           string
	judul          string
	start          time.Time
	months         int
	monthly        float64 // contract value per month
	workers        int
	method         string  // how this period is paid: Kontrak or SPR
	feePct         float64 // management fee as a share of the invoice value
	invoiceDay     int     // day of the following month the invoice is issued
	lateEveryNth   int     // every Nth period is billed one month late (0 = never)
	paymentLagDays int
}

func ym(y int, m time.Month) time.Time { return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC) }

var demoProjects = []demoProject{
	{"SBR1", "40000000-0000-4000-8000-000000000301", demoGroups[0].id, "TAD", "SP2K/DEMO/2026/001", "Pengadaan TAD Sumbar Jan-Feb 2026", ym(2026, 1), 2, 200_000_000, 40, models.PaymentMethodKontrak, 0.07, 8, 0, 30},
	{"SBR2", "40000000-0000-4000-8000-000000000302", demoGroups[0].id, "TAD", "SP2K/DEMO/2026/002", "Pengadaan TAD Sumbar Mar-Apr 2026 [SPR]", ym(2026, 3), 2, 200_000_000, 40, models.PaymentMethodSPR, 0.07, 12, 0, 39},
	{"SBR3", "40000000-0000-4000-8000-000000000303", demoGroups[0].id, "TAD", "SP2K/DEMO/2026/003", "Pengadaan TAD Sumbar Mei-Okt 2026", ym(2026, 5), 6, 200_000_000, 42, models.PaymentMethodKontrak, 0.07, 9, 3, 33},
	{"SMU1", "40000000-0000-4000-8000-000000000304", demoGroups[1].id, "EOS", "SP2K/DEMO/2025/001", "Jasa EOS Sumut Jan-Des 2025", ym(2025, 1), 12, 180_000_000, 35, models.PaymentMethodKontrak, 0.065, 7, 5, 35},
	{"SMU2", "40000000-0000-4000-8000-000000000305", demoGroups[1].id, "EOS", "SP2K/DEMO/2026/004", "Jasa EOS Sumut Jan-Jun 2026", ym(2026, 1), 6, 190_000_000, 36, models.PaymentMethodKontrak, 0.065, 7, 4, 32},
	{"SMU3", "40000000-0000-4000-8000-000000000306", demoGroups[1].id, "EOS", "SP2K/DEMO/2026/005", "Jasa EOS Sumut Jul-Des 2026", ym(2026, 7), 6, 190_000_000, 36, models.PaymentMethodKontrak, 0.065, 7, 0, 30},
	{"RIA1", "40000000-0000-4000-8000-000000000307", "", "TAD", "SP2K/DEMO/2025/002", "Pengadaan TAD Riau Jan-Des 2025", ym(2025, 1), 12, 160_000_000, 30, models.PaymentMethodKontrak, 0.08, 10, 6, 38},
	{"BTM1", "40000000-0000-4000-8000-000000000308", "", "TAD", "SP2K/DEMO/2026/006", "Pengadaan TAD Batam Mar-Des 2026", ym(2026, 3), 10, 150_000_000, 28, models.PaymentMethodKontrak, 0.08, 10, 0, 34},
}

var bulanID = [...]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func (p demoProject) month(i int) time.Time { return p.start.AddDate(0, i, 0) }

func (p demoProject) end() time.Time { return p.start.AddDate(0, p.months, -1) }

// DemoSeeder fills the database with a deterministic simulation of January
// 2025 up to early November 2026: project groups, projects, project expenses,
// operational expenses and invoices. It needs no existing data apart from
// what it creates itself (client ICON+, contract types EOS/TAD) and is safe to
// re-run: every row has a fixed ID and is upserted.
func DemoSeeder(db *gorm.DB) error {
	fmt.Println("📝 Seeding demo data (2025-2026)...")

	if err := OperationalCostTypesSeeder(db); err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		clientID, err := demoClient(tx)
		if err != nil {
			return err
		}
		contractTypes := map[string]uuid.UUID{}
		for _, name := range []string{"EOS", "TAD"} {
			cid, err := demoContractType(tx, name)
			if err != nil {
				return err
			}
			contractTypes[name] = cid
		}
		costTypes, err := demoSeedCostTypes(tx)
		if err != nil {
			return err
		}
		if err := demoSeedGroups(tx); err != nil {
			return err
		}
		if err := demoSeedProjects(tx, clientID, contractTypes); err != nil {
			return err
		}
		if err := demoSeedTransactions(tx, costTypes); err != nil {
			return err
		}
		if err := demoSeedCostOperations(tx); err != nil {
			return err
		}
		if err := demoSeedInvoices(tx); err != nil {
			return err
		}
		fmt.Println("✓ Demo data seeded")
		return nil
	})
}

// findOrCreateByName returns the ID of the row whose name column matches
// (case-insensitively), creating item when there is none. FirstOrCreate is not
// used: with a fixed ID on the struct it would add the ID to the lookup, miss a
// row that already exists under another ID and then fail on the unique name.
func findOrCreateByName[T any](tx *gorm.DB, column, name string, item *T, idOf func(*T) uuid.UUID) (uuid.UUID, error) {
	var existing T
	err := tx.Where("LOWER("+column+") = LOWER(?)", name).First(&existing).Error
	if err == nil {
		return idOf(&existing), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, err
	}
	if err := tx.Create(item).Error; err != nil {
		return uuid.Nil, err
	}
	return idOf(item), nil
}

func demoClient(tx *gorm.DB) (uuid.UUID, error) {
	c := models.Client{ID: id("50000000-0000-4000-8000-000000000001"), Name: "ICON+"}
	cid, err := findOrCreateByName(tx, "name", c.Name, &c, func(m *models.Client) uuid.UUID { return m.ID })
	if err != nil {
		return uuid.Nil, fmt.Errorf("client: %w", err)
	}
	return cid, nil
}

func demoContractType(tx *gorm.DB, name string) (uuid.UUID, error) {
	ids := map[string]string{"EOS": "50000000-0000-4000-8000-000000000011", "TAD": "50000000-0000-4000-8000-000000000012"}
	c := models.ContractType{ID: id(ids[name]), Name: name, Code: strPtr(name)}
	cid, err := findOrCreateByName(tx, "name", name, &c, func(m *models.ContractType) uuid.UUID { return m.ID })
	if err != nil {
		return uuid.Nil, fmt.Errorf("contract type %s: %w", name, err)
	}
	return cid, nil
}

// demoSeedCostTypes returns name -> ID, reusing a cost type that already
// exists under the same name.
func demoSeedCostTypes(tx *gorm.DB) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	for _, ct := range demoCostTypes {
		item := models.CostType{ID: id(ct.id), NamaBiaya: ct.name, Kode: strPtr(ct.code)}
		var existing models.CostType
		err := tx.Where("LOWER(nama_biaya) = LOWER(?)", ct.name).First(&existing).Error
		switch {
		case err == nil:
			out[ct.name] = existing.ID
			continue
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return nil, err
		}
		if err := tx.Create(&item).Error; err != nil {
			return nil, fmt.Errorf("cost type %q: %w", ct.name, err)
		}
		out[ct.name] = item.ID
	}
	return out, nil
}

func demoSeedGroups(tx *gorm.DB) error {
	for _, g := range demoGroups {
		item := models.ProjectGroup{ID: id(g.id), Nama: g.nama, Deskripsi: strPtr(g.deskripsi)}
		var existing models.ProjectGroup
		err := tx.Where("LOWER(nama) = LOWER(?)", g.nama).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("group %q: %w", g.nama, err)
		}
	}
	return nil
}

func demoSeedProjects(tx *gorm.DB, clientID uuid.UUID, contractTypes map[string]uuid.UUID) error {
	for _, p := range demoProjects {
		ct := contractTypes[p.contractType]
		item := models.Project{
			ID:                id(p.id),
			NoSP2K:            p.sp2k,
			NoPerjanjian:      strPtr("PJ/" + p.sp2k),
			TanggalPerjanjian: &models.Date{Time: p.start.AddDate(0, 0, -20)},
			JudulPekerjaan:    p.judul,
			JangkaWaktu:       intPtr(p.months),
			TanggalMulai:      models.Date{Time: p.start},
			TanggalSelesai:    models.Date{Time: p.end()},
			NilaiPekerjaan:    p.monthly * float64(p.months),
			ClientID:          &clientID,
			ContractTypeID:    &ct,
			CreatedBy:         DemoCreatedBy,
		}
		if p.group != "" {
			gid := id(p.group)
			item.ProjectGroupID = &gid
		}
		// Save upserts on the fixed primary key.
		if err := tx.Save(&item).Error; err != nil {
			return fmt.Errorf("project %s: %w", p.sp2k, err)
		}
	}
	return nil
}

// lastBilledMonth is the last contract month that has ended by demoNow.
func (p demoProject) lastMonthIndex() int {
	n := p.months - 1
	for n >= 0 && !p.month(n).AddDate(0, 1, 0).Before(demoNow) {
		n--
	}
	return n
}

func demoSeedTransactions(tx *gorm.DB, costTypes map[string]uuid.UUID) error {
	var rows []models.Transaction
	add := func(p demoProject, i int, name string, day int, amount float64, workers *int, desc string) {
		d := models.Date{Time: p.month(i).AddDate(0, 0, day-1)}
		bulan := helpers.ComputeBulanRealisasi(d)
		rows = append(rows, models.Transaction{
			ID:                 demoID("trx", p.key, i, name),
			ProjectID:          id(p.id),
			CostTypeID:         costTypes[name],
			TanggalTransaksi:   d,
			BulanRealisasi:     &bulan,
			DeskripsiRealisasi: strPtr(desc),
			JumlahRealisasi:    roundK(amount),
			JumlahTenagaKerja:  workers,
			CreatedBy:          DemoCreatedBy,
		})
	}

	for _, p := range demoProjects {
		for i := 0; i <= p.lastMonthIndex(); i++ {
			m := p.month(i)
			label := fmt.Sprintf("%s %d", bulanID[m.Month()], m.Year())
			wobble := float64((i*7+len(p.key))%5-2) * 0.005
			w := p.workers

			add(p, i, "Gaji & Tunjangan", 25, p.monthly*(0.72+wobble), &w, "Gaji dan tunjangan tenaga kerja "+label)
			add(p, i, "BPJS Ketenagakerjaan", 10, p.monthly*0.03, &w, "Iuran BPJS Ketenagakerjaan "+label)
			add(p, i, "BPJS Kesehatan", 10, p.monthly*0.04, &w, "Iuran BPJS Kesehatan "+label)

			if i%2 == 0 {
				add(p, i, "Transportasi Lapangan", 15, p.monthly*(0.035+wobble/2), nil, "Transportasi lapangan "+label)
			} else {
				add(p, i, "Akomodasi", 15, p.monthly*(0.02+wobble/2), nil, "Akomodasi petugas "+label)
			}
			switch {
			case m.Month() == time.March || m.Month() == time.December:
				add(p, i, "THR & Bonus", 20, p.monthly*0.05, &w, "THR dan bonus "+label)
			case i == 0:
				add(p, i, "Rekrutmen", 5, p.monthly*0.015, nil, "Rekrutmen awal kontrak")
				add(p, i, "Seragam & APD", 12, p.monthly*0.02, nil, "Seragam dan APD awal kontrak")
			case i%3 == 1:
				add(p, i, "Perlengkapan Kerja", 18, p.monthly*0.015, nil, "Perlengkapan kerja "+label)
			case i%3 == 2:
				add(p, i, "Pelatihan", 18, p.monthly*0.012, nil, "Pelatihan tenaga kerja "+label)
			}
		}
	}
	return upsertBatch(tx, rows, "transactions")
}

func upsertBatch[T any](tx *gorm.DB, rows []T, what string) error {
	if len(rows) == 0 {
		return nil
	}
	err := tx.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(rows, 100).Error
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	return nil
}

// opsMonthly is the monthly operational expense table: type ID, base amount,
// day of month. "Lainnya" is only booked in some months.
var opsMonthly = []struct {
	typeID string
	base   float64
	day    int
}{
	{"30000000-0000-4000-8000-000000000001", 15_000_000, 1}, // Sewa Kantor
	{"30000000-0000-4000-8000-000000000002", 3_200_000, 12}, // Listrik & Air
	{"30000000-0000-4000-8000-000000000003", 4_500_000, 15}, // Konsumsi
	{"30000000-0000-4000-8000-000000000004", 1_500_000, 8},  // ATK
	{"30000000-0000-4000-8000-000000000005", 2_800_000, 20}, // Transportasi
}

func demoSeedCostOperations(tx *gorm.DB) error {
	var rows []models.CostOperation
	add := func(typeID string, n int, month time.Time, day int, amount float64, desc string) {
		d := models.Date{Time: month.AddDate(0, 0, day-1)}
		bulan := helpers.ComputeBulanRealisasi(d)
		rows = append(rows, models.CostOperation{
			ID:                    demoID("ops", typeID, n),
			OperationalCostTypeID: id(typeID),
			TanggalTransaksi:      d,
			BulanRealisasi:        &bulan,
			Deskripsi:             strPtr(desc),
			Jumlah:                roundK(amount),
			CreatedBy:             DemoCreatedBy,
		})
	}

	names := map[string]string{
		"30000000-0000-4000-8000-000000000001": "Sewa kantor",
		"30000000-0000-4000-8000-000000000002": "Listrik dan air",
		"30000000-0000-4000-8000-000000000003": "Konsumsi rapat dan harian",
		"30000000-0000-4000-8000-000000000004": "Alat tulis kantor",
		"30000000-0000-4000-8000-000000000005": "Transportasi operasional",
	}
	// January 2025 .. October 2026
	for n := range 22 {
		month := ym(2025, 1).AddDate(0, n, 0)
		label := fmt.Sprintf("%s %d", bulanID[month.Month()], month.Year())
		wobble := float64((n*3)%5-2) * 0.02
		for _, o := range opsMonthly {
			factor := 1 + wobble
			if o.typeID == opsMonthly[0].typeID {
				factor = 1 // rent is fixed
			}
			add(o.typeID, n, month, o.day, o.base*factor, names[o.typeID]+" "+label)
		}
		if n%4 == 3 {
			add("30000000-0000-4000-8000-000000000006", n, month, 27, 1_200_000*(1+wobble), "Biaya operasional lain-lain "+label)
		}
	}
	return upsertBatch(tx, rows, "cost_operations")
}

func demoSeedInvoices(tx *gorm.DB) error {
	var rows []models.Invoice
	for _, p := range demoProjects {
		last := p.lastMonthIndex()

		// A SPR project is billed once for all its months (client cash advance).
		if p.method == models.PaymentMethodSPR {
			if last < p.months-1 {
				continue // period not finished yet
			}
			var periods models.MonthList
			for i := 0; i < p.months; i++ {
				periods = append(periods, p.month(i).Format("2006-01"))
			}
			billed := p.month(p.months-1).AddDate(0, 1, p.invoiceDay-1)
			rows = append(rows, demoInvoice(p, "SPR", periods, p.monthly*float64(p.months), billed,
				fmt.Sprintf("Tagihan SPR %s s/d %s", periodLabel(p.month(0)), periodLabel(p.month(p.months-1)))))
			continue
		}

		for i := 0; i <= last; i++ {
			billed := p.month(i).AddDate(0, 1, p.invoiceDay-1)
			if p.lateEveryNth > 0 && (i+1)%p.lateEveryNth == 0 {
				billed = billed.AddDate(0, 1, 0) // billed a month late
			}
			if billed.After(demoNow) {
				continue // not issued yet
			}
			rows = append(rows, demoInvoice(p, fmt.Sprintf("%02d", i+1), models.MonthList{p.month(i).Format("2006-01")},
				p.monthly, billed, "Tagihan "+p.judul+" periode "+periodLabel(p.month(i))))
		}
	}
	return upsertBatch(tx, rows, "invoices")
}

func periodLabel(m time.Time) string { return fmt.Sprintf("%s %d", bulanID[m.Month()], m.Year()) }

func demoInvoice(p demoProject, seq string, periods models.MonthList, nilai float64, billed time.Time, nama string) models.Invoice {
	inv := models.Invoice{
		ID:               demoID("inv", p.key, seq),
		NoInvoice:        fmt.Sprintf("%s%s-%s", DemoInvoicePrefix, p.key, seq),
		ProjectID:        id(p.id),
		NamaInvoice:      nama,
		TglTagihan:       models.Date{Time: billed},
		Status:           models.InvoiceStatusSubmit,
		MetodePembayaran: p.method,
		PeriodeTagihan:   periods,
		Nilai:            nilai,
		ManajemenFee:     roundK(nilai * p.feePct),
		CreatedBy:        DemoCreatedBy,
	}
	// Payment lag varies a little by period so the dates don't all look alike.
	lag := p.paymentLagDays + (len(seq)+int(billed.Month()))%7
	paid := billed.AddDate(0, 0, lag)
	if !paid.After(demoNow) {
		inv.Status = models.InvoiceStatusKliring
		inv.TglBayar = &models.Date{Time: paid}
	}
	return inv
}
