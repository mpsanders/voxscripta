# CHAT GPT - Youtube Transcript Extraction Methods

**You:**

What would be the best, and most complete way to grab the transcript from a YouTube video from a golang codebase. Consider all options (e.g. roll-your-own/libraries/cli tools/etc)

* * *

**ChatGPT:**

For a Go codebase, I would not try to reimplement YouTube's transcript extraction unless you have a very strong reason. YouTube's internal endpoints and page structures change frequently enough that maintaining your own extractor becomes an ongoing job.

The most robust architecture is:

**Go application → `yt-dlp` → existing YouTube captions → audio transcription fallback**

That gives you both speed and completeness.

## Recommended hierarchy

| Method | Reliability | Speed | Complexity | Handles auto-captions | Handles no captions |
| --- | --- | --- | --- | --- | --- |
| `yt-dlp` subprocess | ★★★★★ | ★★★★★ | Low | Yes | No |
| Go transcript library | ★★★ | ★★★★★ | Very low | Yes | No |
| Python `youtube-transcript-api` | ★★★★ | ★★★★★ | Medium | Yes | No |
| Roll your own YouTube extraction | ★★ | ★★★★★ | High | Yes | No |
| YouTube Data API | ★★★★★ | ★★★★★ | High/auth | Limited to videos you can manage | No |
| Download audio + Whisper | ★★★★★ | ★★ | Medium | N/A | **Yes** |
| Browser automation | ★★ | ★ | High | Yes | No |

My production implementation would use **`yt-dlp` as the primary extractor, with Whisper as the fallback**.

---

# 1. Best primary option: `yt-dlp`

