package transactions

// NoteOutput represents any note that can be used as utxo
type NoteOutput interface {
	Transparent() bool
}
