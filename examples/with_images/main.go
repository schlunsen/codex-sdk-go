// Example: send text together with local images.
//
// Usage:
//
//	go run ./examples/with_images ./screenshot.png [./diagram.jpg ...]
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	codex "github.com/schlunsen/codex-sdk-go"
	"github.com/schlunsen/codex-sdk-go/types"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: with_images <image> [<image> ...]")
	}

	client, err := codex.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	inputs := []types.UserInput{types.TextInput("Describe these images and how they relate.")}
	for _, path := range os.Args[1:] {
		inputs = append(inputs, types.LocalImageInput(path))
	}

	thread := client.StartThread(types.NewThreadOptions().WithSkipGitRepoCheck(true))
	turn, err := thread.RunInputs(context.Background(), inputs, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(turn.FinalResponse)
}
