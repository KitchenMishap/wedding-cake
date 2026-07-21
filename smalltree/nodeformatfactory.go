package smalltree

// A factory for a codec pair for a "NodeFormats" (Nf) kind of node encoding

type LevelsCodecNfFactory struct {
	config *SmallTreeConfig
}

// Check that implements
var _ LevelsCodecFactory = (*LevelsCodecNfFactory)(nil)

func NewLevelsCodecNfFactory(config *SmallTreeConfig) LevelsCodecFactory {
	return &LevelsCodecNfFactory{
		config: config,
	}
}

func (lcnf *LevelsCodecNfFactory) MakeLevelsEncoder() LevelsEncoder {
	return &LevelsEncoderNf{config: lcnf.config}
}
func (lcnf *LevelsCodecNfFactory) MakeLevelDecoder(indexBytes []byte, nodesBytes []byte) LevelDecoder {
	result := LevelDecoderNf{
		config:     lcnf.config,
		nodesBytes: nodesBytes,
	}
	result.ConfigureWithIndexBytes(indexBytes)
	return &result
}
