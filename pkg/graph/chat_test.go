package graph_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/picatz/openai"
	"github.com/picatz/openai-chat-graph/pkg/graph"
)

func TestChatMessagesSearch(t *testing.T) {
	// Basic chat graph.
	chat := &graph.Chat{
		ID:   "chat-1",
		Name: "Test Chat",
		Messages: graph.Messages{
			{
				ID: "message-1",
				ChatMessage: openai.ChatMessage{
					Role:    openai.ChatRoleUser,
					Content: "Hello World!",
				},
			},
		},
	}

	// Search for a message.
	results := chat.Messages.Search(context.Background(), "world")
	if len(results) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(results))
	}

	// Check the result.
	result := results[0]
	if result.Message.Content != "Hello World!" {
		t.Fatalf("expected message content to be %q, got %q", "Hello World!", result.Message.Content)
	}
}

func TestChatMessagesVisit(t *testing.T) {
	// Basic chat graph.
	chat := &graph.Chat{
		ID:   "chat-1",
		Name: "Test Chat",
		Messages: graph.Messages{
			&graph.Message{
				ID: "message-1",
				ChatMessage: openai.ChatMessage{
					Role:    openai.ChatRoleUser,
					Content: "Hello World!",
				},
			},
		},
	}

	// Visit the chat graph.
	chat.Visit(context.Background(), func(message *graph.Message) error {
		if message.Content != "Hello World!" {
			t.Fatalf("expected message content to be %q, got %q", "Hello World!", message.Content)
		}
		return nil
	})
}

func TestChatMessagesSummarize(t *testing.T) {
	// Basic chat graph.
	chat := &graph.Chat{
		ID:   "chat-1",
		Name: "Test Chat",
		Messages: graph.Messages{
			{
				ID: "1",
				ChatMessage: openai.ChatMessage{
					Role:    openai.ChatRoleUser,
					Content: "Who is Jon Snow's father?",
				},
			},
			{
				ID: "2",
				ChatMessage: openai.ChatMessage{
					Role: openai.ChatRoleAssistant,
					Content: "It is revealed in the show that Jon Snow's father is Rhaegar Targaryen, " +
						"making him a true Targaryen heir. However, in the books, it remains a popular " +
						"theory that his father is also Rhaegar, making him the legitimate heir to the Iron Throne.",
				},
			},
			{
				ID: "3",
				ChatMessage: openai.ChatMessage{
					Role:    openai.ChatRoleUser,
					Content: "What is his mother?",
				},
			},
			{
				ID: "4",
				ChatMessage: openai.ChatMessage{
					Role: openai.ChatRoleAssistant,
					Content: "In the TV show, Jon Snow's mother is revealed to be Lyanna Stark. " +
						"She is the younger sister of Ned Stark, who is Jon Snow's adoptive father. " +
						"In the books, it is strongly suggested that the same is true, but it has not yet been explicitly confirmed.",
				},
			},
		},
	}

	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	// Summarize the chat graph messages.
	summary, err := chat.Messages.Summarize(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}

	// Must contain the following words
	words := []string{
		"Jon Snow",
		"Rhaegar Targaryen",
		"Lyanna Stark",
	}

	for _, word := range words {
		if !strings.Contains(summary, word) {
			t.Logf("summary does not contain %s", word)
			t.Fail()
		}
	}

	t.Logf("summary: %s", summary)

	// Search for a message "father"
	results := chat.Messages.Search(context.Background(), "father")
	if len(results) != 3 {
		t.Fatalf("expected 3 search result, got %d", len(results))
	}

	// Visit the chat graph.
	chat.Visit(context.Background(), func(message *graph.Message) error {
		t.Logf("message: %s", message.Content)
		return nil
	})
}
