package db

import (
	"strconv"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DBConnect struct {
	Host     string
	User     string
	Password string
	Dbname   string
	Port     uint16
	Sslmode  bool
	TimeZone string

	db *gorm.DB
}

func (c *DBConnect) Connect() (self *DBConnect, err error) {
	if len(c.Host) == 0 {
		c.Host = "localhost"
	}
	if len(c.User) == 0 {
		c.User = "postgres"
	}
	if len(c.Dbname) == 0 {
		c.Dbname = "postgres"
	}
	if c.Port == 0 {
		c.Port = 5432
	}
	sslmode := "disable"
	if c.Sslmode {
		sslmode = "enable"
	}

	params := []string{
		"host=" + c.Host,
		"user=" + c.User,
		"dbname=" + c.Dbname,
		"port=" + strconv.Itoa(int(c.Port)),
		"sslmode=" + sslmode,
	}
	if len(c.Password) != 0 {
		params = append(params, "password="+c.Password)
	}
	if len(c.TimeZone) != 0 {
		params = append(params, "TimeZone="+c.TimeZone)
	}

	dsn := strings.Join(params, " ")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	c.db = db
	self = c

	return
}

func (c *DBConnect) Databasenames() (names []string, err error) {
	rows, err := c.db.Table("pg_database").Where("datistemplate = ?", false).Select("datname").Rows()
	if err != nil {
		return
	}

	defer rows.Close()

	names = []string{}
	for rows.Next() {
		var name string
		rows.Scan(&name)
		names = append(names, name)
	}

	return
}

func (c *DBConnect) Tablenames() (names []string, err error) {
	rows, err := c.db.Table("information_schema.tables").Where("table_schema = ?", "public").Select("table_name").Rows()
	if err != nil {
		return
	}

	defer rows.Close()

	names = []string{}
	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)
		names = append(names, tableName)
	}

	return
}

type ForeignKey struct {
	ConstraintName string
	TableSchema    string
	TableName      string
	ColumnName     string
	MatchOption    string
	UpdateRule     string
	DeleteRule     string
}

type Column struct {
	TableCatalog           string
	TableSchema            string
	TableName              string
	ColumnName             string
	OrdinalPosition        int
	ColumnDefault          string
	IsNullable             string
	DataType               string
	CharacterMaximumLength string
	CharacterOctetLength   string
	NumericPrecision       string
	NumericPrecisionRadix  string
	NumericScale           string
	DatetimePrecision      string
	IntervalType           string
	IntervalPrecision      string
	CharacterSetCatalog    string
	CharacterSetSchema     string
	CharacterSetName       string
	CollationCatalog       string
	CollationSchema        string
	CollationName          string
	DomainCatalog          string
	DomainSchema           string
	DomainName             string
	UdtCatalog             string
	UdtSchema              string
	UdtName                string
	ScopeCatalog           string
	ScopeSchema            string
	ScopeName              string
	MaximumCardinality     string
	DtdIdentifier          string
	IsSelfReferencing      string
	IsIdentity             string
	IdentityGeneration     string
	IdentityStart          string
	IdentityIncrement      string
	IdentityMaximum        string
	IdentityMinimum        string
	IdentityCycle          string
	IsGenerated            string
	GenerationExpression   string
	IsUpdatable            string

	IsPrimaryKey    bool
	IsUnique        bool
	Comment         string
	AlternativeName string
	ForeignKey
}
type Columns map[string]*Column
type OrdinalColumns map[int]*Column

func (c *DBConnect) columnInfo(n string) (columns Columns, index_for_columns OrdinalColumns, err error) {
	rows, err := c.db.Table("information_schema.columns").Where("table_name = ?", n).Order("table_name, ordinal_position").Select("*").Rows()
	if err != nil {
		return
	}

	defer rows.Close()

	columns = Columns{}
	index_for_columns = OrdinalColumns{}
	for rows.Next() {
		var column Column
		c.db.ScanRows(rows, &column)
		columns[column.ColumnName] = &column
		index_for_columns[column.OrdinalPosition] = &column
	}

	return
}

type constraint struct {
	ConstraintName    string
	ConstraintCatalog string
	ConstraintSchema  string
	ConstraintType    string
	IsDeferrable      string
	InitiallyDeferred string
	Enforced          string
	TableSchema       string
	TableName         string
	ColumnName        string
	TargetTableSchema string
	TargetTableName   string
	TargetColumnName  string
}

