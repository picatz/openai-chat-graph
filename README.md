# openai-chat-graph [![Go Reference](https://pkg.go.dev/badge/github.com/picatz/openai.svg)](https://pkg.go.dev/github.com/picatz/openai-chat-graph) [![Go Report Card](https://goreportcard.com/badge/github.com/picatz/openai-chat-graph)](https://goreportcard.com/report/github.com/picatz/openai-chat-graph) [![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0) 
 
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

This package provides a flexible package to model OpenAI chat 
messages as a [graph](https://en.wikipedia.org/wiki/Graph_theory).
This allows for a variety of use cases, such as summarizing a
thread of messages, modeling and traversing relationships between
messages, or searching messages with language-specific matching.

This helps programatically extend the memory (the context) of a conversation 
an individual bot can have which may leak outside of the token bucket boundaries 
of the current chat API provided by OpenAI.

It can be used to enable short-term memories to select relevant pieces to 
include in a single chat request to form a new idea. You may also serialize
the conversation to disk (or a database) to enable long-term memory.

Messages can be modified, added or removed from the graph in any way that makes 
sense for your application.

```go
// Linking questions and answers together.

lotrFellowshipQuestion := &graph.Message{
	ID: "1",
	ChatMessage: openai.ChatMessage{
		Role:    openai.ChatRoleUser,
		Content: "What are characters part of the fellowship in Lord of the Rings?",
	},
}

lotrFellowshipAnswer := &graph.Message{
	ID: "2",
	ChatMessage: openai.ChatMessage{
		Role:    openai.ChatRoleUser,
		Content: "The Fellowship of the Ring consists of nine members, ...",
	},
}

lotrFellowshipQuestion.AddOut(lotrFellowshipAnswer)

lotrFellowshipHobbitsQuestion := &graph.Message{
	ID: "3",
	ChatMessage: openai.ChatMessage{
		Role:    openai.ChatRoleUser,
		Content: "How do the Hobbits know eachother?",
	},
}

lotrFellowshipAnswer.AddOut(lotrFellowshipHobbitsQuestion)

lotrFellowshipHobbitsAnswer := &graph.Message{
	ID: "4",
	ChatMessage: openai.ChatMessage{
		Role:    openai.ChatRoleUser,
		Content: "Frodo, Sam, Merry, and Pippin are all from the Shire, ...",
	},
}

lotrFellowshipHobbitsQuestion.AddOut(lotrFellowshipHobbitsAnswer)

chat := &graph.Chat{
	ID:   "LOTR",
	Name: "Lord of the Rings",
	Messages: graph.Messages{
		lotrFellowshipQuestion, // The root message of the chat graph.
	},
}

// Visit each message in the graph, depth-first. Print each message's content.
chat.Visit(ctx, func(message *graph.Message) error {
	fmt.Println(message.Content)
	return nil
})

// Resulting connections:
//
// lotrFellowshipQuestion → lotrFellowshipAnswer → lotrFellowshipHobbitsQuestion → lotrFellowshipHobbitsAnswer
```

```go
// Basic (flat) list of messages can also be used at the top-level.
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

summary, _ := chat.Messages.Summarize(ctx, client, openai.ModelGPT4)

fmt.Println(summary)
// Output: The conversation is about Jon Snow's parentage. According to the 
// show, Jon Snow's father is Rhaegar Targaryen and his mother is Lyanna Stark. 
// However, in the books, it is a popular theory that his father is also 
// Rhaegar, making him the true heir to the Iron Throne.


// Search for messages containing the word "father".
results, _ := chat.Messages.Search(ctx, "father")
fmt.Println(len(results))
// Output: 3

// Visit each message in the graph, depth-first.
chat.Visit(ctx, func(message *graph.Message) error {
	// Do something with the message.
	fmt.Println(message.Content)
	return nil
})
```

