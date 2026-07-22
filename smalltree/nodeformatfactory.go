package smalltree

// A factory for a codec pair for a "NodeFormats" (Nf) kind of node encoding

type LevelsCodecNfFactory struct {
	config *SmallTreeConfig
}

// Check that implements
var _ LevelsCodecFactory = (*LevelsCodecNfFactory)(nil)

func NewLevelsCodecNfFactory(config *SmallTreeConfig) *LevelsCodecNfFactory {
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

func (lcnf *LevelsCodecNfFactory) MakeDecoderLevelsTest(indexBytes [][]byte, nodesBytes [][]byte, prefixNibbles byte,
	rootNodeId LocalNodeIdType, rootLevel byte) DecoderLevelsTest {
	decoders := make([]LevelDecoder, len(indexBytes))
	for i := range indexBytes {
		decoders[i] = lcnf.MakeLevelDecoder(indexBytes[i], nodesBytes[i])
	}
	return DecoderLevelsTest{levels: decoders, prefixNibbles: prefixNibbles, config: lcnf.config, rootNodeId: rootNodeId, rootLevel: rootLevel}
}
