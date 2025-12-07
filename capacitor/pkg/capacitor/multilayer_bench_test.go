package capacitor

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Benchmarks for MultiLayerDAL

func BenchmarkMultiLayerDAL_Get_L1Hit(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1, TTL: time.Minute},
			{Name: "L2", Layer: l2, TTL: time.Hour},
		},
		EnablePromotion: true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate L1
	for i := 0; i < 1000; i++ {
		l1.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Get(ctx, fmt.Sprintf("key%d", i%1000))
	}
}

func BenchmarkMultiLayerDAL_Get_L2Hit(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1, TTL: time.Minute},
			{Name: "L2", Layer: l2, TTL: time.Hour},
		},
		EnablePromotion: false, // Disable to avoid L1 pollution
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate L2 only
	for i := 0; i < 1000; i++ {
		l2.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Get(ctx, fmt.Sprintf("key%d", i%1000))
	}
}

func BenchmarkMultiLayerDAL_Get_L2HitWithPromotion(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1, TTL: time.Minute},
			{Name: "L2", Layer: l2, TTL: time.Hour},
		},
		EnablePromotion: true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate L2 only
	for i := 0; i < 1000; i++ {
		l2.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Get(ctx, fmt.Sprintf("key%d", i%1000))
	}
}

func BenchmarkMultiLayerDAL_Get_Miss(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1, TTL: time.Minute},
			{Name: "L2", Layer: l2, TTL: time.Hour},
		},
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Get(ctx, fmt.Sprintf("nonexistent%d", i))
	}
}

func BenchmarkMultiLayerDAL_Set_WriteThrough(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")
	l3 := newMockLayer[string, int]("L3")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1, TTL: time.Minute},
			{Name: "L2", Layer: l2, TTL: time.Hour},
			{Name: "L3", Layer: l3, TTL: 0},
		},
		WriteThrough: true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Set(ctx, fmt.Sprintf("key%d", i%10000), i)
	}
}

func BenchmarkMultiLayerDAL_Set_L1Only(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")
	l3 := newMockLayer[string, int]("L3")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1, TTL: time.Minute},
			{Name: "L2", Layer: l2, TTL: time.Hour},
			{Name: "L3", Layer: l3, TTL: 0},
		},
		WriteThrough: false,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Set(ctx, fmt.Sprintf("key%d", i%10000), i)
	}
}

func BenchmarkMultiLayerDAL_Delete(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate
	keys := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = fmt.Sprintf("key%d", i)
		l1.Set(ctx, keys[i], i)
		l2.Set(ctx, keys[i], i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Delete(ctx, keys[i])
	}
}

func BenchmarkMultiLayerDAL_Exists(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate L2
	for i := 0; i < 1000; i++ {
		l2.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Exists(ctx, fmt.Sprintf("key%d", i%1000))
	}
}

func BenchmarkMultiLayerDAL_GetMulti(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
		EnablePromotion: true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate: 50% in L1, 50% in L2
	for i := 0; i < 1000; i++ {
		if i%2 == 0 {
			l1.Set(ctx, fmt.Sprintf("key%d", i), i)
		} else {
			l2.Set(ctx, fmt.Sprintf("key%d", i), i)
		}
	}

	// Prepare batch of keys
	keys := make([]string, 10)
	for i := range keys {
		keys[i] = fmt.Sprintf("key%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.GetMulti(ctx, keys)
	}
}

func BenchmarkMultiLayerDAL_SetMulti(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
		WriteThrough: true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Prepare batch of items
	items := make(map[string]int, 10)
	for i := 0; i < 10; i++ {
		items[fmt.Sprintf("key%d", i)] = i
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.SetMulti(ctx, items)
	}
}

func BenchmarkMultiLayerDAL_DeleteMulti(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Prepare batch of keys
	keys := make([]string, 10)
	for i := range keys {
		keys[i] = fmt.Sprintf("key%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Pre-populate before each iteration
		for _, key := range keys {
			l1.Set(ctx, key, 100)
			l2.Set(ctx, key, 100)
		}

		dal.DeleteMulti(ctx, keys)
	}
}

// Mixed workload benchmarks

func BenchmarkMultiLayerDAL_MixedWorkload_80R_10W_10D(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1, TTL: time.Minute},
			{Name: "L2", Layer: l2, TTL: time.Hour},
		},
		EnablePromotion: true,
		WriteThrough:    true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		l1.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		switch i % 10 {
		case 0, 1, 2, 3, 4, 5, 6, 7: // 80% reads
			dal.Get(ctx, fmt.Sprintf("key%d", i%1000))
		case 8: // 10% writes
			dal.Set(ctx, fmt.Sprintf("key%d", i%1000), i)
		case 9: // 10% deletes
			dal.Delete(ctx, fmt.Sprintf("key%d", i%1000))
		}
	}
}

// Scalability benchmarks with different layer counts

func BenchmarkMultiLayerDAL_2Layers(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
		EnablePromotion: true,
		WriteThrough:    true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate L2
	for i := 0; i < 1000; i++ {
		l2.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Get(ctx, fmt.Sprintf("key%d", i%1000))
	}
}

func BenchmarkMultiLayerDAL_3Layers(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")
	l3 := newMockLayer[string, int]("L3")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
			{Name: "L3", Layer: l3},
		},
		EnablePromotion: true,
		WriteThrough:    true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate L3
	for i := 0; i < 1000; i++ {
		l3.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Get(ctx, fmt.Sprintf("key%d", i%1000))
	}
}

func BenchmarkMultiLayerDAL_4Layers(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")
	l3 := newMockLayer[string, int]("L3")
	l4 := newMockLayer[string, int]("L4")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
			{Name: "L3", Layer: l3},
			{Name: "L4", Layer: l4},
		},
		EnablePromotion: true,
		WriteThrough:    true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate L4
	for i := 0; i < 1000; i++ {
		l4.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dal.Get(ctx, fmt.Sprintf("key%d", i%1000))
	}
}

// Parallel benchmarks

func BenchmarkMultiLayerDAL_Get_Parallel(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
		EnablePromotion: true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		l1.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			dal.Get(ctx, fmt.Sprintf("key%d", i%1000))
			i++
		}
	})
}

func BenchmarkMultiLayerDAL_Set_Parallel(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
		WriteThrough: true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			dal.Set(ctx, fmt.Sprintf("key%d", i%10000), i)
			i++
		}
	})
}

func BenchmarkMultiLayerDAL_MixedWorkload_Parallel(b *testing.B) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
		EnablePromotion: true,
		WriteThrough:    true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		l1.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 10 {
			case 0, 1, 2, 3, 4, 5, 6, 7: // 80% reads
				dal.Get(ctx, fmt.Sprintf("key%d", i%1000))
			case 8: // 10% writes
				dal.Set(ctx, fmt.Sprintf("key%d", i%1000), i)
			case 9: // 10% deletes
				dal.Delete(ctx, fmt.Sprintf("key%d", i%1000))
			}
			i++
		}
	})
}
