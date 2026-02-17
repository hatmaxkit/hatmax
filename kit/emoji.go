package kit

// Emoji represents an emoji character.
type Emoji string

// Standard emoji presets.
const (
	EmojiCheck    Emoji = "✅"
	EmojiCross    Emoji = "❌"
	EmojiWarning  Emoji = "⚠️"
	EmojiInfo     Emoji = "ℹ️"
	EmojiStar     Emoji = "⭐"
	EmojiHeart    Emoji = "❤️"
	EmojiPin      Emoji = "📌"
	EmojiFolder   Emoji = "📁"
	EmojiEdit     Emoji = "✏"
	EmojiTrash    Emoji = "🗑"
	EmojiSearch   Emoji = "🔍"
	EmojiSettings Emoji = "⚙"
	EmojiUser     Emoji = "👤"
	EmojiLock     Emoji = "🔒"
	EmojiUnlock   Emoji = "🔓"
	EmojiMail     Emoji = "✉"
	EmojiPhone    Emoji = "📞"
	EmojiCalendar Emoji = "📅"
	EmojiClock    Emoji = "🕐"
	EmojiHome     Emoji = "🏠"
	EmojiLink     Emoji = "🔗"
	EmojiPlus     Emoji = "➕"
	EmojiMinus    Emoji = "➖"
	EmojiRefresh  Emoji = "🔄"
	EmojiDownload Emoji = "⬇"
	EmojiUpload   Emoji = "⬆"
	EmojiSave     Emoji = "💾"
	EmojiFire     Emoji = "🔥"
	EmojiSparkle  Emoji = "✨"
)

// String returns the emoji as a string.
func (e Emoji) String() string {
	return string(e)
}
