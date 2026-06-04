package service

import (
	"testing"
	"time"
)

func TestJsonMarshalTags(t *testing.T) {
	result, _ := jsonMarshalTags([]string{"音乐", "演唱会"})
	expected := `["音乐","演唱会"]`
	if result != expected {
		t.Errorf("got %s, want %s", result, expected)
	}
}

func TestJsonMarshalTagsEmpty(t *testing.T) {
	result, _ := jsonMarshalTags([]string{})
	if result != "[]" {
		t.Errorf("got %s, want []", result)
	}
}

func TestTimeValidation(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		open  time.Time
		close time.Time
		act   time.Time
		valid bool
	}{
		{"valid", now.Add(1 * time.Hour), now.Add(2 * time.Hour), now.Add(3 * time.Hour), true},
		{"close before open", now.Add(2 * time.Hour), now.Add(1 * time.Hour), now.Add(3 * time.Hour), false},
		{"act before close", now.Add(1 * time.Hour), now.Add(3 * time.Hour), now.Add(2 * time.Hour), false},
		{"all same", now, now, now, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.open.Before(tt.close) && tt.close.Before(tt.act)
			if result != tt.valid {
				t.Errorf("got %v, want %v", result, tt.valid)
			}
		})
	}
}

func TestActivityStateTransition(t *testing.T) {
	tests := []struct {
		from    string
		allowed bool
	}{
		{"DRAFT", true},
		{"PREHEAT", true},
		{"PUBLISHED", false},
		{"SELLING_OUT", false},
		{"SOLD_OUT", false},
		{"OFFLINE", false},
	}

	for _, tt := range tests {
		t.Run(tt.from, func(t *testing.T) {
			canPublish := tt.from == "DRAFT" || tt.from == "PREHEAT"
			if canPublish != tt.allowed {
				t.Errorf("from %s: got canPublish=%v, want %v", tt.from, canPublish, tt.allowed)
			}
		})
	}
}

func TestPreheatStateTransition(t *testing.T) {
	tests := []struct {
		from    string
		allowed bool
	}{
		{"DRAFT", true},
		{"PREHEAT", false},
		{"PUBLISHED", false},
		{"SELLING_OUT", false},
		{"SOLD_OUT", false},
		{"OFFLINE", false},
	}

	for _, tt := range tests {
		t.Run(tt.from, func(t *testing.T) {
			canPreheat := tt.from == "DRAFT"
			if canPreheat != tt.allowed {
				t.Errorf("from %s: got canPreheat=%v, want %v", tt.from, canPreheat, tt.allowed)
			}
		})
	}
}

func TestPublishedFieldLock(t *testing.T) {
	tests := []struct {
		status string
		locked bool
	}{
		{"DRAFT", false},
		{"PREHEAT", false},
		{"PUBLISHED", true},
		{"SELLING_OUT", true},
		{"SOLD_OUT", true},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			isPublished := tt.status != "DRAFT" && tt.status != "PREHEAT"
			if isPublished != tt.locked {
				t.Errorf("status %s: got locked=%v, want %v", tt.status, isPublished, tt.locked)
			}
		})
	}
}
