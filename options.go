package mmapforge

// StoreOption configures how a Store is opened or created.
type StoreOption func(*storeConfig)

type storeConfig struct {
	readOnly  bool
	oneWriter bool
}

// WithReadOnly opens the store in read-only mode.
// Writes, appends, and grow operations return ErrReadOnly.
// The file is mapped with PROT_READ only and no file lock is acquired.
func WithReadOnly() StoreOption {
	return func(c *storeConfig) {
		c.readOnly = true
	}
}

// WithOneWriter acquires an exclusive file lock (flock) on a sidecar
// .lock file, ensuring only one writer process at a time.
// If another writer already holds the lock, open fails with ErrLocked.
func WithOneWriter() StoreOption {
	return func(c *storeConfig) {
		c.oneWriter = true
	}
}

func applyOptions(opts []StoreOption) storeConfig {
	var cfg storeConfig
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}
