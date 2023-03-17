# openai-chat-graph
 
Model OpenAI chat messages as a graph.

## Features

- Summarize a thread of messages.
- Model and traverse relationships between messages, within branches and threads.
- Search messages with language-specific matching.

## Installation

```console
$ go get github.com/picatz/openai-chat-graph
```

## Usage

```go
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

summary, _ := chat.Messages.Summarize(ctx, client)

fmt.Println(summary)
// Output: The conversation is about Jon Snow's parentage. According to the 
// show, Jon Snow's father is Rhaegar Targaryen and his mother is Lyanna Stark. 
// However, in the books, it is a popular theory that his father is also 
// Rhaegar, making him the true heir to the Iron Throne.
```

```go
results, _ := chat.Messages.Search(ctx, "father")

fmt.Println(len(results))
// Output: 3
```

```go
// Visit each message in the graph, depth-first.
chat.Visit(ctx, func(message *graph.Message) error {
	// Do something with the message.
    fmt.Println(message.Content)
	return nil
})
```

