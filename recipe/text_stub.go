package recipe

const (
	TextKindText   = "text"
	TextKindBinary = "binary"
	TextKindLarge  = "large"
)

func ClassifyText(sample []byte, size int) string { return TextKindText }
