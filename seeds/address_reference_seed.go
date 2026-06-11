package seeds

import (
	"fmt"
	"boilerplates-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AddressReferenceSeeder seeds countries, provinces, cities, and areas.
// Safe to re-run: uses FirstOrCreate so existing records are skipped.
func AddressReferenceSeeder(db *gorm.DB) error {
	fmt.Println("📝 Seeding address reference tables...")

	if err := seedCountries(db); err != nil {
		return err
	}
	if err := seedProvinces(db); err != nil {
		return err
	}
	if err := seedCities(db); err != nil {
		return err
	}
	if err := seedAreas(db); err != nil {
		return err
	}

	fmt.Println("✓ Address reference seeding completed")
	return nil
}

func id(s string) uuid.UUID { return uuid.MustParse(s) }

func seedCountries(db *gorm.DB) error {
	countries := []models.Country{
		{ID: id("10000000-0000-0000-0000-000000000001"), Name: "Indonesia", Code: "ID"},
		{ID: id("10000000-0000-0000-0000-000000000002"), Name: "United States", Code: "US"},
	}
	for _, c := range countries {
		if err := db.Where("code = ?", c.Code).FirstOrCreate(&c).Error; err != nil {
			return fmt.Errorf("country %s: %w", c.Code, err)
		}
	}
	fmt.Printf("  ✓ Countries: %d records\n", len(countries))
	return nil
}

func seedProvinces(db *gorm.DB) error {
	idn := id("10000000-0000-0000-0000-000000000001")
	provinces := []models.Province{
		{ID: id("20000000-0000-0000-0000-000000000001"), Name: "DKI Jakarta", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000002"), Name: "Jawa Barat", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000003"), Name: "Jawa Tengah", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000004"), Name: "Jawa Timur", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000005"), Name: "Banten", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000006"), Name: "Bali", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000007"), Name: "Sumatera Utara", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000008"), Name: "Sumatera Barat", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000009"), Name: "Sumatera Selatan", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000010"), Name: "Kalimantan Barat", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000011"), Name: "Kalimantan Selatan", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000012"), Name: "Kalimantan Timur", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000013"), Name: "Sulawesi Selatan", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000014"), Name: "Sulawesi Utara", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000015"), Name: "Lampung", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000016"), Name: "Aceh", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000017"), Name: "Riau", CountryID: idn},
		{ID: id("20000000-0000-0000-0000-000000000018"), Name: "Papua", CountryID: idn},
	}
	for _, p := range provinces {
		if err := db.Where("id = ?", p.ID).FirstOrCreate(&p).Error; err != nil {
			return fmt.Errorf("province %s: %w", p.Name, err)
		}
	}
	fmt.Printf("  ✓ Provinces: %d records\n", len(provinces))
	return nil
}

