package error

type Location struct {
	Filename string
	Line     uint
	Column   uint
	Index    uint
}

func (self *Location) Advance(newline bool) {
	self.Index += 1
	if newline {
		self.Column = 1
		self.Line += 1
	} else {
		self.Column += 1
	}
}

func NewLocation(filename string) Location {
	return Location{
		Filename: filename,
		Line:     1,
		Column:   1,
		Index:    0,
	}
}
