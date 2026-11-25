package platform

import (
	"testing"
	"time"
)

func TestDateToDays(t *testing.T) {
	d, err := DateToDays("1970-01-02")
	if err != nil || d != 1 {
		t.Fatalf("got %d, %v", d, err)
	}
	if _, err := DateToDays("not-a-date"); err == nil {
		t.Fatal("expected error for bad date")
	}
}

func TestOptionalDateDefaultsToday(t *testing.T) {
	got, err := OptionalDateToDays("")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := DateToDays(localDate(time.Now()))
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}