func seedCities(db *gorm.DB) error {
	p := func(n int) uuid.UUID {
		return id(fmt.Sprintf("20000000-0000-0000-0000-%012d", n))
	}
	cities := []models.City{
		// DKI Jakarta
		{ID: id("30000000-0000-0000-0000-000000000001"), Name: "Kota Jakarta Pusat", ProvinceID: p(1)},
		{ID: id("30000000-0000-0000-0000-000000000002"), Name: "Kota Jakarta Selatan", ProvinceID: p(1)},
		{ID: id("30000000-0000-0000-0000-000000000003"), Name: "Kota Jakarta Barat", ProvinceID: p(1)},
		{ID: id("30000000-0000-0000-0000-000000000004"), Name: "Kota Jakarta Timur", ProvinceID: p(1)},
		{ID: id("30000000-0000-0000-0000-000000000005"), Name: "Kota Jakarta Utara", ProvinceID: p(1)},
		// Jawa Barat
		{ID: id("30000000-0000-0000-0000-000000000010"), Name: "Kota Bandung", ProvinceID: p(2)},
		{ID: id("30000000-0000-0000-0000-000000000011"), Name: "Kota Bekasi", ProvinceID: p(2)},
		{ID: id("30000000-0000-0000-0000-000000000012"), Name: "Kota Depok", ProvinceID: p(2)},
		{ID: id("30000000-0000-0000-0000-000000000013"), Name: "Kota Bogor", ProvinceID: p(2)},
		{ID: id("30000000-0000-0000-0000-000000000014"), Name: "Kabupaten Cianjur", ProvinceID: p(2)},
		{ID: id("30000000-0000-0000-0000-000000000015"), Name: "Kabupaten Purwakarta", ProvinceID: p(2)},
		{ID: id("30000000-0000-0000-0000-000000000016"), Name: "Kabupaten Subang", ProvinceID: p(2)},
		// Jawa Tengah
		{ID: id("30000000-0000-0000-0000-000000000020"), Name: "Kota Semarang", ProvinceID: p(3)},
		{ID: id("30000000-0000-0000-0000-000000000021"), Name: "Kota Solo", ProvinceID: p(3)},
		{ID: id("30000000-0000-0000-0000-000000000022"), Name: "Kota Yogyakarta", ProvinceID: p(3)},
		// Jawa Timur
		{ID: id("30000000-0000-0000-0000-000000000030"), Name: "Kota Surabaya", ProvinceID: p(4)},
		{ID: id("30000000-0000-0000-0000-000000000031"), Name: "Kota Malang", ProvinceID: p(4)},
		{ID: id("30000000-0000-0000-0000-000000000032"), Name: "Kota Kediri", ProvinceID: p(4)},
		// Banten
		{ID: id("30000000-0000-0000-0000-000000000040"), Name: "Kota Tangerang", ProvinceID: p(5)},
		{ID: id("30000000-0000-0000-0000-000000000041"), Name: "Kota Serang", ProvinceID: p(5)},
		{ID: id("30000000-0000-0000-0000-000000000042"), Name: "Kota Cilegon", ProvinceID: p(5)},
		// Bali
		{ID: id("30000000-0000-0000-0000-000000000050"), Name: "Kota Denpasar", ProvinceID: p(6)},
		{ID: id("30000000-0000-0000-0000-000000000051"), Name: "Kabupaten Badung", ProvinceID: p(6)},
		{ID: id("30000000-0000-0000-0000-000000000052"), Name: "Kabupaten Gianyar", ProvinceID: p(6)},
		// Sumatera Utara
		{ID: id("30000000-0000-0000-0000-000000000060"), Name: "Kota Medan", ProvinceID: p(7)},
		{ID: id("30000000-0000-0000-0000-000000000061"), Name: "Kota Binjai", ProvinceID: p(7)},
		// Sumatera Barat
		{ID: id("30000000-0000-0000-0000-000000000070"), Name: "Kota Padang", ProvinceID: p(8)},
		{ID: id("30000000-0000-0000-0000-000000000071"), Name: "Kota Bukittinggi", ProvinceID: p(8)},
		// Sumatera Selatan
		{ID: id("30000000-0000-0000-0000-000000000080"), Name: "Kota Palembang", ProvinceID: p(9)},
		{ID: id("30000000-0000-0000-0000-000000000081"), Name: "Kota Prabumulih", ProvinceID: p(9)},
		// Kalimantan Barat
		{ID: id("30000000-0000-0000-0000-000000000090"), Name: "Kota Pontianak", ProvinceID: p(10)},
		{ID: id("30000000-0000-0000-0000-000000000091"), Name: "Kota Singkawang", ProvinceID: p(10)},
		// Kalimantan Selatan
		{ID: id("30000000-0000-0000-0000-000000000100"), Name: "Kota Banjarmasin", ProvinceID: p(11)},
		{ID: id("30000000-0000-0000-0000-000000000101"), Name: "Kota Banjarbaru", ProvinceID: p(11)},
		// Kalimantan Timur
		{ID: id("30000000-0000-0000-0000-000000000110"), Name: "Kota Samarinda", ProvinceID: p(12)},
		{ID: id("30000000-0000-0000-0000-000000000111"), Name: "Kota Balikpapan", ProvinceID: p(12)},
		// Sulawesi Selatan
		{ID: id("30000000-0000-0000-0000-000000000120"), Name: "Kota Makassar", ProvinceID: p(13)},
		{ID: id("30000000-0000-0000-0000-000000000121"), Name: "Kota Parepare", ProvinceID: p(13)},
		// Sulawesi Utara
		{ID: id("30000000-0000-0000-0000-000000000130"), Name: "Kota Manado", ProvinceID: p(14)},
		{ID: id("30000000-0000-0000-0000-000000000131"), Name: "Kota Bitung", ProvinceID: p(14)},
		// Lampung
		{ID: id("30000000-0000-0000-0000-000000000140"), Name: "Kota Bandar Lampung", ProvinceID: p(15)},
		{ID: id("30000000-0000-0000-0000-000000000141"), Name: "Kota Metro", ProvinceID: p(15)},
		// Aceh
		{ID: id("30000000-0000-0000-0000-000000000150"), Name: "Kota Banda Aceh", ProvinceID: p(16)},
		{ID: id("30000000-0000-0000-0000-000000000151"), Name: "Kota Lhokseumawe", ProvinceID: p(16)},
		// Riau
		{ID: id("30000000-0000-0000-0000-000000000160"), Name: "Kota Pekanbaru", ProvinceID: p(17)},
		{ID: id("30000000-0000-0000-0000-000000000161"), Name: "Kota Dumai", ProvinceID: p(17)},
		// Papua
		{ID: id("30000000-0000-0000-0000-000000000170"), Name: "Kota Jayapura", ProvinceID: p(18)},
	}
	for _, c := range cities {
		if err := db.Where("id = ?", c.ID).FirstOrCreate(&c).Error; err != nil {
			return fmt.Errorf("city %s: %w", c.Name, err)
		}
	}
	fmt.Printf("  ✓ Cities: %d records\n", len(cities))
	return nil
}

