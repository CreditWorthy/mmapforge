//go:build unix

package example

import (
	"os"
	"path/filepath"
	"testing"
)

const benchRecords = 1024

func benchMarketCapStore(b *testing.B) *MarketCapStore {
	b.Helper()
	path := filepath.Join(b.TempDir(), "bench.mmf")
	s, createErr := NewMarketCapStore(path)
	if createErr != nil {
		b.Fatalf("NewMarketCapStore: %v", createErr)
	}
	var id uint64
	for i := 0; i < benchRecords; i++ {
		idx, appendErr := s.Append()
		if appendErr != nil {
			b.Fatalf("Append: %v", appendErr)
		}
		if setErr := s.Set(idx, &MarketCapRecord{
			ID:        id,
			Price:     float64(i) * 1.5,
			Volume:    float64(i) * 1000,
			MarketCap: float64(i) * 1e6,
			Stale:     i%2 == 0,
		}); setErr != nil {
			b.Fatalf("Set: %v", setErr)
		}
		id++
	}
	return s
}

func BenchmarkMarketCap_GetID(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.GetID(i % benchRecords); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkMarketCap_GetPrice(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.GetPrice(i % benchRecords); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkMarketCap_GetVolume(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.GetVolume(i % benchRecords); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkMarketCap_GetMarketCap(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.GetMarketCap(i % benchRecords); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkMarketCap_GetStale(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.GetStale(i % benchRecords); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkMarketCap_SetID(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	var v uint64
	for i := 0; i < b.N; i++ {
		v++
		if setErr := s.SetID(i%benchRecords, v); setErr != nil {
			b.Fatal(setErr)
		}
	}
}

func BenchmarkMarketCap_SetPrice(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if setErr := s.SetPrice(i%benchRecords, float64(i)*1.5); setErr != nil {
			b.Fatal(setErr)
		}
	}
}

func BenchmarkMarketCap_SetVolume(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if setErr := s.SetVolume(i%benchRecords, float64(i)*1000); setErr != nil {
			b.Fatal(setErr)
		}
	}
}

func BenchmarkMarketCap_SetMarketCap(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if setErr := s.SetMarketCap(i%benchRecords, float64(i)*1e6); setErr != nil {
			b.Fatal(setErr)
		}
	}
}

func BenchmarkMarketCap_SetStale(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if setErr := s.SetStale(i%benchRecords, i%2 == 0); setErr != nil {
			b.Fatal(setErr)
		}
	}
}

func BenchmarkMarketCap_BulkGet(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, readErr := s.Get(i % benchRecords); readErr != nil {
			b.Fatal(readErr)
		}
	}
}

func BenchmarkMarketCap_BulkSet(b *testing.B) {
	s := benchMarketCapStore(b)
	defer s.Close()
	rec := &MarketCapRecord{
		ID:        42,
		Price:     99.95,
		Volume:    50000,
		MarketCap: 1e9,
		Stale:     false,
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if setErr := s.Set(i%benchRecords, rec); setErr != nil {
			b.Fatal(setErr)
		}
	}
}

func BenchmarkMarketCap_Append(b *testing.B) {
	path := filepath.Join(b.TempDir(), "bench_append.mmf")
	s, createErr := NewMarketCapStore(path)
	if createErr != nil {
		b.Fatalf("NewMarketCapStore: %v", createErr)
	}
	defer s.Close()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, appendErr := s.Append(); appendErr != nil {
			b.StopTimer()
			s.Close()
			os.Remove(path)
			s, createErr = NewMarketCapStore(path)
			if createErr != nil {
				b.Fatalf("NewMarketCapStore: %v", createErr)
			}
			b.StartTimer()
		}
	}
}

func BenchmarkMarketCap_OpenClose(b *testing.B) {
	path := filepath.Join(b.TempDir(), "bench_oc.mmf")
	s, createErr := NewMarketCapStore(path)
	if createErr != nil {
		b.Fatalf("NewMarketCapStore: %v", createErr)
	}
	for i := 0; i < 100; i++ {
		if _, appendErr := s.Append(); appendErr != nil {
			b.Fatal(appendErr)
		}
	}
	s.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		store, openErr := OpenMarketCapStore(path)
		if openErr != nil {
			b.Fatal(openErr)
		}
		store.Close()
	}
}
