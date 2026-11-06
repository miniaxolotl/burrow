package capture

type KeyStore interface {
	Key(scope string) ([]byte, error)
}
type OSKeys struct{}
