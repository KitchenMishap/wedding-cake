package cake

import (
	"github.com/kitchenmishap/wedding-cake/inputtier"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

type Cake struct {
	folderPath string
	config     *smalltree.SmallTreeConfig

	inputTier *inputtier.InputTier
}

func (c *Cake) Close() error {
	err := c.inputTier.Close()
	if err != nil {
		return err
	}
	return nil
}

func (c *Cake) AppendHash(gpi types.GlobalPi, hash []byte) error {
	err := c.inputTier.AppendHash(gpi, hash)
	if err != nil {
		return err
	}
	return nil
}
