package conduit

type Conduit struct {
	dstChan chan []byte
	srcChan chan []byte
}

func New() *Conduit {
	return &Conduit{
		dstChan: make(chan []byte),
		srcChan: make(chan []byte),
	}
}

func (c *Conduit) Src() chan []byte {
	return c.srcChan
}

func (c *Conduit) Dst() chan []byte {
	return c.dstChan
}
