package rawpaq

import "testing"

func TestDefaultConfigValid(t *testing.T) {
	if err := DefaultConfig().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigRejectsPartialFEC(t *testing.T) {
	config := DefaultConfig()
	config.DataShards = 10
	if err := config.Validate(); err == nil {
		t.Fatal("partial FEC config accepted")
	}
}
