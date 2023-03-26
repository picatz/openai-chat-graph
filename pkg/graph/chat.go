package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/picatz/openai"
	"golang.org/x/text/language"
	"golang.org/x/text/search"
)

// Chat is a "chat graph" that contains a connected set of messages.
type Chat struct {
	ID   string
	Name string
	Messages
}

// Visit visits the chat graph in a depth-first-search manner
// and calls the given function for each message. This function is
// useful as a foundation for other graph traversal algorithms.
func (c *Chat) Visit(ctx context.Context, fn func(*Message) error) error {
	seenMsgs := NewMessageSet()

	for _, message := range c.Messages {
		if seenMsgs.Has(message) {
			continue
		}

		if err := VisitMessages(ctx, message, seenMsgs, fn); err != nil {
			return err
		}
	}

	return nil
}

// VisitMessages visits messages in a depth-first-search manner
// and calls the given function for each message. This function is
// useful as a foundation for other graph traversal algorithms.
func VisitMessages(ctx context.Context, message *Message, mset MessageSet, fn func(*Message) error) error {
	// If we've already seen this message, return.
	if mset.Has(message) {
		return nil
	}

	// Mark the message as seen.
	mset.Add(message)

	// Call the function on the current message.
	if err := fn(message); err != nil {
		return err
	}

	// Visit the "out" messages to "drill down" not "up", if any.
	for _, next := range message.Out {
		// If we've already seen this message, skip.
		if mset.Has(next) {
			continue
		}

		if err := VisitMessages(ctx, next, mset, fn); err != nil {
			return err
		}
	}

	// Done.
	return nil
}

// Message is a single chat message that is connected to other messages.
//
// This essentially a small wrapper around openai.ChatMessage to include
// additional functionality for graph traversal, storage, searching, etc.
//
// # In and Out
//
// What it means for other messages to be "in" or "out" is a bit arbitrary,
// and can be used for different purposes that are specific to your application.
//
// For example, in a chat graph, "in" messages are messages that are referenced
// by this message, and "out" messages are messages that reference this
// message. But, in a different application, "in" messages could be
// messages that are "before" this message, and "out" messages could be
// messages that are "after" this message. It all depends on the
// application's requirements.
type Message struct {
	// ID is the unique identifier for the message.
	ID string

	// ChatMessage is the underlying OpenAI chat message, embedded
	// for some convenience to access the underlying fields (e.g. Role, Content).
	openai.ChatMessage

	// In is a collection of messages that are going "in" (←) to this message,
	// (e.g. referencing this message).
	//
	// Example, if this message is a response to another message, the
	// other message could be in the "in" collection.
	In Messages

	// Out is a collection of messages that are going "out" (→) from this message,
	// (e.g. referenced by this message).
	//
	// Example, if this message is a question, the response message could
	// be in the "out" collection.
	Out Messages
}

// AddIn adds a message to the "in" messages.
func (m *Message) AddIn(msg *Message) {
	m.In = append(m.In, msg)
}

// AddOut adds a message to the "out" messages.
func (m *Message) AddOut(msg *Message) {
	m.Out = append(m.Out, msg)
}

// AddInOut adds a message to the "in" messages,
// and adds this message to the "out" messages
// of the other message to create an easily traversalable
// bi-directional graph that is more strongly connected.
func (m *Message) AddInOut(msg *Message) {
	m.In = append(m.In, msg)
	msg.Out = append(msg.Out, m)
}

// AddOutIn adds a message to the "out" messages,
// and adds this message to the "in" messages
// of the other message to create an easily traversalable
// bi-directional graph that is more strongly connected.
func (m *Message) AddOutIn(msg *Message) {
	m.Out = append(m.Out, msg)
	msg.In = append(msg.In, m)
}

// String returns a string representation of the message.
func (m *Message) String() string {
	return fmt.Sprintf("%s: %s", m.Role, m.Content)
}

// Messages is a collection of messages.
type Messages []*Message

