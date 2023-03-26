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
	t.Run("basic", func(t *testing.T) {
		// Basic chat graph with a single message.
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
	})

	t.Run("multiple", func(t *testing.T) {
		// Basic chat graph with multiple messages.
		chat := &graph.Chat{
			ID:   "chat-1",
			Name: "Test Chat",
			Messages: graph.Messages{
				&graph.Message{
					ID: "message-1",
					ChatMessage: openai.ChatMessage{
						Role:    openai.ChatRoleUser,
						Content: "Hello World from 1!",
					},
				},
				&graph.Message{
					ID: "message-2",
					ChatMessage: openai.ChatMessage{
						Role:    openai.ChatRoleUser,
						Content: "Hello World from 2!",
					},
				},
			},
		}

		count := 0

		// Visit the chat graph.
		chat.Visit(context.Background(), func(message *graph.Message) error {
			if !strings.Contains(message.Content, "Hello World") {
				t.Fatalf("expected message content to contain %q, got %q", "Hello World", message.Content)
			}

			count++

			return nil
		})

		if count != 2 {
			t.Fatalf("expected 2 messages to be visited, got %d", count)
		}
	})

	t.Run("shallow", func(t *testing.T) {
		m1 := &graph.Message{
			ID: "message-1",
			ChatMessage: openai.ChatMessage{
				Role:    openai.ChatRoleUser,
				Content: "a",
			},
		}

		m2 := &graph.Message{
			ID: "message-2",
			ChatMessage: openai.ChatMessage{
				Role:    openai.ChatRoleUser,
				Content: "b",
			},
		}

		m1.AddOut(m2)

		chat := &graph.Chat{
			ID:   "chat-1",
			Name: "Test Chat",
			Messages: graph.Messages{
				m1,
			},
		}

		count := 0

		chat.Visit(context.Background(), func(message *graph.Message) error {
			switch message.ID {
			case "message-1":
				if message.Content != "a" {
					t.Fatalf("expected message content to be %q, got %q", "a", message.Content)
				}
			case "message-2":
				if message.Content != "b" {
					t.Fatalf("expected message content to be %q, got %q", "b", message.Content)
				}
			default:
				t.Fatalf("unexpected message id %q", message.ID)
			}

			count++

			return nil
		})
	})

	t.Run("deep", func(t *testing.T) {
		m1 := &graph.Message{
			ID: "message-1",
			ChatMessage: openai.ChatMessage{
				Role:    openai.ChatRoleUser,
				Content: "a",
			},
		}

		m2 := &graph.Message{
			ID: "message-2",
			ChatMessage: openai.ChatMessage{
				Role:    openai.ChatRoleAssistant,
				Content: "b",
			},
		}

		m3 := &graph.Message{
			ID: "message-3",
			ChatMessage: openai.ChatMessage{
				Role:    openai.ChatRoleUser,
				Content: "c",
			},
		}

		m4 := &graph.Message{
			ID: "message-4",
			ChatMessage: openai.ChatMessage{
				Role:    openai.ChatRoleAssistant,
				Content: "d",
			},
		}

		m1.AddOut(m2)
		m2.AddOut(m3)
		m3.AddOut(m4)

		chat := &graph.Chat{
			ID:   "chat-1",
			Name: "Test Chat",
			Messages: graph.Messages{
				m1,
			},
		}

		count := 0

		chat.Visit(context.Background(), func(message *graph.Message) error {
			switch message.ID {
			case "message-1":
				if message.Content != "a" {
					t.Fatalf("expected message content to be %q, got %q", "a", message.Content)
				}
			case "message-2":
				if message.Content != "b" {
					t.Fatalf("expected message content to be %q, got %q", "b", message.Content)
				}
			case "message-3":
				if message.Content != "c" {
					t.Fatalf("expected message content to be %q, got %q", "c", message.Content)
				}
			case "message-4":
				if message.Content != "d" {
					t.Fatalf("expected message content to be %q, got %q", "d", message.Content)
				}
			default:
				t.Fatalf("unexpected message id %q", message.ID)
			}

			count++

			return nil
		})
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
	summary, err := chat.Messages.Summarize(context.Background(), client, openai.ModelGPT4) // TODO: use OPENAI_MODEL environment variable
	if err != nil {
		t.Fatal(err)
	}

	// Must contain the following words
	words := []string{
		"Jon Snow",
		"Rhaegar Targaryen",
		"Lyanna Stark",
		"father",
		"mother",
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
