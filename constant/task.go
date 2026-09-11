package constant

type TaskPlatform string

const (
	TaskPlatformSuno       TaskPlatform = "suno"
	TaskPlatformMidjourney              = "mj"
	// TaskPlatformMiniMaxH3 is stored separately from the legacy MiniMax
	// platform so the polling worker can select the OpenAI-compatible H3 API
	// without changing the channel type (35) used for key and billing lookup.
	TaskPlatformMiniMaxH3 TaskPlatform = "minimax-h3"
	// TaskPlatformVideoCTMoai identifies the dedicated CTMOAI video-account
	// integration. It intentionally is not a channel type: credentials and
	// model capabilities live in the video_accounts table.
	TaskPlatformVideoCTMoai TaskPlatform = "ctmoai-video"
)

const (
	SunoActionMusic  = "MUSIC"
	SunoActionLyrics = "LYRICS"

	TaskActionGenerate          = "generate"
	TaskActionTextGenerate      = "textGenerate"
	TaskActionFirstTailGenerate = "firstTailGenerate"
	TaskActionReferenceGenerate = "referenceGenerate"
	TaskActionRemix             = "remixGenerate"
)

var SunoModel2Action = map[string]string{
	"suno_music":  SunoActionMusic,
	"suno_lyrics": SunoActionLyrics,
}
