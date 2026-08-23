package decimal

import "testing"

func TestParseUnits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		decimals int
		want     string
	}{
		{name: "whole", value: "10", decimals: 4, want: "100000"},
		{name: "fraction", value: "366.14886400", decimals: 8, want: "36614886400"},
		{name: "leading decimal", value: ".5", decimals: 2, want: "50"},
		{name: "negative", value: "-1.25", decimals: 2, want: "-125"},
		{name: "zero", value: "0.0000", decimals: 4, want: "0"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseUnits(tt.value, tt.decimals)
			if err != nil {
				t.Fatalf("ParseUnits() error = %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("ParseUnits() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseUnitsRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", ".", "1.234", "1.2.3", "one", "--1"} {
		if _, err := ParseUnits(value, 2); err == nil {
			t.Errorf("ParseUnits(%q) unexpectedly succeeded", value)
		}
	}
}
