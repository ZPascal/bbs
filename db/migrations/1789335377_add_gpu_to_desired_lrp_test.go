package migrations_test

import (
	"time"

	"code.cloudfoundry.org/bbs/db/migrations"
	"code.cloudfoundry.org/bbs/migration"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/diego-db-helpers/sqldb/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Add GPU to Desired LRPs", func() {
	var (
		mig migration.Migration
	)

	BeforeEach(func() {
		fakeClock = fakeclock.NewFakeClock(time.Now())
		rawSQLDB.Exec("DROP TABLE domains;")
		rawSQLDB.Exec("DROP TABLE tasks;")
		rawSQLDB.Exec("DROP TABLE desired_lrps;")
		rawSQLDB.Exec("DROP TABLE actual_lrps;")

		mig = migrations.NewAddGPUToDesiredLRPs()
	})

	It("appends itself to the migration list", func() {
		Expect(migrations.AllMigrations()).To(ContainElement(mig))
	})

	Describe("Version", func() {
		It("returns the timestamp from which it was created", func() {
			Expect(mig.Version()).To(BeEquivalentTo(1789335377))
		})
	})

	Describe("Up", func() {
		BeforeEach(func() {
			initialMigrations := []migration.Migration{
				migrations.NewInitSQL(),
				migrations.NewIncreaseRunInfoColumnSize(),
			}

			for _, m := range initialMigrations {
				m.SetDBFlavor(flavor)
				m.SetClock(fakeClock)
				testUpInTransaction(rawSQLDB, m, logger)
			}

			mig.SetDBFlavor(flavor)
			mig.SetClock(fakeClock)
		})

		It("defaults gpu_limit and gpu_type on new desired lrps inserted before the migration ran", func() {
			_, err := rawSQLDB.Exec(
				helpers.RebindForFlavor(
					`INSERT INTO desired_lrps
						  (process_guid, domain, log_guid, instances, memory_mb,
							  disk_mb, rootfs, routes, volume_placement, modification_tag_epoch, run_info)
						  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					flavor,
				),
				"meow-guid", "domain",
				"log guid", 2, 1, 1, "rootfs", "routes", "volumes yo", "1", "run info",
			)
			Expect(err).NotTo(HaveOccurred())

			testUpInTransaction(rawSQLDB, mig, logger)

			var fetchedGPULimit int
			var fetchedGPUType string
			query := helpers.RebindForFlavor("select gpu_limit, gpu_type from desired_lrps where process_guid = 'meow-guid' limit 1", flavor)
			row := rawSQLDB.QueryRow(query)
			Expect(row.Scan(&fetchedGPULimit, &fetchedGPUType)).NotTo(HaveOccurred())
			Expect(fetchedGPULimit).To(Equal(0))
			Expect(fetchedGPUType).To(Equal(""))
		})

		It("allows inserting a desired lrp with explicit gpu_limit and gpu_type after the migration ran", func() {
			testUpInTransaction(rawSQLDB, mig, logger)

			_, err := rawSQLDB.Exec(
				helpers.RebindForFlavor(
					`INSERT INTO desired_lrps
						  (process_guid, domain, log_guid, instances, memory_mb,
							  disk_mb, rootfs, routes, volume_placement, modification_tag_epoch, run_info,
							  gpu_limit, gpu_type)
						  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					flavor,
				),
				"gpu-guid", "domain",
				"log guid", 2, 1, 1, "rootfs", "routes", "volumes yo", "1", "run info",
				1, "nvidia",
			)
			Expect(err).NotTo(HaveOccurred())

			var fetchedGPULimit int
			var fetchedGPUType string
			query := helpers.RebindForFlavor("select gpu_limit, gpu_type from desired_lrps where process_guid = 'gpu-guid' limit 1", flavor)
			row := rawSQLDB.QueryRow(query)
			Expect(row.Scan(&fetchedGPULimit, &fetchedGPUType)).NotTo(HaveOccurred())
			Expect(fetchedGPULimit).To(Equal(1))
			Expect(fetchedGPUType).To(Equal("nvidia"))
		})

		It("is idempotent", func() {
			testIdempotency(rawSQLDB, mig, logger)
		})
	})
})
