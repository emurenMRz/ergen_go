package canvas

type StyleMap map[string]string

func (sm StyleMap) String() string {
	s := ""

	for key, value := range sm {
		if len(s) > 0 {
			s += ";"
		}
		s += key + ":" + value
	}

	return s
}
