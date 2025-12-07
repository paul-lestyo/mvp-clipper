package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

var client *openai.Client

func init() {
	config := openai.DefaultConfig(os.Getenv("OPENAI_API_KEY"))
	config.BaseURL = os.Getenv("OPENAI_API_BASE_URL")
	client = openai.NewClientWithConfig(config)
}

func AnalyzeTranscript(srtText string, videoTitle string, channelName string) (string, error) {

	prompt := fmt.Sprintf(`
You are an expert viral content strategist specializing in Indonesian social media. Your task is to find the BEST 3 viral moments from this transcript that will perform exceptionally well on TikTok, Instagram Reels, and YouTube Shorts.

Channel: %s
Video Title: %s

VIRAL CONTENT CRITERIA:
1. HOOK FACTOR: Moments that grab attention in first 3 seconds
2. EMOTIONAL TRIGGERS: Funny, shocking, surprising, controversial, relatable, or inspiring
3. COMPLETE CONTEXT: Each clip must be a complete conversation/story that makes sense independently (usually ended with dots or pauses in srt)
4. OPTIMAL LENGTH: 60-90 seconds maximum for best engagement
5. UNIVERSAL APPEAL: Can be understood by general Indonesian audience (not niche references)

CONTENT TYPES TO PRIORITIZE:
- Roasting/savage comebacks
- Funny misunderstandings or awkward moments  
- Shocking confessions or reveals
- Relatable life experiences
- Controversial hot takes
- Inspiring or motivational moments
- Celebrity drama or gossip
- Unexpected plot twists in stories

CONTEXT REQUIREMENTS:
- Include enough setup so viewers understand the situation
- Capture the full reaction/aftermath
- Ensure natural conversation flow (complete thoughts)
- Add 2-3 seconds buffer before and after for smooth transitions

Transcript:
%s

Analyze EVERY part of the transcript and output strictly in JSON without markdown code blocks:

[
  {
    "start": "00:00:10",
    "end": "00:01:25",
    "why": "roasting moment", -> [2 words max]
    "title": "Deddy Corbuzier Roasting Habis-habisan Artis yang Sok Pinter", -> [Hook title max 10 words with context and Gen Z style writing]
  },
  {
    "start": "00:05:15", 
    "end": "00:06:30",
    "why": "shocking confession",
    "title": "Artis Ini Bongkar Rahasia Gelap Industry Entertainment Indonesia",
  },
  {
    "start": "00:12:30",
    "end": "00:13:45", 
    "why": "funny reaction",
    "title": "Reaksi Kocak Pas Ditanya Soal Mantan Pacar"
  }
]

`, channelName, videoTitle, srtText)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4Dot1Nano,
			Messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: prompt},
			},
		},
	)

	if err != nil {
		return "", err
	}

	result := resp.Choices[0].Message.Content

	// Clean markdown code blocks if present
	result = cleanJSONResponse(result)

	return result, nil
}

// cleanJSONResponse removes markdown code blocks from AI response
func cleanJSONResponse(s string) string {
	s = strings.TrimSpace(s)

	// Remove ```json and ``` wrapper
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}

	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}

	return strings.TrimSpace(s)
}
