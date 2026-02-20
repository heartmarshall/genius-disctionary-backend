package domain

import "testing"

func TestInt32PtrToIntPtr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   *int32
		want *int
	}{
		{name: "nil returns nil", in: nil, want: nil},
		{name: "zero value", in: int32Ptr(0), want: intPtr(0)},
		{name: "positive value", in: int32Ptr(42), want: intPtr(42)},
		{name: "negative value", in: int32Ptr(-1), want: intPtr(-1)},
		{name: "max int32", in: int32Ptr(2147483647), want: intPtr(2147483647)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Int32PtrToIntPtr(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Errorf("Int32PtrToIntPtr(%v) = %v, want nil", tt.in, *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("Int32PtrToIntPtr(%v) = nil, want %d", *tt.in, *tt.want)
			}
			if *got != *tt.want {
				t.Errorf("Int32PtrToIntPtr(%v) = %d, want %d", *tt.in, *got, *tt.want)
			}
		})
	}
}

func TestIntPtrToInt32Ptr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   *int
		want *int32
	}{
		{name: "nil returns nil", in: nil, want: nil},
		{name: "zero value", in: intPtr(0), want: int32Ptr(0)},
		{name: "positive value", in: intPtr(42), want: int32Ptr(42)},
		{name: "negative value", in: intPtr(-1), want: int32Ptr(-1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IntPtrToInt32Ptr(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Errorf("IntPtrToInt32Ptr(%v) = %v, want nil", tt.in, *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("IntPtrToInt32Ptr(%v) = nil, want %d", *tt.in, *tt.want)
			}
			if *got != *tt.want {
				t.Errorf("IntPtrToInt32Ptr(%v) = %d, want %d", *tt.in, *got, *tt.want)
			}
		})
	}
}

func int32Ptr(v int32) *int32 { return &v }
func intPtr(v int) *int       { return &v }
