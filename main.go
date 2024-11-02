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

func drawText(targetText, typedText string) {
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
		}

		termbox.SetCell(x+i, y, char, color, bgColor)
	}

	// Draw cursor
	if len(typedText) < len(targetText) {
		cursorChar := '│'
		if len(typedText) < len(targetText) {
			termbox.SetCell(x+len(typedText), y, cursorChar, termbox.ColorWhite, termbox.ColorDefault)
		}
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

	drawText(stats.targetText, stats.typedText)

	stats.startTime = time.Now()
	stats.totalChars = 0
	stats.correctChars = 0

mainloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
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
				// Handle both regular characters and space
				var charToAdd string
				if ev.Key == termbox.KeySpace {
					charToAdd = " "
				} else if ev.Ch != 0 {
					charToAdd = string(ev.Ch)
				}

				if charToAdd != "" && len(stats.typedText) < len(stats.targetText) {
					stats.typedText += charToAdd
					stats.totalChars++
					
					// Check if the character matches the target
					if string(stats.targetText[len(stats.typedText)-1]) == charToAdd {
						stats.correctChars++
					}
				}
			}

			drawText(stats.targetText, stats.typedText)

			if len(stats.typedText) >= len(stats.targetText) {
				break mainloop
			}

		case termbox.EventError:
			panic(ev.Err)
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