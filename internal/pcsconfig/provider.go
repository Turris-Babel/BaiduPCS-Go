package pcsconfig

// ConfigProvider provides read-only access to configuration values
// required by other packages. Implemented by PCSConfig to promote
// decoupling from the global Config variable.
type ConfigProvider interface {
	GetMaxDownloadLoad() int
	GetNoCheck() bool
	GetCacheSize() int
	GetMaxDownloadRate() int64
	GetEnableHTTPS() bool
	GetMaxParallel() int
}

// Below are adapter methods so *PCSConfig satisfies ConfigProvider.
func (c *PCSConfig) GetMaxDownloadLoad() int   { return c.MaxDownloadLoad }
func (c *PCSConfig) GetNoCheck() bool          { return c.NoCheck }
func (c *PCSConfig) GetCacheSize() int         { return c.CacheSize }
func (c *PCSConfig) GetMaxDownloadRate() int64 { return c.MaxDownloadRate }
func (c *PCSConfig) GetEnableHTTPS() bool      { return c.EnableHTTPS }
func (c *PCSConfig) GetMaxParallel() int       { return c.MaxParallel }
