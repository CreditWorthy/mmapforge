package example

//go:generate mmapforge -input example_types.go

// mmapforge:schema version=1
type MarketCap struct {
	ID        uint64  `mmap:"id"`
	Price     float64 `mmap:"price"`
	Volume    float64 `mmap:"volume"`
	MarketCap float64 `mmap:"market_cap"`
	Stale     bool    `mmap:"stale"`
}