func seedAreas(db *gorm.DB) error {
	c := func(n int) uuid.UUID {
		return id(fmt.Sprintf("30000000-0000-0000-0000-%012d", n))
	}
	areas := []models.Area{
		// Jakarta Pusat
		{ID: id("40000000-0000-0000-0000-000000000001"), Name: "Gambir", CityID: c(1)},
		{ID: id("40000000-0000-0000-0000-000000000002"), Name: "Menteng", CityID: c(1)},
		{ID: id("40000000-0000-0000-0000-000000000003"), Name: "Senen", CityID: c(1)},
		{ID: id("40000000-0000-0000-0000-000000000004"), Name: "Tanah Abang", CityID: c(1)},
		// Jakarta Selatan
		{ID: id("40000000-0000-0000-0000-000000000010"), Name: "Kebayoran Baru", CityID: c(2)},
		{ID: id("40000000-0000-0000-0000-000000000011"), Name: "Tebet", CityID: c(2)},
		{ID: id("40000000-0000-0000-0000-000000000012"), Name: "Mampang Prapatan", CityID: c(2)},
		{ID: id("40000000-0000-0000-0000-000000000013"), Name: "Pasar Minggu", CityID: c(2)},
		// Jakarta Barat
		{ID: id("40000000-0000-0000-0000-000000000020"), Name: "Grogol Petamburan", CityID: c(3)},
		{ID: id("40000000-0000-0000-0000-000000000021"), Name: "Tambora", CityID: c(3)},
		// Jakarta Timur
		{ID: id("40000000-0000-0000-0000-000000000030"), Name: "Jatinegara", CityID: c(4)},
		{ID: id("40000000-0000-0000-0000-000000000031"), Name: "Cakung", CityID: c(4)},
		// Jakarta Utara
		{ID: id("40000000-0000-0000-0000-000000000040"), Name: "Penjaringan", CityID: c(5)},
		{ID: id("40000000-0000-0000-0000-000000000041"), Name: "Tanjung Priok", CityID: c(5)},
		// Bandung
		{ID: id("40000000-0000-0000-0000-000000000050"), Name: "Coblong", CityID: c(10)},
		{ID: id("40000000-0000-0000-0000-000000000051"), Name: "Cicendo", CityID: c(10)},
		{ID: id("40000000-0000-0000-0000-000000000052"), Name: "Bandung Wetan", CityID: c(10)},
		// Bekasi
		{ID: id("40000000-0000-0000-0000-000000000060"), Name: "Bekasi Timur", CityID: c(11)},
		{ID: id("40000000-0000-0000-0000-000000000061"), Name: "Bekasi Barat", CityID: c(11)},
		// Depok
		{ID: id("40000000-0000-0000-0000-000000000070"), Name: "Beji", CityID: c(12)},
		{ID: id("40000000-0000-0000-0000-000000000071"), Name: "Cimanggis", CityID: c(12)},
		// Bogor
		{ID: id("40000000-0000-0000-0000-000000000080"), Name: "Bogor Tengah", CityID: c(13)},
		{ID: id("40000000-0000-0000-0000-000000000081"), Name: "Bogor Selatan", CityID: c(13)},
		// Semarang
		{ID: id("40000000-0000-0000-0000-000000000090"), Name: "Semarang Tengah", CityID: c(20)},
		{ID: id("40000000-0000-0000-0000-000000000091"), Name: "Semarang Selatan", CityID: c(20)},
		// Solo
		{ID: id("40000000-0000-0000-0000-000000000100"), Name: "Laweyan", CityID: c(21)},
		{ID: id("40000000-0000-0000-0000-000000000101"), Name: "Serengan", CityID: c(21)},
		// Surabaya
		{ID: id("40000000-0000-0000-0000-000000000110"), Name: "Gubeng", CityID: c(30)},
		{ID: id("40000000-0000-0000-0000-000000000111"), Name: "Genteng", CityID: c(30)},
		{ID: id("40000000-0000-0000-0000-000000000112"), Name: "Wonokromo", CityID: c(30)},
		// Malang
		{ID: id("40000000-0000-0000-0000-000000000120"), Name: "Klojen", CityID: c(31)},
		{ID: id("40000000-0000-0000-0000-000000000121"), Name: "Lowokwaru", CityID: c(31)},
		// Tangerang
		{ID: id("40000000-0000-0000-0000-000000000130"), Name: "Tangerang", CityID: c(40)},
		{ID: id("40000000-0000-0000-0000-000000000131"), Name: "Ciledug", CityID: c(40)},
		// Serang
		{ID: id("40000000-0000-0000-0000-000000000140"), Name: "Serang", CityID: c(41)},
		{ID: id("40000000-0000-0000-0000-000000000141"), Name: "Cipocok Jaya", CityID: c(41)},
		// Denpasar
		{ID: id("40000000-0000-0000-0000-000000000150"), Name: "Denpasar Selatan", CityID: c(50)},
		{ID: id("40000000-0000-0000-0000-000000000151"), Name: "Denpasar Barat", CityID: c(50)},
		// Badung
		{ID: id("40000000-0000-0000-0000-000000000160"), Name: "Kuta", CityID: c(51)},
		{ID: id("40000000-0000-0000-0000-000000000161"), Name: "Kuta Selatan", CityID: c(51)},
		// Medan
		{ID: id("40000000-0000-0000-0000-000000000170"), Name: "Medan Kota", CityID: c(60)},
		{ID: id("40000000-0000-0000-0000-000000000171"), Name: "Medan Baru", CityID: c(60)},
		// Makassar
		{ID: id("40000000-0000-0000-0000-000000000180"), Name: "Makassar", CityID: c(120)},
		{ID: id("40000000-0000-0000-0000-000000000181"), Name: "Rappocini", CityID: c(120)},
	}
	for _, a := range areas {
		if err := db.Where("id = ?", a.ID).FirstOrCreate(&a).Error; err != nil {
			return fmt.Errorf("area %s: %w", a.Name, err)
		}
	}
	fmt.Printf("  ✓ Areas: %d records\n", len(areas))
	return nil
}