// SearchResults is a collection of search results.
type SearchResult struct {
	// The message that matched the search query.
	Message *Message

	// MessageIndex is the index of the message in the chat history.
	MessageIndex int

	// MatchStart is the index of the start of the match in the message.
	StartIndex int

	// MatchEnd is the index of the end of the match in the message.
	EndIndex int
}

// Search searches the messages for matches to a given query.
func (msgs Messages) Search(ctx context.Context, query string) []*SearchResult {
	// Create a new matcher to be compiled into a pattern.
	matcher := search.New(language.AmericanEnglish, search.IgnoreCase)

	// Compile the query into a pattern that can be used to match messages.
	pattern := matcher.CompileString(query)

	// Results retrieved from the search.
	results := []*SearchResult{}

	// Iterate over the messages and collect any matches.
	for i, msg := range msgs {
		msg := msg // Avoid shadowing.

		// If the message matches the pattern, add it to the results.
		if start, end := pattern.IndexString(msg.Content); start != -1 && end != -1 {
			// Add the result.
			results = append(results, &SearchResult{
				Message:      msg,
				MessageIndex: i,
				StartIndex:   start,
				EndIndex:     end,
			})
		}
	}

	// Return the results.
	return results
}

// DefaultSummaryPrompt is the default prompt used to summarize messages for the Summarize method.
var DefaultSummaryPrompt = strings.Join(
	[]string{
		"You are an expert at summarization that answers as concisely as possible.",
		"Provide a summary of the given conversation, including all the key information (e.g. people, places, events, things, etc) to continue on the conversation.",
	}, " ",
)

// Summarize summarizes the messages using the OpenAI API.
func (msgs Messages) Summarize(ctx context.Context, client *openai.Client, model string) (string, error) {
	return msgs.SummarizeWithSystemPrompt(ctx, client, model, DefaultSummaryPrompt)
}

// Summarize summarizes the messages using the OpenAI API.
func (msgs Messages) SummarizeWithSystemPrompt(ctx context.Context, client *openai.Client, model string, summarySystemPrompt string) (string, error) {
	// Create a thread of two messages, using a new system prompt to summarize conversation.
	chatHistory := []openai.ChatMessage{
		{
			Role:    openai.ChatRoleSystem,
			Content: summarySystemPrompt,
		},
		{
			Role: openai.ChatRoleUser,
			Content: func() string {
				var b strings.Builder

				for _, m := range msgs {
					if m.Role == openai.ChatRoleSystem {
						continue // TODO: is this always the right thing to do?
					}
					b.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
				}

				return b.String()
			}(),
		},
	}

	// create a summary of the chat history
	summary, err := client.CreateChat(ctx, &openai.CreateChatRequest{
		Model:    model,
		Messages: chatHistory,
	})

	if err != nil {
		return "", fmt.Errorf("failed to create summary of %d chat messages: %w", len(msgs), err)
	}

	return summary.Choices[0].Message.Content, nil
}

// Visit visits the messages in a depth-first-search manner
// and calls the given function for each message. This function is
// useful as a foundation for other graph traversal algorithms.
func (msgs Messages) Visit(ctx context.Context, fn func(*Message) error) error {
	seenMsgs := NewMessageSet()

	for _, msg := range msgs {
		if seenMsgs.Has(msg) {
			continue
		}

		if err := VisitMessages(ctx, msg, seenMsgs, fn); err != nil {
			return err
		}
	}

	return nil
}

// MessageSet is a collection of messages, used to track seen messages
// when traversing a graph to avoid infinite loops.
type MessageSet map[*Message]struct{}

// NewMessageSet returns a new seen messages collection.
func NewMessageSet() MessageSet {
	return MessageSet{}
}

// Add adds a message to the seen messages.
func (s MessageSet) Add(message *Message) {
	s[message] = struct{}{}
}

// Has returns true if the message has been seen.
func (s MessageSet) Has(message *Message) bool {
	_, ok := s[message]
	return ok
}
