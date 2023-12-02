package canvas

import (
	"github.com/emurenMRz/ergen_go/cmd/pg_ergen/internal/db"
)

type column struct {
	nm string
	pt Point
}

type relationaly struct {
	schema string
	table  string
	column string
}

func (r *relationaly) fullname() string {
	return r.schema + "." + r.table + "." + r.column
}

func (r *relationaly) valid() bool {
	return len(r.schema) > 0 || len(r.table) > 0 || len(r.column) > 0
}

type row struct {
	frame *Rectangle

	order        int
	isPrimaryKey bool
	isNotNull    bool

	notNull      Rectangle
	physicalName column
	dataType     column
	logicalName  column

	relationaly relationaly
}

type rows []*row

func NewRow(c *db.Column) *row {
	fk := c.ForeignKey
	dt := c.DataType
	rel := relationaly{}
	if len(fk.ConstraintName) != 0 {
		dt += "(FK)"
		rel = relationaly{
			schema: fk.TableSchema,
			table:  fk.TableName,
			column: fk.ColumnName,
		}
	}

	nn := false
	if c.IsPrimaryKey || c.IsNullable == "YES" {
		nn = true
	}

	return &row{
		order:        c.OrdinalPosition,
		isPrimaryKey: c.IsPrimaryKey,
		isNotNull:    nn,

		physicalName: column{nm: c.ColumnName},
		dataType:     column{nm: dt},
		logicalName:  column{nm: c.Comment},

		relationaly: rel,
	}
}
