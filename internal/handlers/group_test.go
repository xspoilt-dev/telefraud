package handlers

import (
	"testing"
)

func TestContainsScamKeyword(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Hey guys, he is a scammer!", true},
		{"ei lok ta ekta batpar", true},
		{"pura batpari kortese", true},
		{"chor sala taka niye palise", true},
		{"ye banda chor hai", true},
		{"ye bada dhokebaaz nikla", true},
		{"yeh fraud hai mat bhejo paise", true},
		{"vai o ekta protarok", true},
		{"he is a big thief and cheater", true},
		{"lutera loot liya sab", true},
		{"hello everyone, how are you?", false},
		{"let's trade crypto safely", false},
		{"good morning team", false},
	}

	for _, tt := range tests {
		got := containsScamKeyword(tt.input)
		if got != tt.want {
			t.Errorf("containsScamKeyword(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
