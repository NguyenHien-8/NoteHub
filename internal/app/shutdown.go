package app

func (b *Backend) Close() error {
	if b == nil || b.Store == nil {
		return nil
	}
	return b.Store.Close()
}
