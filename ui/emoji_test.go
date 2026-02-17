package ui

import "testing"

func TestEmojiString(t *testing.T) {
	tests := []struct {
		emoji Emoji
		want  string
	}{
		{EmojiCheck, "✅"},
		{EmojiCross, "❌"},
		{EmojiWarning, "⚠️"},
		{EmojiTrash, "🗑"},
		{Emoji("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.emoji), func(t *testing.T) {
			if got := tt.emoji.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmojiPresets(t *testing.T) {
	presets := []Emoji{
		EmojiCheck,
		EmojiCross,
		EmojiWarning,
		EmojiInfo,
		EmojiStar,
		EmojiHeart,
		EmojiPin,
		EmojiFolder,
		EmojiEdit,
		EmojiTrash,
		EmojiSearch,
		EmojiSettings,
		EmojiUser,
		EmojiLock,
		EmojiUnlock,
		EmojiMail,
		EmojiPhone,
		EmojiCalendar,
		EmojiClock,
		EmojiHome,
		EmojiLink,
		EmojiPlus,
		EmojiMinus,
		EmojiRefresh,
		EmojiDownload,
		EmojiUpload,
		EmojiSave,
		EmojiFire,
		EmojiSparkle,
	}

	for _, e := range presets {
		if e == "" {
			t.Errorf("Emoji preset is empty")
		}
		if e.String() == "" {
			t.Errorf("Emoji.String() returned empty for %v", e)
		}
	}
}
