package validator

import "testing"

func TestUniqueUsesCollectionValues(t *testing.T) {
	validate, err := New()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		value any
		valid bool
	}{
		{"different integers", []int{1, 2}, true},
		{"duplicate integers", []int{1, 1}, false},
		{"different strings", []string{"first", "second"}, true},
		{"duplicate strings", []string{"same", "same"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validate.Var(tt.value, "unique"); (err == nil) != tt.valid {
				t.Fatalf("unique(%v) = %v, valid = %v", tt.value, err, tt.valid)
			}
		})
	}
}
