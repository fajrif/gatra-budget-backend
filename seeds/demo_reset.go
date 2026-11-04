package seeds

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"gorm.io/gorm"
)

// DemoReset interactively removes the rows written by DemoSeeder. For each kind
// of data it asks "[y/N]" (empty answer or end of input means no). With
// assumeYes every question is answered y. Only demo rows are touched: rows
// created by hand (or by anyone other than DemoSeeder) are always kept.
func DemoReset(db *gorm.DB, in io.Reader, out io.Writer, assumeYes bool) error {
	reader := bufio.NewReader(in)

	var dbName string
	db.Raw("SELECT current_database()").Scan(&dbName)
	fmt.Fprintf(out, "Database tujuan: %s\n", dbName)
	fmt.Fprintln(out, "Hanya data demo (hasil ./seed.sh demo) yang dihapus; data yang diinput manual tidak disentuh.")
	fmt.Fprintln(out)

	ask := func(question string) bool {
		if assumeYes {
			fmt.Fprintf(out, "%s [y/N] y\n", question)
			return true
		}
		fmt.Fprintf(out, "%s [y/N] ", question)
		line, _ := reader.ReadString('\n')
		answer := strings.ToLower(strings.TrimSpace(line))
		return answer == "y" || answer == "ya" || answer == "yes"
	}

	const demoProjects = "SELECT id FROM projects WHERE created_by = '" + DemoCreatedBy + "'"

	del := func(label, sql string, args ...any) error {
		res := db.Exec(sql, args...)
		if res.Error != nil {
			return fmt.Errorf("%s: %w", label, res.Error)
		}
		fmt.Fprintf(out, "  → %d baris %s dihapus\n", res.RowsAffected, label)
		return nil
	}

	if ask(`Apakah anda ingin kosongkan data "Invoice" demo?`) {
		if err := del("invoice", "DELETE FROM invoices WHERE created_by = ?", DemoCreatedBy); err != nil {
			return err
		}
	}
	if ask(`Apakah anda ingin kosongkan data "Transaksi" (proyek) demo?`) {
		if err := del("transaksi", "DELETE FROM transactions WHERE created_by = ?", DemoCreatedBy); err != nil {
			return err
		}
	}
	if ask(`Apakah anda ingin kosongkan data "Transaksi Beban Operasional" demo?`) {
		if err := del("beban operasional", "DELETE FROM cost_operations WHERE created_by = ?", DemoCreatedBy); err != nil {
			return err
		}
	}
	if ask(`Apakah anda ingin kosongkan data "Proyek" demo? (invoice dan transaksi demo milik proyek itu ikut dihapus)`) {
		if err := del("invoice demo milik proyek demo", "DELETE FROM invoices WHERE created_by = ? AND project_id IN ("+demoProjects+")", DemoCreatedBy); err != nil {
			return err
		}
		if err := del("transaksi demo milik proyek demo", "DELETE FROM transactions WHERE created_by = ? AND project_id IN ("+demoProjects+")", DemoCreatedBy); err != nil {
			return err
		}
		// A demo project that someone later attached real invoices/transactions
		// to is kept: deleting it would destroy that data.
		if err := del("proyek",
			`DELETE FROM projects p WHERE p.created_by = ?
			   AND NOT EXISTS (SELECT 1 FROM invoices i WHERE i.project_id = p.id)
			   AND NOT EXISTS (SELECT 1 FROM transactions t WHERE t.project_id = p.id)`, DemoCreatedBy); err != nil {
			return err
		}
		var kept int64
		db.Raw("SELECT COUNT(*) FROM projects WHERE created_by = ?", DemoCreatedBy).Scan(&kept)
		if kept > 0 {
			fmt.Fprintf(out, "  ! %d proyek demo dipertahankan karena sudah punya invoice/transaksi non-demo\n", kept)
		}
	}
	if ask(`Apakah anda ingin kosongkan data "Group Proyek" demo? (hanya group yang sudah tidak punya proyek)`) {
		if err := del("group proyek",
			`DELETE FROM project_groups g WHERE g.id::text LIKE ?
			   AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.project_group_id = g.id)`, DemoIDPrefix+"%"); err != nil {
			return err
		}
		var kept int64
		db.Raw("SELECT COUNT(*) FROM project_groups WHERE id::text LIKE ?", DemoIDPrefix+"%").Scan(&kept)
		if kept > 0 {
			fmt.Fprintf(out, "  ! %d group demo dipertahankan karena masih punya proyek\n", kept)
		}
	}
	if ask(`Apakah anda ingin kosongkan "Jenis Biaya" tambahan dari demo? (hanya yang tidak dipakai transaksi)`) {
		if err := del("jenis biaya",
			`DELETE FROM cost_types c WHERE c.id::text LIKE ?
			   AND NOT EXISTS (SELECT 1 FROM transactions t WHERE t.cost_type_id = c.id)`, DemoIDPrefix+"%"); err != nil {
			return err
		}
	}

	fmt.Fprintln(out, "\nSelesai.")
	return nil
}
