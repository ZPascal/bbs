package migrations

import (
	"database/sql"

	"code.cloudfoundry.org/bbs/encryption"
	"code.cloudfoundry.org/bbs/format"
	"code.cloudfoundry.org/bbs/migration"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

func init() {
	appendMigration(NewAddGPUToDesiredLRPs())
}

type AddGPUToDesiredLRPs struct {
	serializer format.Serializer
	clock      clock.Clock
	dbFlavor   string
}

func NewAddGPUToDesiredLRPs() migration.Migration {
	return &AddGPUToDesiredLRPs{}
}

func (e *AddGPUToDesiredLRPs) String() string {
	return migrationString(e)
}

func (e *AddGPUToDesiredLRPs) Version() int64 {
	return 1789335377
}

func (e *AddGPUToDesiredLRPs) SetCryptor(cryptor encryption.Cryptor) {
	e.serializer = format.NewSerializer(cryptor)
}

func (e *AddGPUToDesiredLRPs) SetClock(c clock.Clock)    { e.clock = c }
func (e *AddGPUToDesiredLRPs) SetDBFlavor(flavor string) { e.dbFlavor = flavor }

func (e *AddGPUToDesiredLRPs) Up(tx *sql.Tx, logger lager.Logger) error {
	var alterDesiredLRPAddGPUSQL string
	if e.dbFlavor == "mysql" {
		alterDesiredLRPAddGPUSQL = `ALTER TABLE desired_lrps
	ADD COLUMN gpu_limit INT NOT NULL DEFAULT 0,
	ADD COLUMN gpu_type VARCHAR(255) NOT NULL DEFAULT '';`
	} else {
		alterDesiredLRPAddGPUSQL = `ALTER TABLE desired_lrps
	ADD COLUMN IF NOT EXISTS gpu_limit INT NOT NULL DEFAULT 0,
	ADD COLUMN IF NOT EXISTS gpu_type VARCHAR(255) NOT NULL DEFAULT '';`
	}
	logger.Info("altering the table", lager.Data{"query": alterDesiredLRPAddGPUSQL})
	_, err := tx.Exec(alterDesiredLRPAddGPUSQL)
	if err != nil && !isDuplicateColumnError(err) {
		logger.Error("failed-altering-tables", err)
		return err
	}
	logger.Info("altered the table", lager.Data{"query": alterDesiredLRPAddGPUSQL})

	return nil
}
