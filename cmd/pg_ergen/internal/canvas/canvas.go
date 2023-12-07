package canvas

import (
	"fmt"
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
	id := "single-nodes"
	s.Def()
	s.Gid(id)
	x := 0
	for _, g := range n.group {
		e := g.entity
		e.Draw(s, x, 0)
		x += e.view.w + space
	}
	s.Gend()
	s.DefEnd()
	s.Use(dx, dy, "#"+id)
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

var seq int = 0

func (ri *regionInfo) draw(s *svg.SVG, dx, dy int, space int) {
	id := fmt.Sprintf("region-%d", seq)
	s.Def()
	s.Gid(id)
	levels := ri.levels
	size := len(levels)
	x := 0
	half := space >> 1
	for i, lvl := range levels {
		cpx := x + lvl.w + space
		y := 0
		for _, g := range lvl.g {
			e := g.entity
			ml1 := (lvl.w + space - e.view.w) / 2
			e.Draw(s, x+ml1, y+half)
			for _, r := range e.rows {
				if r.relationaly.valid() {
					x1 := ml1 + x + r.frame.x + r.frame.w
					y1 := half + y + r.frame.y + r.frame.h>>1
					rnm := r.relationaly.fullname()
					nx := x + lvl.w + space
					hh := 0

				search:
					for li := i + 1; li < size; li += 1 {
						curlvl := levels[li]
						if hh < curlvl.h+half {
							hh = curlvl.h + half
						}
						ny := 0
						for _, rg := range curlvl.g {
							re := rg.entity
							ml2 := (curlvl.w + space - re.view.w) / 2
							c := re.collision[rnm]
							if c != nil {
								x2 := ml2 + nx + c.x
								y2 := half + ny + c.y + c.h>>1
								if cpx == nx {
									s.Bezier(x1, y1, cpx, y1, nx, y2, x2, y2, e.lineStyle)
								} else {
									hhh := 0
									cy := 0
									if y1 < y2 {
										cy = (y2-y1)/2 + y1
									} else {
										cy = (y1-y2)/2 + y2
									}
									if cy > hh/2 {
										hhh = hh
									}
									s.Bezier(x1, y1, cpx, y1, cpx, hhh, cpx+half, hhh, e.lineStyle)
									s.Bezier(x2, y2, nx, y2, nx, hhh, nx-half, hhh, e.lineStyle)
									s.Line(cpx+half, hhh, nx-half, hhh, e.lineStyle)
								}
								break search
							}
							ny += re.view.h + space
						}
						nx += curlvl.w + space
					}
				}
			}
			y += e.view.h + space
		}
		x = x + lvl.w + space
	}
	s.Gend()
	s.DefEnd()
	s.Use(dx, dy, "#"+id)
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

	return &regionInfo{w + space, h + space, levels}
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
