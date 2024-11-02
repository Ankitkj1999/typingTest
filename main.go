package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

var words = []string{
	"the", "be", "to", "of", "and", "a", "in", "that", "have", "I",
	"it", "for", "not", "on", "with", "he", "as", "you", "do", "at",
	"this", "but", "his", "by", "from", "they", "we", "say", "her", "she",
}

type Stats struct {
	startTime    time.Time
	endTime      time.Time
	totalChars   int
	correctChars int
	typedText    string
	targetText   string
}

func generateText() string {
	rand.Seed(time.Now().UnixNano())
	var result []string
	for i := 0; i < 15; i++ {
		result = append(result, words[rand.Intn(len(words))])
	}
	return strings.Join(result, " ")
}

func drawText(targetText, typedText string, showCursor bool) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	x, y := 2, 2

	// Draw header
	header := "Type the following text:"
	for i, char := range header {
		termbox.SetCell(x+i, y-1, char, termbox.ColorWhite, termbox.ColorDefault)
	}

	// Draw target text and typed text
	for i, char := range targetText {
		color := termbox.ColorWhite | termbox.AttrDim
		bgColor := termbox.ColorDefault

		if i < len(typedText) {
			if string(typedText[i]) == string(char) {
				color = termbox.ColorYellow
			} else {
				color = termbox.ColorRed
			}
		} else if i == len(typedText) {
			// Current character to type
			if showCursor {
				color = termbox.ColorWhite
			} else {
				color = termbox.ColorWhite | termbox.AttrDim
			}
		}

		termbox.SetCell(x+i, y, char, color, bgColor)
	}

	termbox.Flush()
}

func calculateStats(stats Stats) (wpm float64, cpm float64, accuracy float64) {
	duration := stats.endTime.Sub(stats.startTime).Minutes()
	
	// Calculate WPM (assuming average word length of 5 characters)
	words := float64(stats.correctChars) / 5.0
	wpm = words / duration

	// Calculate CPM
	cpm = float64(stats.correctChars) / duration

	// Calculate accuracy
	if stats.totalChars > 0 {
		accuracy = float64(stats.correctChars) / float64(stats.totalChars) * 100
	}

	return wpm, cpm, accuracy
}

func main() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	stats := Stats{
		targetText: generateText(),
		typedText:  "",
	}

	// Create a ticker for cursor blinking
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	showCursor := true
	drawText(stats.targetText, stats.typedText, showCursor)

	stats.startTime = time.Now()
	stats.totalChars = 0
	stats.correctChars = 0

	// Channel for handling keyboard events
	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()

mainloop:
	for {
		select {
		case ev := <-eventQueue:
			if ev.Type == termbox.EventError {
				panic(ev.Err)
			}
			if ev.Type != termbox.EventKey {
				continue
			}

			if ev.Key == termbox.KeyEsc {
				break mainloop
			}

			if ev.Key == termbox.KeyEnter && len(stats.typedText) >= len(stats.targetText) {
				break mainloop
			}

			if ev.Key == termbox.KeyBackspace || ev.Key == termbox.KeyBackspace2 {
				if len(stats.typedText) > 0 {
					stats.typedText = stats.typedText[:len(stats.typedText)-1]
					stats.totalChars--
					if len(stats.typedText) < len(stats.targetText) &&
						string(stats.typedText[len(stats.typedText)-1]) == string(stats.targetText[len(stats.typedText)-1]) {
						stats.correctChars--
					}
				}
			} else {
				var charToAdd string
				if ev.Key == termbox.KeySpace {
					charToAdd = " "
				} else if ev.Ch != 0 {
					charToAdd = string(ev.Ch)
				}

				if charToAdd != "" && len(stats.typedText) < len(stats.targetText) {
					stats.typedText += charToAdd
					stats.totalChars++
					
					if string(stats.targetText[len(stats.typedText)-1]) == charToAdd {
						stats.correctChars++
					}
				}
			}

			drawText(stats.targetText, stats.typedText, showCursor)

			if len(stats.typedText) >= len(stats.targetText) {
				break mainloop
			}

		case <-ticker.C:
			showCursor = !showCursor
			drawText(stats.targetText, stats.typedText, showCursor)
		}
	}

	stats.endTime = time.Now()
	wpm, cpm, accuracy := calculateStats(stats)

	termbox.Close()
	fmt.Printf("\nTyping Test Results:\n")
	fmt.Printf("WPM: %.2f\n", wpm)
	fmt.Printf("CPM: %.2f\n", cpm)
	fmt.Printf("Accuracy: %.2f%%\n", accuracy)
}
//  TODO: The cursor is hindind the character, need to fix it