package canvas

import (
	"io"
	"sort"

	svg "github.com/ajstarks/svgo"
)

type relation struct {
	left  []*relation
	right []*relation

	entity *Entity
	use    bool
	offset int
}

type Canvas struct {
	groups  []*relation
	bgStyle string
}

func NewCanvas() *Canvas {
	return &Canvas{
		groups: []*relation{},
		bgStyle: StyleMap{
			"fill":   "white",
			"stroke": "none",
		}.String(),
	}
}

func (c *Canvas) RegisterEntity(e *Entity) {
	c.groups = append(c.groups, &relation{
		entity: e,
	})
}

func (c *Canvas) linkage() {
	for _, g := range c.groups {
		g.use = false
		g.left = g.left[:0]
		g.right = g.right[:0]
	}
	for _, g := range c.groups {
		for _, row := range g.entity.rows {
			rel := row.relationaly
			for _, g2 := range c.groups {
				if rel.schema == g2.entity.schema && rel.table == g2.entity.name {
					g.right = append(g.right, g2)
					g2.left = append(g2.left, g)
				}
			}
		}
	}
}

type singleNodesInfo struct {
	w, h  int
	group []*relation
}

func (n *singleNodesInfo) draw(s *svg.SVG, dx, dy int, space int) {
	x := 0
	for _, g := range n.group {
		e := g.entity
		e.Draw(s, dx+x, dy)
		x += e.view.w + space
	}
}

func (c *Canvas) extractSingle(space int) (singleNodes *singleNodesInfo) {
	if len(c.groups) == 0 {
		return
	}

	w := 0
	h := 0
	group := []*relation{}
	ng := []*relation{}
	for _, g := range c.groups {
		if len(g.left) == 0 && len(g.right) == 0 {
			group = append(group, g)
		} else {
			ng = append(ng, g)
		}
	}
	c.groups = ng

	for _, g := range group {
		e := g.entity
		w += e.view.w + space
		if e.view.h > h {
			h = e.view.h
		}
	}

	return &singleNodesInfo{w, h, group}
}

type levelBox struct {
	lv int
	w  int
	h  int
	g  []*relation
}

type regionInfo struct {
	w, h   int
	levels []*levelBox
}

func (ri *regionInfo) draw(s *svg.SVG, dx, dy int, space int) {
	levels := ri.levels
	size := len(levels)
	x := 0
	for i, lvl := range levels {
		y := dy
		nx := x + lvl.w + space
		for _, g := range lvl.g {
			e := g.entity
			e.Draw(s, x, y)
			for _, r := range e.rows {
				if r.relationaly.valid() {
					x1 := x + r.frame.x + r.frame.w
					y1 := y + r.frame.y + r.frame.h>>1
					rnm := r.relationaly.fullname()
					ny := dy
					for li := i + 1; li < size; li += 1 {
						for _, rg := range levels[li].g {
							re := rg.entity
							c := re.collision[rnm]
							if c != nil {
								x2 := nx + c.x
								y2 := ny + c.y + c.h>>1
								s.Line(x1, y1, x2, y2, e.lineStyle)
							}
							ny += re.view.h + space
						}
					}
				}
			}
			y += e.view.h + space
		}
		x = nx
	}
}

func (c *Canvas) extractRegion(space int) (region *regionInfo) {
	if len(c.groups) == 0 {
		return
	}

	w := 0
	h := 0
	base := c.groups[0]
	base.use = true

	var left func(*relation, int)
	var right func(*relation, int)

	left = func(c *relation, o int) {
		c.use = true
		c.offset = o
		for _, l := range c.left {
			if !l.use || l.offset > o-1 {
				left(l, o-1)
			}
		}
		for _, r := range c.right {
			if !r.use || r.offset < o+1 {
				right(r, o+1)
			}
		}
	}
	right = func(c *relation, o int) {
		c.use = true
		c.offset = o
		for _, r := range c.right {
			if !r.use || r.offset < o+1 {
				right(r, o+1)
			}
		}
		for _, l := range c.left {
			if !l.use || l.offset > o-1 {
				left(l, o-1)
			}
		}
	}
	left(base, 0)
	right(base, 0)

	l := map[int][]*relation{}
	ng := []*relation{}
	for _, r := range c.groups {
		if r.use {
			l[r.offset] = append(l[r.offset], r)
		} else {
			ng = append(ng, r)
		}
	}
	c.groups = ng

	levels := []*levelBox{}
	for offset, r := range l {
		tw := 0
		th := 0
		for _, g := range r {
			e := g.entity
			if e.view.w > tw {
				tw = e.view.w
			}
			th += e.view.h
		}
		th += (len(r) - 1) * space
		levels = append(levels, &levelBox{lv: offset, w: tw, h: th, g: r})
		w += tw + space
		if th > h {
			h = th
		}
	}

	sort.Slice(levels, func(i, j int) bool { return levels[i].lv < levels[j].lv })

	return &regionInfo{w, h, levels}
}

func (c *Canvas) OutputSVG(o io.Writer) {
	c.linkage()

	space := 48
	singleNodes := c.extractSingle(space)

	regions := []*regionInfo{}
	for {
		r := c.extractRegion(space)
		if r == nil {
			break
		}
		regions = append(regions, r)
	}

	w := 0
	h := 0
	if singleNodes != nil {
		w = singleNodes.w
		h = singleNodes.h
	}
	for i := range regions {
		w2 := regions[i].w
		h2 := regions[i].h
		if w < w2 {
			w = w2
		}
		h += space + h2
	}

	s := svg.New(o)
	s.Start(w, h)
	s.Rect(0, 0, w, h, c.bgStyle)

	regionY := 0
	if singleNodes != nil {
		singleNodes.draw(s, 0, 0, space)
		regionY += singleNodes.h + space
	}

	for _, region := range regions {
		region.draw(s, 0, regionY, space)
		regionY += region.h + space
	}

	s.End()
}
