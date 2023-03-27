package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/picatz/openai"
	"golang.org/x/text/language"
	"golang.org/x/text/search"
)

// Chat is a "chat graph" that contains a connected set of messages.
type Chat struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Messages `json:"messages"`
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
	ID string `json:"id,omitempty"`

	// ChatMessage is the underlying OpenAI chat message, embedded
	// for some convenience to access the underlying fields (e.g. Role, Content).
	openai.ChatMessage

	// In is a collection of messages that are going "in" (←) to this message,
	// (e.g. referencing this message).
	//
	// Example, if this message is a response to another message, the
	// other message could be in the "in" collection.
	In Messages `json:"in,omitempty"`

	// Out is a collection of messages that are going "out" (→) from this message,
	// (e.g. referenced by this message).
	//
	// Example, if this message is a question, the response message could
	// be in the "out" collection.
	Out Messages `json:"out,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface for Message,
// which is like the normal json.Marshal, but only includes message IDs
// for the "in" and "out" collections, to reduce the size of the JSON.
func (m *Message) MarshalJSON() ([]byte, error) {
	// Using fmt.Sprintf instead of json.Marshal to avoid
	// an infinite loop, and to avoid marshalling a another struct.
	return []byte(
		fmt.Sprintf(
			`{"id":"%s","role":"%s","content":"%s","in":[%s],"out":[%s]}`,
			m.ID,
			m.Role,
			m.Content,
			strings.Join(m.In.IDs(), ","),
			strings.Join(m.Out.IDs(), ","),
		),
	), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for Message,
// partially unmarshalling the "in" and "out" messages, and leaving the
// rest to the caller to do, if needed.
//
// This can be done at the message set or the graph level.
func (m *Message) UnmarshalJSON(b []byte) error {
	// Using json.Unmarshal instead of fmt.Sprintf to avoid
	// an infinite loop, and to avoid unmarshalling a another struct.
	var raw struct {
		ID      string   `json:"id"`
		Role    string   `json:"role"`
		Content string   `json:"content"`
		In      []string `json:"in"`
		Out     []string `json:"out"`
	}

	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	m.ID = raw.ID
	m.Role = raw.Role
	m.Content = raw.Content

	// Parially unmarshal the "in" messages.
	for _, id := range raw.In {
		m.In = append(m.In, &Message{ID: id})
	}

	// Parially unmarshal the "out" messages.
	for _, id := range raw.Out {
		m.Out = append(m.Out, &Message{ID: id})
	}

	return nil
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

// OpenAIChatMessages returns a slice of OpenAI chat messages.
func (msgs Messages) OpenAIChatMessages() []openai.ChatMessage {
	chatMsgs := make([]openai.ChatMessage, len(msgs))
	for i, msg := range msgs {
		chatMsgs[i] = msg.ChatMessage
	}
	return chatMsgs
}

// Match returns a slice of messages that match the given predicate function.
func (msgs Messages) Match(prFn func(*Message) bool) Messages {
	matches := Messages{}
	for _, msg := range msgs {
		if prFn(msg) {
			matches = append(matches, msg)
		}
	}
	return matches
}

// IDs returns a slice of message IDs.
func (msgs Messages) IDs() []string {
	ids := make([]string, len(msgs))
	for i, msg := range msgs {
		ids[i] = msg.ID
	}
	return ids
}

// GetByID returns a message by ID (first match).
func (msgs Messages) GetByID(id string) *Message {
	for _, msg := range msgs {
		if msg.ID == id {
			return msg
		}
	}
	return nil
}

// Hydrate fully hydrates the messages by adding the "in" and "out"
// messages to the message collections instead of just the message IDs.
func (msgs Messages) Hydrate(ctx context.Context, graph *Chat) {
	for _, msg := range msgs {
		msg.In = graph.GetMessages(msg.In.IDs()...)
		msg.Out = graph.GetMessages(msg.Out.IDs()...)
	}
}

// Hydrated returns true if the messages are fully hydrated.
func (msgs Messages) Hydrated() bool {
	for _, msg := range msgs {
		for _, in := range msg.In {
			if in.ID == "" {
				return false
			}
			if in.Content == "" && in.Role == "" {
				return false
			}
		}
	}
	return true
}

// GetMessages returns a collection of messages by ID for the graph.
func (graph *Chat) GetMessages(ids ...string) Messages {
	msgs := make(Messages, len(ids))
	for _, msg := range graph.Messages {
		for _, id := range ids {
			if msg.ID == id {
				msgs = append(msgs, msg)
			}
		}
	}
	return msgs
}

// GetMessageByID returns a message by ID (first match) for the graph.
func (graph *Chat) GetMessageByID(id string) *Message {
	for _, msg := range graph.Messages {
		if msg.ID == id {
			return msg
		}
	}
	return nil
}

// HydrateMessages fully hydrates the messages by adding the "in" and "out"
// messages to the message collections instead of just the message IDs.
//
// This only need to be called when loaded from a serialized graph,
// since nested message collections are not fully serialized, only
// the message IDs.
func (graph *Chat) HydrateMessages(ctx context.Context) {
	graph.Messages.Hydrate(ctx, graph)
}

// SearchResults is a collection of search results.
type SearchResult struct {
	// The message that matched the search query.
	Message *Message `json:"message"`

	// MessageIndex is the index of the message in the chat history.
	MessageIndex int `json:"message_index"`

	// MatchStart is the index of the start of the match in the message.
	StartIndex int `json:"start_index"`

	// MatchEnd is the index of the end of the match in the message.
	EndIndex int `json:"end_index"`
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
		"Do not include any unnecessary information, or a prefix in the output.",
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

// GetOrPut returns the message if it has been seen, or adds it to the
// seen messages and returns it. This is useful for adding a message
// to the seen messages collection and returning it to be used in
// a graph traversal algorithm. It is a convenience function that
// combines the Add and Has functions into one.
func (s MessageSet) GetOrPut(message *Message) *Message {
	if s.Has(message) {
		return message
	}

	s.Add(message)
	return message
}
