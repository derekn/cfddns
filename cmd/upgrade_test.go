package main

import "testing"

func TestUpgrade(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		wantUpgrade bool
		wantErr     bool
	}{
		{"Older", "2015.1.1", true, false},
		{"Newer", "2035.1.1", false, false},
		// {"PreRelease", "2035.1.1-dev", true, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version = test.version
			upgraded, err := selfUpgrade()
			if (err != nil) != test.wantErr {
				t.Errorf("expected %v got %v - %v", test.wantErr, (err != nil), err)
			}
			if test.wantUpgrade != upgraded {
				t.Errorf("expected %v got %v", test.wantUpgrade, upgraded)
			}
		})
	}
}
