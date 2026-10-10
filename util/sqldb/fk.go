package sqldb

import (
	"strings"

	"gorm.io/gorm"
)

// AddForeignKey adds FK if DB supports it
func AddForeignKey(orm *gorm.DB, table, column, referenceTable, referenceColumn, onDelete, onUpdate string) (err error) {
	d, err := gDialects.get(orm.Dialector.Name())
	if err != nil {
		return err
	}
	if d.AddForeignKey == nil {
		return nil
	}

	return d.AddForeignKey(orm, foreignKeyOptions{
		IfNotExists: true,
		// simulate gorm name
		Name:            strings.ToLower(table + "_" + column + "_" + referenceTable + "_" + referenceColumn + "_foreign"),
		Table:           table,
		Column:          column,
		ReferenceTable:  referenceTable,
		ReferenceColumn: referenceColumn,
		OnDelete:        onDelete,
		OnUpdate:        onUpdate,
	})
}

type foreignKeyOptions struct {
	IfNotExists     bool
	Name            string
	Table           string
	Column          string
	ReferenceTable  string
	ReferenceColumn string
	OnUpdate        string
	OnDelete        string
}

func addForeignKeyStatementDefault(o foreignKeyOptions) string {
	return "ALTER TABLE " + o.Table +
		" ADD CONSTRAINT " + o.Name +
		" FOREIGN KEY (" + o.Column + ") REFERENCES " + o.ReferenceTable + "(" + o.ReferenceColumn + ")" +
		" ON UPDATE " + o.OnUpdate + " ON DELETE " + o.OnDelete
}
