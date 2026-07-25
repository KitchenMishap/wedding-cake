package cake

import (
	"github.com/kitchenmishap/wedding-cake/smalltree"
)

type Cake struct {
	folderPath string
	config     *smalltree.SmallTreeConfig
}

func (c *Cake) Close() error {
	return nil
}