`yt-dlp` is by far the most battle-tested YouTube extractor. It supports both creator-uploaded subtitles and YouTube-generated automatic captions. It also has active maintenance specifically because sites like YouTube continually change their internals. The project itself recommends its nightly channel for regular users because YouTube-side changes can break older versions. [![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/yt-dlp/yt-dlp)

It can explicitly retrieve:

```
--write-subs
--write-auto-subs```

and supports language patterns such as:

```
--sub-langs "en.*"```

[![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/yt-dlp/yt-dlp)

From Go, simply execute it as a subprocess.

For example:

Go

```
func downloadTranscript(ctx context.Context, url, dir string) error {
	cmd := exec.CommandContext(
		ctx,
		"yt-dlp",

		"--skip-download",

		"--write-subs",
		"--write-auto-subs",

		"--sub-langs", "en.*",

		"--sub-format", "vtt",

		"--output", filepath.Join(dir, "%(id)s.%(ext)s"),

		url,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"yt-dlp failed: %w\n%s",
			err,
			output,
		)
	}

	return nil
}```

`--skip-download` is useful here: it tells yt-dlp **not to download the video while still writing related files such as subtitles**. [![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/yt-dlp/yt-dlp)

You'll typically end up with something like:

```
dQw4w9WgXcQ.en.vtt```

Then parse the VTT into your own internal representation.

For example:

Go

```
type Transcript struct {
	VideoID string
	Language string
	Source   TranscriptSource
	Segments []Segment
}

type Segment struct {
	Start time.Duration
	End   time.Duration
	Text  string
}

type TranscriptSource string

const (
	SourceManual  TranscriptSource = "youtube_manual"
	SourceAuto    TranscriptSource = "youtube_auto"
	SourceWhisper TranscriptSource = "whisper"
)```

That structure is much more useful downstream than keeping a blob of text.

---

# 2. Let `yt-dlp` discover what exists first

A slightly more sophisticated implementation would perform a metadata/discovery pass first.

For example:

Bash

```
yt-dlp --list-subs VIDEO_URL```

yt-dlp can enumerate available subtitle tracks and languages. [![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/yt-dlp/yt-dlp)

You can also use its JSON output:

Bash

```
yt-dlp -J VIDEO_URL```

The project explicitly recommends machine-readable mechanisms such as `-J` and `--print` rather than scraping its normal stdout because ordinary CLI output may change. [![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/yt-dlp/yt-dlp)

From Go:

Go

```
cmd := exec.CommandContext(
	ctx,
	"yt-dlp",
	"-J",
	"--skip-download",
	url,
)

output, err := cmd.Output()```

Then:

Go

```
var info struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Language string `json:"language"`

	Subtitles map[string][]struct {
		Ext string `json:"ext"`
		URL string `json:"url"`
	} `json:"subtitles"`

	AutomaticCaptions map[string][]struct {
		Ext string `json:"ext"`
		URL string `json:"url"`
	} `json:"automatic_captions"`
}

err = json.Unmarshal(output, &info)```

This gives you a particularly nice architecture:

```
Go
 │
 ├── yt-dlp metadata
 │       │
 │       ├── Manual captions available?
 │       │       └── Download
 │       │
 │       ├── Auto captions available?
 │       │       └── Download
 │       │
 │       └── Neither
 │
 └── Audio transcription fallback```

You could even download the subtitle URL directly from Go after yt-dlp discovers it.

That avoids temporary subtitle files completely.

---

# 3. Caption preference strategy

I would prefer sources in this order:

```
1. Creator/manual captions in requested language
2. Creator/manual captions in original language
3. YouTube auto captions in requested language
4. YouTube auto captions in original language
5. Translated YouTube captions
6. Whisper transcription of audio```

Manual captions often include better punctuation, names and formatting than automatic captions.

One important point: don't request:

```
--sub-langs all```

by default.

YouTube exposes a huge number of automatically translated caption variants, and you can easily end up retrieving hundreds of tracks.

Instead:

```
en
en.*```

or dynamically select the user's requested language.

---

# 4. Go-native libraries

There are Go packages that reverse-engineer the same YouTube transcript mechanisms.

One reasonably feature-rich example is:

`github.com/horiagug/youtube-transcript-api-go`

It supports manual and auto-generated captions, languages, timestamps, concurrent retrieval and JSON/text output. [![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/horiagug/youtube-transcript-api-go)

Usage is attractive because everything remains in-process:

Go

```
import youtube "github.com/horiagug/youtube-transcript-api-go"```

Conceptually:

Go

```
transcript, err := client.Fetch(videoID)```

The advantage is obvious:

```
Go binary
   ↓
Go package
   ↓
YouTube```

No Python.

No external executable.

No temp files.

However, there is an important downside.

These packages depend on **undocumented YouTube interfaces**. YouTube can change those interfaces at any time.

You can see the same issue throughout transcript-scraping libraries: seemingly innocuous YouTube changes can cause the underlying caption metadata to disappear or move. [![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/algolia/youtube-captions-scraper/issues/30?utm_source=chatgpt.com)

That means I'd happily use a Go library for:

* internal tooling
* personal projects
* low-volume services

but I'd be more cautious making one the only production mechanism.

`yt-dlp` has a vastly larger maintenance community whose entire purpose is dealing with exactly these changes.

---

# 5. `youtube-transcript-api`

Another excellent project is the Python:

```
youtube-transcript-api```

It retrieves both manually created and automatically generated subtitles, doesn't require an API key and doesn't require a browser. [![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/jdepoix/youtube-transcript-api)

Its basic API is:

Python

```
from youtube_transcript_api import YouTubeTranscriptApi

ytt_api = YouTubeTranscriptApi()
ytt_api.fetch(video_id)```

[![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/jdepoix/youtube-transcript-api)

You could invoke it from Go:

Go

```
cmd := exec.CommandContext(
	ctx,
	"python",
	"-m",
	"your_transcript_wrapper",
	videoID,
)```

and return JSON.

But if you're already accepting a Python runtime dependency, I would generally choose **yt-dlp** because it covers vastly more edge cases beyond transcripts.

---

# 6. Official YouTube API

You'd reasonably expect:

```
YouTube Data API
    ↓
captions.list
    ↓
captions.download```

to be the cleanest solution.

Unfortunately it isn't suitable for arbitrary public YouTube videos.

Google's current API documentation says `captions.download` requires the authenticated user to **have permission to edit the video**. [![](https://www.google.com/s2/favicons?domain=https://developers.google.com&sz=128)Google for Developers](https://developers.google.com/youtube/v3/docs/captions/download)

Likewise, `captions.list` requires OAuth authorization. [![](https://www.google.com/s2/favicons?domain=https://developers.google.com&sz=128)Google for Developers](https://developers.google.com/youtube/v3/docs/captions/list)

So it is great for:

```
Videos belonging to your channel```

but not:

```
Give me the transcript of this random public YouTube video.```

For your own content, though, this should absolutely be your preferred mechanism.

---

# 7. Roll-your-own extraction

Technically, you can replicate what many transcript libraries do.

The rough process is:

```
GET YouTube video page
       ↓
Find player/bootstrap JSON
       ↓
Find captionTracks
       ↓
Choose language
       ↓
Retrieve timed-text endpoint
       ↓
Parse XML/JSON
       ↓
Normalise transcript```

Conceptually you'll encounter metadata resembling:

JSON

```
{
  "captionTracks": [
    {
      "baseUrl": "...",
      "name": {...},
      "vssId": "...",
      "languageCode": "en",
      "kind": "asr"
    }
  ]
}```

`kind: "asr"` generally indicates automatically generated speech recognition captions.

You then call the `baseUrl`, which returns the timed transcript representation.

You could implement all of that in Go using:

Go

```
net/http
encoding/json
encoding/xml```

and have zero external dependencies.

### Why I wouldn't

You'd effectively be maintaining a miniature YouTube extractor.

Changes could involve:

```
HTML structure
player JSON
Innertube API
visitor tokens
cookies
signatures
rate limiting
bot detection
PO tokens
caption endpoints
client variants```

And suddenly:

```
Friday: works
Monday: production is broken```

That maintenance cost dwarfs the convenience of avoiding yt-dlp.

---

# 8. The important fallback: transcribe the actual audio

There is a class of video for which **none of the transcript extraction techniques work**:

```
Video has no captions.```

If your requirement is genuinely:

> Get me a transcript from any usable YouTube video

then your system needs speech-to-text.

This is where yt-dlp becomes particularly useful again.

Download only the audio:

Bash

```
yt-dlp \
  -x \
  --audio-format wav \
  -o "%(id)s.%(ext)s" \
  VIDEO_URL```

yt-dlp supports extracting audio and converting it using ffmpeg. [![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/yt-dlp/yt-dlp)

Then send it through:

```
Whisper
faster-whisper
whisper.cpp
OpenAI transcription API
Deepgram
AssemblyAI
Google Speech-to-Text
AWS Transcribe
etc.```

For a self-contained local service, I'd strongly consider:

```
whisper.cpp```

or:

```
faster-whisper service```

depending on your infrastructure.

---

# 9. What I'd actually build

I'd separate the concept of **transcript acquisition** from YouTube itself.

Something like:

Go

```
type TranscriptProvider interface {
	Get(
		ctx context.Context,
		video Video,
		options TranscriptOptions,
	) (*Transcript, error)
}```

Then:

Go

```
type YouTubeCaptionProvider struct {
	YTDLP string
}

type WhisperProvider struct {
	// local or remote transcription client
}```

And orchestrate them:

Go

```
type TranscriptService struct {
	providers []TranscriptProvider
}```

Something roughly like:

Go

```
func (s *TranscriptService) Get(
	ctx context.Context,
	video Video,
	opts TranscriptOptions,
) (*Transcript, error) {

	for _, provider := range s.providers {

		transcript, err := provider.Get(ctx, video, opts)

		if err == nil && len(transcript.Segments) > 0 {
			return transcript, nil
		}
	}

	return nil, ErrTranscriptUnavailable
}```

Then configure:

Go

```
service := TranscriptService{
	providers: []TranscriptProvider{
		NewYTDLPManualCaptionProvider(),
		NewYTDLPAutoCaptionProvider(),
		NewWhisperProvider(),
	},
}```

---

# 10. I'd also preserve timestamps

Don't immediately turn this:

```
00:04:32.400 --> 00:04:36.800
Black holes aren't actually holes in the conventional sense.```

into:

```
Black holes aren't actually holes in the conventional sense.```

Store:

JSON

```
{
  "start": 272.4,
  "duration": 4.4,
  "text": "Black holes aren't actually holes in the conventional sense."
}```

You can always generate plain text later.

But timestamps enable:

```
LLM citations
"YouTube at 4:32"
semantic search
video navigation
chapter generation
clip creation
quote verification
RAG
speaker analysis
summarisation by section```

This becomes extremely valuable if the transcript is going into an AI pipeline.

---

# 11. Normalisation layer

I would normalise all input formats:

```
YouTube JSON
VTT
SRT
XML
Whisper JSON```

into one structure:

Go

```
type Transcript struct {
	VideoID string

	Language string

	Source TranscriptSource

	Generated bool

	Segments []TranscriptSegment
}

type TranscriptSegment struct {
	Start time.Duration
	End   time.Duration
	Text  string
}```

Then add convenience methods:

Go

```
func (t Transcript) Text() string
func (t Transcript) Markdown() string
func (t Transcript) SRT() string
func (t Transcript) VTT() string```

Your downstream code never needs to know where the transcript came from.

---

# 12. Production pipeline

The final system I'd use is:

```
                     YouTube URL
                          │
                          ▼
                    Extract video ID
                          │
                          ▼
                    yt-dlp metadata
                          │
              ┌───────────┴───────────┐
              │                       │
        Manual captions?        Auto captions?
              │                       │
             YES                     YES
              │                       │
              ▼                       ▼
          Download                 Download
              │                       │
              └───────────┬───────────┘
                          ▼
                    Parse / normalise
                          │
                          ▼
                      Transcript
                          
                    NO CAPTIONS
                          │
                          ▼
                 yt-dlp audio only
                          │
                          ▼
                       ffmpeg
                          │
                          ▼
                  Whisper / STT
                          │
                          ▼
                    Normalise
                          │
                          ▼
                      Transcript```

That gets you extremely close to:

> **If a human can hear speech in the YouTube video, my system can produce a transcript.**

---

## One improvement I'd make over simply invoking `yt-dlp`

Run yt-dlp inside a small abstraction such as:

Go

```
type YTDLP struct {
	Path    string
	Timeout time.Duration
}```

and expose operations like:

Go

```
func (y *YTDLP) Inspect(ctx context.Context, url string) (*VideoInfo, error)

func (y *YTDLP) DownloadSubtitles(
	ctx context.Context,
	url string,
	languages []string,
) ([]Subtitle, error)

func (y *YTDLP) DownloadAudio(
	ctx context.Context,
	url string,
) (io.ReadCloser, error)```

Then the rest of your Go application doesn't know or care that yt-dlp is Python underneath.

It is simply another executable dependency, much like:

```
ffmpeg
git
docker```

---

## My ranking

If this were my Go project:

**1. `yt-dlp` subprocess — primary choice**

Best combination of reliability, capabilities and maintenance.

**2. Whisper/audio transcription — mandatory fallback if "complete" really means complete**

Catches videos without captions.

**3. Go transcript library — good optimisation**

Potentially use this as a fast first attempt:

```
Go library
    ↓ fails
yt-dlp
    ↓ fails/no captions
Whisper```

**4. `youtube-transcript-api`**

Excellent library, but adds Python without offering as much overall capability as yt-dlp.

**5. Official API**

Use for videos you control. Not suitable for arbitrary videos because caption downloading requires edit permission. [![](https://www.google.com/s2/favicons?domain=https://developers.google.com&sz=128)Google for Developers](https://developers.google.com/youtube/v3/docs/captions/download)

**6. Roll-your-own YouTube scraper**

Only do this if building and maintaining YouTube extraction infrastructure is itself part of your product.

**7. Playwright/Selenium/browser automation**

Last resort. Much heavier and more fragile than directly hitting the underlying endpoints.

### Best overall architecture

```
Go
 │
 ├─ Native transcript interface
 │
 ├─ yt-dlp
 │    ├─ manual captions
 │    └─ auto captions
 │
 └─ fallback
      ├─ yt-dlp audio extraction
      └─ Whisper```

For something you're likely to feed into summarisation/RAG/AI processing, **this is the approach I'd choose rather than a Go-only solution**. It gives you the highest chance of getting a timestamped transcript while isolating YouTube's continual breaking changes behind a tool whose maintainers spend their time fixing exactly that problem. [![](https://www.google.com/s2/favicons?domain=https://github.com&sz=128)GitHub](https://github.com/yt-dlp/yt-dlp)