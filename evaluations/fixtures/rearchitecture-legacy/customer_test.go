package customer

import "testing"

func TestLegacyRecordRemainsReadable(t *testing.T) {
	if (LegacyRecord{Name: "Ada"}).Name != "Ada" {
		t.Fatal("legacy record must remain readable")
	}
}
