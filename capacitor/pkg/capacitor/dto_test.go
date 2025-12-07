package capacitor

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// Test DTOs

type TestUserDTO struct {
	BaseDTO
	UserID     int64     `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Age        int       `json:"age"`
	CreateTime time.Time `json:"created_at"`
	UpdateTime time.Time `json:"updated_at"`
	VersionNum int64     `json:"version"`
}

func (u *TestUserDTO) Validate(ctx context.Context) error {
	u.ClearValidationErrors()

	if err := RequiredInt("UserID", u.UserID); err != nil {
		u.AddValidationError("UserID", err.Error())
	}
	if err := RequiredString("Email", u.Email); err != nil {
		u.AddValidationError("Email", err.Error())
	}
	if err := RequiredString("Name", u.Name); err != nil {
		u.AddValidationError("Name", err.Error())
	}
	if err := MinValue("Age", u.Age, 0); err != nil {
		u.AddValidationError("Age", err.Error())
	}

	return u.ValidateWithType(ctx, "TestUserDTO")
}

func (u *TestUserDTO) CacheKey() string {
	return fmt.Sprintf("user:%d", u.UserID)
}

func (u *TestUserDTO) CacheTTL() time.Duration {
	return 1 * time.Hour
}

func (u *TestUserDTO) Version() int64 {
	return u.VersionNum
}

func (u *TestUserDTO) SetVersion(version int64) {
	u.VersionNum = version
}

func (u *TestUserDTO) CreatedAt() time.Time {
	return u.CreateTime
}

func (u *TestUserDTO) UpdatedAt() time.Time {
	return u.UpdateTime
}

func (u *TestUserDTO) Touch() {
	u.UpdateTime = time.Now()
}

func (u *TestUserDTO) ID() int64 {
	return u.UserID
}

func (u *TestUserDTO) SetID(id int64) {
	u.UserID = id
}

func (u *TestUserDTO) Clone() *TestUserDTO {
	return &TestUserDTO{
		UserID:     u.UserID,
		Email:      u.Email,
		Name:       u.Name,
		Age:        u.Age,
		CreateTime: u.CreateTime,
		UpdateTime: u.UpdateTime,
		VersionNum: u.VersionNum,
	}
}

func (u *TestUserDTO) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         u.UserID,
		"email":      u.Email,
		"name":       u.Name,
		"age":        u.Age,
		"created_at": u.CreateTime,
		"updated_at": u.UpdateTime,
		"version":    u.VersionNum,
	}
}

func (u *TestUserDTO) FromMap(m map[string]interface{}) error {
	if id, ok := m["id"].(int64); ok {
		u.UserID = id
	}
	if email, ok := m["email"].(string); ok {
		u.Email = email
	}
	if name, ok := m["name"].(string); ok {
		u.Name = name
	}
	if age, ok := m["age"].(int); ok {
		u.Age = age
	}
	if createdAt, ok := m["created_at"].(time.Time); ok {
		u.CreateTime = createdAt
	}
	if updatedAt, ok := m["updated_at"].(time.Time); ok {
		u.UpdateTime = updatedAt
	}
	if version, ok := m["version"].(int64); ok {
		u.VersionNum = version
	}
	return nil
}

// Tests

func TestBaseDTO_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dto     *TestUserDTO
		wantErr bool
	}{
		{
			name: "valid DTO",
			dto: &TestUserDTO{
				UserID: 1,
				Email:  "test@example.com",
				Name:   "Test User",
				Age:    25,
			},
			wantErr: false,
		},
		{
			name: "missing email",
			dto: &TestUserDTO{
				UserID: 1,
				Name:   "Test User",
				Age:    25,
			},
			wantErr: true,
		},
		{
			name: "negative age",
			dto: &TestUserDTO{
				UserID: 1,
				Email:  "test@example.com",
				Name:   "Test User",
				Age:    -5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dto.Validate(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSerializationFormat_String(t *testing.T) {
	tests := []struct {
		format SerializationFormat
		want   string
	}{
		{FormatJSON, "JSON"},
		{FormatMsgPack, "MessagePack"},
		{FormatProtobuf, "Protobuf"},
		{FormatCustom, "Custom"},
		{SerializationFormat(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONSerializer(t *testing.T) {
	serializer := NewJSONSerializer[*TestUserDTO]()

	dto := &TestUserDTO{
		UserID:     1,
		Email:      "test@example.com",
		Name:       "Test User",
		Age:        25,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
		VersionNum: 1,
	}

	// Test Marshal
	data, err := serializer.Marshal(dto)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Test Unmarshal
	newDTO := &TestUserDTO{}
	if err := serializer.Unmarshal(data, newDTO); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify data
	if newDTO.UserID != dto.UserID {
		t.Errorf("UserID = %v, want %v", newDTO.UserID, dto.UserID)
	}
	if newDTO.Email != dto.Email {
		t.Errorf("Email = %v, want %v", newDTO.Email, dto.Email)
	}
	if newDTO.Name != dto.Name {
		t.Errorf("Name = %v, want %v", newDTO.Name, dto.Name)
	}

	// Test Format
	if got := serializer.Format(); got != FormatJSON {
		t.Errorf("Format() = %v, want %v", got, FormatJSON)
	}
}

func TestSimpleValidator(t *testing.T) {
	validator := NewSimpleValidator[*TestUserDTO]()

	// Add validation rules
	validator.AddRule("email", func(ctx context.Context, dto *TestUserDTO) error {
		return RequiredString("email", dto.Email)
	})

	validator.AddRule("age", func(ctx context.Context, dto *TestUserDTO) error {
		return InRange("age", dto.Age, 0, 150)
	})

	tests := []struct {
		name    string
		dto     *TestUserDTO
		wantErr bool
	}{
		{
			name: "valid",
			dto: &TestUserDTO{
				Email: "test@example.com",
				Age:   25,
			},
			wantErr: false,
		},
		{
			name: "missing email",
			dto: &TestUserDTO{
				Age: 25,
			},
			wantErr: true,
		},
		{
			name: "invalid age",
			dto: &TestUserDTO{
				Email: "test@example.com",
				Age:   200,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(context.Background(), tt.dto)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Test ValidateField
	dto := &TestUserDTO{Age: 200}
	if err := validator.ValidateField(context.Background(), dto, "age"); err == nil {
		t.Error("ValidateField() expected error for invalid age")
	}

	// Test RemoveRule
	validator.RemoveRule("age")
	if err := validator.Validate(context.Background(), dto); err == nil {
		t.Error("After RemoveRule, validation should fail for missing email only")
	}
}

func TestValidationHelpers(t *testing.T) {
	t.Run("RequiredString", func(t *testing.T) {
		if err := RequiredString("field", ""); err == nil {
			t.Error("RequiredString() expected error for empty string")
		}
		if err := RequiredString("field", "value"); err != nil {
			t.Errorf("RequiredString() unexpected error: %v", err)
		}
	})

	t.Run("RequiredInt", func(t *testing.T) {
		if err := RequiredInt("field", 0); err == nil {
			t.Error("RequiredInt() expected error for zero")
		}
		if err := RequiredInt("field", 1); err != nil {
			t.Errorf("RequiredInt() unexpected error: %v", err)
		}
	})

	t.Run("MinLength", func(t *testing.T) {
		if err := MinLength("field", "ab", 3); err == nil {
			t.Error("MinLength() expected error")
		}
		if err := MinLength("field", "abc", 3); err != nil {
			t.Errorf("MinLength() unexpected error: %v", err)
		}
	})

	t.Run("MaxLength", func(t *testing.T) {
		if err := MaxLength("field", "abcd", 3); err == nil {
			t.Error("MaxLength() expected error")
		}
		if err := MaxLength("field", "abc", 3); err != nil {
			t.Errorf("MaxLength() unexpected error: %v", err)
		}
	})

	t.Run("MinValue", func(t *testing.T) {
		if err := MinValue("field", 5, 10); err == nil {
			t.Error("MinValue() expected error")
		}
		if err := MinValue("field", 10, 10); err != nil {
			t.Errorf("MinValue() unexpected error: %v", err)
		}
	})

	t.Run("MaxValue", func(t *testing.T) {
		if err := MaxValue("field", 15, 10); err == nil {
			t.Error("MaxValue() expected error")
		}
		if err := MaxValue("field", 10, 10); err != nil {
			t.Errorf("MaxValue() unexpected error: %v", err)
		}
	})

	t.Run("InRange", func(t *testing.T) {
		if err := InRange("field", 5, 10, 20); err == nil {
			t.Error("InRange() expected error for value below range")
		}
		if err := InRange("field", 25, 10, 20); err == nil {
			t.Error("InRange() expected error for value above range")
		}
		if err := InRange("field", 15, 10, 20); err != nil {
			t.Errorf("InRange() unexpected error: %v", err)
		}
	})
}

func TestDTOPool(t *testing.T) {
	pool := NewDTOPool(func() *TestUserDTO {
		return &TestUserDTO{}
	})

	// Get from pool
	dto1 := pool.Get()
	if dto1 == nil {
		t.Fatal("Get() returned nil")
	}

	// Modify and return to pool
	dto1.UserID = 123
	pool.Put(dto1)

	// Get again (might be the same object)
	dto2 := pool.Get()
	if dto2 == nil {
		t.Fatal("Get() returned nil")
	}

	// Return to pool
	pool.Put(dto2)
}

func TestDTOCache(t *testing.T) {
	cache := NewDTOCache[int64, *TestUserDTO]()

	// Test Set and Get
	dto := &TestUserDTO{
		UserID: 1,
		Email: "test@example.com",
		Name:  "Test User",
	}

	cache.Set(1, dto)

	retrieved, exists := cache.Get(1)
	if !exists {
		t.Error("Get() expected value to exist")
	}
	if retrieved.UserID != dto.UserID {
		t.Errorf("Get() UserID = %v, want %v", retrieved.UserID, dto.UserID)
	}

	// Test Size
	if size := cache.Size(); size != 1 {
		t.Errorf("Size() = %v, want 1", size)
	}

	// Test Delete
	cache.Delete(1)
	if _, exists := cache.Get(1); exists {
		t.Error("Delete() value still exists")
	}

	// Test Clear
	cache.Set(1, dto)
	cache.Set(2, &TestUserDTO{UserID: 2})
	cache.Clear()
	if size := cache.Size(); size != 0 {
		t.Errorf("Clear() size = %v, want 0", size)
	}
}

func TestTestUserDTO_Cacheable(t *testing.T) {
	dto := &TestUserDTO{UserID: 1}

	// Test CacheKey
	key := dto.CacheKey()
	if key == "" {
		t.Error("CacheKey() returned empty string")
	}

	// Test CacheTTL
	ttl := dto.CacheTTL()
	if ttl != 1*time.Hour {
		t.Errorf("CacheTTL() = %v, want 1h", ttl)
	}
}

func TestTestUserDTO_Versionable(t *testing.T) {
	dto := &TestUserDTO{}

	// Test SetVersion and Version
	dto.SetVersion(5)
	if v := dto.Version(); v != 5 {
		t.Errorf("Version() = %v, want 5", v)
	}
}

func TestTestUserDTO_Timestamped(t *testing.T) {
	now := time.Now()
	dto := &TestUserDTO{
		CreateTime: now,
		UpdateTime: now,
	}

	// Test CreatedAt
	if !dto.CreatedAt().Equal(now) {
		t.Errorf("CreatedAt() = %v, want %v", dto.CreatedAt(), now)
	}

	// Test UpdatedAt
	if !dto.UpdatedAt().Equal(now) {
		t.Errorf("UpdatedAt() = %v, want %v", dto.UpdatedAt(), now)
	}

	// Test Touch
	time.Sleep(10 * time.Millisecond)
	dto.Touch()
	if !dto.UpdatedAt().After(now) {
		t.Error("Touch() did not update timestamp")
	}
}

func TestTestUserDTO_Identifiable(t *testing.T) {
	dto := &TestUserDTO{}

	// Test SetID and ID
	dto.SetID(123)
	if id := dto.ID(); id != 123 {
		t.Errorf("ID() = %v, want 123", id)
	}
}

func TestTestUserDTO_Cloneable(t *testing.T) {
	original := &TestUserDTO{
		UserID: 1,
		Email: "test@example.com",
		Name:  "Test User",
		Age:   25,
	}

	clone := original.Clone()

	// Verify clone has same values
	if clone.UserID != original.UserID {
		t.Errorf("Clone UserID = %v, want %v", clone.UserID, original.UserID)
	}
	if clone.Email != original.Email {
		t.Errorf("Clone Email = %v, want %v", clone.Email, original.Email)
	}

	// Verify clone is independent
	clone.Email = "changed@example.com"
	if original.Email == clone.Email {
		t.Error("Clone is not independent")
	}
}

func TestTestUserDTO_Mappable(t *testing.T) {
	original := &TestUserDTO{
		UserID: 1,
		Email: "test@example.com",
		Name:  "Test User",
		Age:   25,
	}

	// Test ToMap
	m := original.ToMap()
	if m["id"] != int64(1) {
		t.Errorf("ToMap() id = %v, want 1", m["id"])
	}
	if m["email"] != "test@example.com" {
		t.Errorf("ToMap() email = %v, want test@example.com", m["email"])
	}

	// Test FromMap
	dto := &TestUserDTO{}
	if err := dto.FromMap(m); err != nil {
		t.Errorf("FromMap() error = %v", err)
	}
	if dto.UserID != original.UserID {
		t.Errorf("FromMap() UserID = %v, want %v", dto.UserID, original.UserID)
	}
	if dto.Email != original.Email {
		t.Errorf("FromMap() Email = %v, want %v", dto.Email, original.Email)
	}
}

func TestBaseDTO_ValidationErrors(t *testing.T) {
	dto := &BaseDTO{}

	// Test AddValidationError
	dto.AddValidationError("field1", "error1")
	dto.AddValidationError("field2", "error2")

	err := dto.Validate(context.Background())
	if err == nil {
		t.Fatal("Validate() expected error")
	}

	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatal("Validate() error is not ValidationError")
	}

	if len(vErr.Errors) != 2 {
		t.Errorf("ValidationError has %d errors, want 2", len(vErr.Errors))
	}

	// Test ClearValidationErrors
	dto.ClearValidationErrors()
	if err := dto.Validate(context.Background()); err != nil {
		t.Errorf("Validate() after clear unexpected error: %v", err)
	}
}

// Benchmarks

func BenchmarkJSONSerializer_Marshal(b *testing.B) {
	serializer := NewJSONSerializer[*TestUserDTO]()
	dto := &TestUserDTO{
		UserID:     1,
		Email:     "test@example.com",
		Name:      "Test User",
		Age:       25,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = serializer.Marshal(dto)
	}
}

func BenchmarkJSONSerializer_Unmarshal(b *testing.B) {
	serializer := NewJSONSerializer[*TestUserDTO]()
	dto := &TestUserDTO{
		UserID:     1,
		Email:     "test@example.com",
		Name:      "Test User",
		Age:       25,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	data, _ := serializer.Marshal(dto)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		newDTO := &TestUserDTO{}
		_ = serializer.Unmarshal(data, newDTO)
	}
}

func BenchmarkDTOPool_GetPut(b *testing.B) {
	pool := NewDTOPool(func() *TestUserDTO {
		return &TestUserDTO{}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dto := pool.Get()
		pool.Put(dto)
	}
}

func BenchmarkDTOCache_Get(b *testing.B) {
	cache := NewDTOCache[int64, *TestUserDTO]()
	dto := &TestUserDTO{UserID: 1, Email: "test@example.com"}
	cache.Set(1, dto)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(1)
	}
}

func BenchmarkDTOCache_Set(b *testing.B) {
	cache := NewDTOCache[int64, *TestUserDTO]()
	dto := &TestUserDTO{UserID: 1, Email: "test@example.com"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Set(int64(i), dto)
	}
}

func BenchmarkValidator_Validate(b *testing.B) {
	validator := NewSimpleValidator[*TestUserDTO]()
	validator.AddRule("email", func(ctx context.Context, dto *TestUserDTO) error {
		return RequiredString("email", dto.Email)
	})
	validator.AddRule("age", func(ctx context.Context, dto *TestUserDTO) error {
		return InRange("age", dto.Age, 0, 150)
	})

	dto := &TestUserDTO{
		Email: "test@example.com",
		Age:   25,
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = validator.Validate(ctx, dto)
	}
}

func BenchmarkDTO_Clone(b *testing.B) {
	dto := &TestUserDTO{
		UserID:     1,
		Email:     "test@example.com",
		Name:      "Test User",
		Age:       25,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = dto.Clone()
	}
}

func BenchmarkDTO_ToMap(b *testing.B) {
	dto := &TestUserDTO{
		UserID:     1,
		Email:     "test@example.com",
		Name:      "Test User",
		Age:       25,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = dto.ToMap()
	}
}