func (c *DBConnect) constraint(n string, col *Columns) (err error) {
	sql := `
	SELECT
		A.constraint_name,
		A.constraint_catalog,
		A.constraint_schema,
		A.constraint_type,
		A.is_deferrable,
		A.initially_deferred,
		A.enforced,
		B.table_schema,
		B.table_name,
		B.column_name,
		C.table_schema target_table_schema,
		C.table_name target_table_name,
		C.column_name target_column_name
	FROM
		(
			SELECT * FROM information_schema.table_constraints
			WHERE
				constraint_type <> 'CHECK'
				AND constraint_schema = ?
				AND table_name = ?
		) A
		INNER JOIN (
			SELECT * FROM information_schema.key_column_usage ) B
			USING (
				constraint_name,
				constraint_catalog,
				constraint_schema,
				table_catalog
			)
		INNER JOIN (
			SELECT * FROM information_schema.constraint_column_usage ) C
			USING (
				constraint_name,
				constraint_catalog,
				constraint_schema,
				table_catalog
			)
	`
	rows, err := c.db.Raw(sql, "public", n).Rows()
	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		var constraint constraint
		c.db.ScanRows(rows, &constraint)
		if constraint.ConstraintType == "PRIMARY KEY" {
			(*col)[constraint.ColumnName].IsPrimaryKey = true
		} else if constraint.ConstraintType == "FOREIGN KEY" {
			(*col)[constraint.ColumnName].ForeignKey = ForeignKey{
				ConstraintName: constraint.ConstraintName,
				TableSchema:    constraint.TargetTableSchema,
				TableName:      constraint.TargetTableName,
				ColumnName:     constraint.TargetColumnName,
			}
		} else if constraint.ConstraintType == "UNIQUE" {
			(*col)[constraint.ColumnName].IsUnique = true
		}
	}

	return
}

type referentialConstraint struct {
	ConstraintName          string
	ConstraintCatalog       string
	ConstraintSchema        string
	UniqueConstraintCatalog string
	UniqueConstraintSchema  string
	UniqueConstraintName    string
	MatchOption             string
	UpdateRule              string
	DeleteRule              string
}

func (c *DBConnect) referentialConstraints(col *Columns) (err error) {
	rows, err := c.db.Table("information_schema.referential_constraints").Where("constraint_schema = ?", "public").Select("*").Rows()
	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		var rc referentialConstraint
		c.db.ScanRows(rows, &rc)
		for _, c := range *col {
			if rc.ConstraintName == c.ForeignKey.ConstraintName {
				fk := c.ForeignKey
				fk.MatchOption = rc.MatchOption
				fk.UpdateRule = rc.UpdateRule
				fk.DeleteRule = rc.DeleteRule
				c.ForeignKey = fk
			}
		}
	}

	return
}

func (c *DBConnect) comment(n string, ifc *OrdinalColumns) (table_comment string, err error) {
	sql := `
	SELECT
	    objsubid,
	    description
	FROM
	    pg_stat_user_tables A
	    INNER JOIN pg_description B
	    ON A.relid = B.objoid
	WHERE
	    relname = ?
	ORDER BY
	    B.objsubid
	`
	rows, err := c.db.Raw(sql, n).Rows()
	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		var objsubid int
		var description string
		rows.Scan(&objsubid, &description)
		if objsubid == 0 {
			table_comment = description
		} else {
			column := (*ifc)[objsubid]
			column.Comment = description
			(*ifc)[objsubid] = column
		}
	}

	return
}

type TableInfo struct {
	Schema          string
	Name            string
	Columns         Columns
	Comment         string
	AlternativeName string
}

func (c *DBConnect) GetTableInfo(n string) (info TableInfo, err error) {
	columns, index_for_columns, err := c.columnInfo(n)
	if err != nil {
		return
	}

	err = c.constraint(n, &columns)
	if err != nil {
		return
	}

	err = c.referentialConstraints(&columns)
	if err != nil {
		return
	}

	table_comment, err := c.comment(n, &index_for_columns)
	if err != nil {
		return
	}

	info.Schema = "public"
	info.Name = n
	info.Columns = columns
	info.Comment = table_comment
	info.AlternativeName = ""

	return
}
