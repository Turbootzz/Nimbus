package models

import "testing"

func TestCardSizes(t *testing.T) {
	for _, size := range []string{"1x1", "2x1", "2x2"} {
		if !IsValidCardSize(size) || !IsValidWidgetCardSize(size) {
			t.Errorf("%s should be valid for services and widgets", size)
		}
	}
	if IsValidCardSize("1x2") {
		t.Error("1x2 is for widgets only")
	}
	if !IsValidWidgetCardSize("1x2") {
		t.Error("1x2 should be valid for widgets")
	}
	if IsValidWidgetCardSize("3x3") {
		t.Error("3x3 is not a size")
	}
}
