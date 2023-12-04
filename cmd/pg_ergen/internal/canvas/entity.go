package canvas

import (
	"fmt"
	"sort"

	svg "github.com/ajstarks/svgo"
	"github.com/emurenMRz/ergen_go/cmd/pg_ergen/internal/db"
)

type Entity struct {
	schema  string
	name    string
	comment string
	rows    rows
	pkeys   []int
	field   []int

	margin int
	width  int
	height int
	radius int

	hasForeignKey bool
	isChildren    bool
	title         string
	view          Rectangle
	tiltePos      Point
	frame         Rectangle
	separateLine  TwoPointCoordinates
	collision     map[string]*Rectangle

	lineStyle string
	font      string
	typeFont  string
}

func NewEntity(schema string, name string, comment string) *Entity {
	height := 16

	title := name
	if len(comment) > 0 {
		title = fmt.Sprintf("%s (%s)", comment, name)
	}

	return &Entity{
		schema:  schema,
		name:    name,
		comment: comment,

		margin: 2,
		width:  8,
		height: height,
		radius: height >> 2,

		title:     title,
		collision: map[string]*Rectangle{},

		lineStyle: StyleMap{
			"fill":   "none",
			"stroke": "black",
		}.String(),
		font: StyleMap{
			"fill":        "black",
			"stroke":      "none",
			"font-family": "monospace",
			"font-size":   fmt.Sprintf("%dpx", height),
		}.String(),
		typeFont: `fill="#6b3400"`,
	}
}

func NewEntityFromTableInfo(ti *db.TableInfo) *Entity {
	e := NewEntity(ti.Schema, ti.Name, ti.Comment)

	for _, col := range ti.Columns {
		fk := col.ForeignKey
		if fk.UpdateRule == "CASCADE" || fk.DeleteRule == "CASCADE" {
			e.isChildren = true
		}
		e.rows = append(e.rows, NewRow(col))
	}
	e.Build()

	return e
}

func (e *Entity) Build() {
	sort.Slice(e.rows, func(i, j int) bool {
		return e.rows[i].order < e.rows[j].order
	})
	for i, r := range e.rows {
		if r.isPrimaryKey {
			e.pkeys = append(e.pkeys, i)
		} else {
			e.field = append(e.field, i)
		}
	}

	e.hasForeignKey = e.getForeignKey()

	cw := e.getColumnWidths()

	pad := 2
	m := e.margin
	w := e.width
	h := e.height + pad*2

	baseLine := 2 + pad
	nnw := (cw.notNull + 1) * w
	lnw := 0
	if cw.logicalName != 0 {
		lnw = (cw.logicalName + 2) * w
	}
	pnw := (cw.physicalName + 2) * w
	dtw := (cw.dataType + 2) * w

	columnW := nnw + lnw + pnw + dtw
	rw := columnW + m*2
	rh := (len(e.pkeys) + len(e.field)) * h
	ew := (width(e.title) + 2) * w
	if ew > columnW {
		rw = ew + m*2
	}

	cellLeft := []int{
		m + w/2,
		m + nnw,
		m + nnw + lnw,
		m + nnw + lnw + pnw,
	}

	t := h + m/2
	y := t + len(e.pkeys)*h

	e.view = Rectangle{0, 0, rw + m*2, rh + m*2 + h}
	e.tiltePos = Point{m, h + m - baseLine}
	e.frame = Rectangle{m, t, rw, rh}
	e.separateLine = TwoPointCoordinates{m, y, m + rw, y}

	drawRow := func(indexes []int) {
		for _, i := range indexes {
			frame := &Rectangle{m, t - h, rw, h}
			c := e.rows[i]
			fnm := e.schema + "." + e.name + "." + c.physicalName.nm
			e.collision[fnm] = frame
			c.frame = frame
			if c.isNotNull {
				c.notNull = Rectangle{cellLeft[0], t - h + 4, w / 2, h - 8}
			}
			c.logicalName.pt = Point{cellLeft[1], t - baseLine}
			c.physicalName.pt = Point{cellLeft[2], t - baseLine}
			c.dataType.pt = Point{cellLeft[3], t - baseLine}
			e.rows[i] = c
			t += h
		}
	}

	t += h
	drawRow(e.pkeys)
	drawRow(e.field)
}

func (e *Entity) Draw(s *svg.SVG, dx int, dy int) {
	s.Group(`id="`+e.name+`"`, e.font)
	s.Text(dx+e.tiltePos.x, dy+e.tiltePos.y, e.title)

	if !e.isChildren {
		s.Rect(dx+e.frame.x, dy+e.frame.y, e.frame.w, e.frame.h, e.lineStyle)
	} else {
		s.Roundrect(dx+e.frame.x, dy+e.frame.y, e.frame.w, e.frame.h, e.radius, e.radius, e.lineStyle)
	}

	s.Line(dx+e.separateLine.x1, dy+e.separateLine.y1, dx+e.separateLine.x2, dy+e.separateLine.y2, e.lineStyle)

	drawRow := func(indexes []int) {
		for _, i := range indexes {
			c := e.rows[i]
			if c.isNotNull {
				r := c.notNull
				s.Rect(dx+r.x, dy+r.y, r.w, r.h, e.lineStyle)
			}
			s.Text(dx+c.logicalName.pt.x, dy+c.logicalName.pt.y, c.logicalName.nm)
			s.Text(dx+c.physicalName.pt.x, dy+c.physicalName.pt.y, c.physicalName.nm)
			s.Text(dx+c.dataType.pt.x, dy+c.dataType.pt.y, c.dataType.nm, e.typeFont)
		}
	}

	drawRow(e.pkeys)
	drawRow(e.field)
	s.Gend()
}

func (e *Entity) getForeignKey() bool {
	for _, r := range e.rows {
		if r.relationaly.valid() {
			return true
		}
	}
	return false
}

func (e *Entity) getColumnWidths() (cw columnWidth) {
	cw = columnWidth{1, 0, 0, 0}
	f := func(indexes []int) {
		for _, i := range indexes {
			c := e.rows[i]
			cw.logicalName = max(cw.logicalName, width(c.logicalName.nm))
			cw.physicalName = max(cw.physicalName, width(c.physicalName.nm))
			cw.dataType = max(cw.dataType, width(c.dataType.nm))
		}
	}

	f(e.pkeys)
	f(e.field)

	return
}

type columnWidth struct {
	notNull      int
	physicalName int
	dataType     int
	logicalName  int
}

func width(s string) int {
	w := 0
	for _, c := range s {
		if c < 256 {
			w += 1
		} else {
			w += 2
		}
	}
	return w
}
