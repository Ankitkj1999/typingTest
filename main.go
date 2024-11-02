package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

type Stats struct {
	startTime    time.Time
	endTime      time.Time
	totalChars   int
	correctChars int
	typedText    string
	targetText   string
}

func loadWordsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			words = append(words, word)
		}
	}
	return words, scanner.Err()
}

func combineWordLists(wordLists ...[]string) []string {
	totalSize := 0
	for _, list := range wordLists {
		totalSize += len(list)
	}

	combined := make([]string, 0, totalSize)
	for _, list := range wordLists {
		combined = append(combined, list...)
	}
	return combined
}

func getWordList() ([]string, error) {
	fmt.Println("Choose word list:")
	fmt.Println("1: Short words")
	fmt.Println("2: Medium words")
	fmt.Println("3: Long words")
	fmt.Println("4: All combined")

	var choice int
	fmt.Print("Enter your choice (1-4): ")
	fmt.Scan(&choice)

	shortWords, err := loadWordsFromFile(filepath.Join("assets", "short-english.txt"))
	if err != nil {
		return nil, fmt.Errorf("error loading short words: %v", err)
	}

	mediumWords, err := loadWordsFromFile(filepath.Join("assets", "medium-english.txt"))
	if err != nil {
		return nil, fmt.Errorf("error loading medium words: %v", err)
	}

	longWords, err := loadWordsFromFile(filepath.Join("assets", "long-english.txt"))
	if err != nil {
		return nil, fmt.Errorf("error loading long words: %v", err)
	}

	switch choice {
	case 1:
		return shortWords, nil
	case 2:
		return mediumWords, nil
	case 3:
		return longWords, nil
	case 4:
		return combineWordLists(shortWords, mediumWords, longWords), nil
	default:
		return nil, fmt.Errorf("invalid choice: %d", choice)
	}
}

func getWordCount() (int, error) {
	fmt.Print("Enter number of words to type (5-50): ")
	var count int
	fmt.Scan(&count)
	
	if count < 5 || count > 50 {
		return 0, fmt.Errorf("word count must be between 5 and 50")
	}
	return count, nil
}

func generateText(words []string, wordCount int) string {
	rand.Seed(time.Now().UnixNano())
	var result []string
	for i := 0; i < wordCount; i++ {
		result = append(result, words[rand.Intn(len(words))])
	}
	return strings.Join(result, " ")
}

func calculateLiveStats(stats Stats) (wpm float64, cpm float64, accuracy float64) {
	duration := time.Since(stats.startTime).Minutes()
	if duration < 0.017 { // Less than 1 second (1/60 minute)
		return 0, 0, 0
	}
	
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

func drawText(stats Stats, showCursor bool) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	x, y := 2, 2

	// Draw header
	header := "Type the following text:"
	for i, char := range header {
		termbox.SetCell(x+i, y-1, char, termbox.ColorWhite, termbox.ColorDefault)
	}

	// Draw target text and typed text
	for i, char := range stats.targetText {
		color := termbox.ColorWhite | termbox.AttrDim
		bgColor := termbox.ColorDefault

		if i < len(stats.typedText) {
			if string(stats.typedText[i]) == string(char) {
				color = termbox.ColorYellow
			} else {
				color = termbox.ColorRed
			}
		} else if i == len(stats.typedText) {
			if showCursor {
				color = termbox.ColorWhite
			} else {
				color = termbox.ColorWhite | termbox.AttrDim
			}
		}

		termbox.SetCell(x+i, y, char, color, bgColor)
	}

	// Calculate and draw live stats
	wpm, cpm, accuracy := calculateLiveStats(stats)
	statsText := fmt.Sprintf("WPM: %.1f | CPM: %.1f | Accuracy: %.1f%%", wpm, cpm, accuracy)
	for i, char := range statsText {
		termbox.SetCell(x+i, y+2, char, termbox.ColorCyan, termbox.ColorDefault)
	}

	termbox.Flush()
}
func main() {
	// Get word list choice from user
	words, err := getWordList()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Get word count from user
	wordCount, err := getWordCount()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Clear the screen before starting the game
	fmt.Print("\033[H\033[2J")
	fmt.Println("Starting typing test... Press any key to begin!")
	fmt.Scanln() // Wait for user input

	err = termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	stats := Stats{
		targetText: generateText(words, wordCount),
		typedText:  "",
		startTime:  time.Now(),
	}

	// Create a ticker for cursor blinking
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Create a ticker for stats updates
	statsTicker := time.NewTicker(100 * time.Millisecond)
	defer statsTicker.Stop()

	showCursor := true
	drawText(stats, showCursor)

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

			drawText(stats, showCursor)

			if len(stats.typedText) >= len(stats.targetText) {
				break mainloop
			}

		case <-ticker.C:
			showCursor = !showCursor
			drawText(stats, showCursor)
			
		case <-statsTicker.C:
			drawText(stats, showCursor)
		}
	}

	stats.endTime = time.Now()
	wpm, cpm, accuracy := calculateLiveStats(stats)

	termbox.Close()
	fmt.Printf("\nFinal Results:\n")
	fmt.Printf("WPM: %.1f\n", wpm)
	fmt.Printf("CPM: %.1f\n", cpm)
	fmt.Printf("Accuracy: %.1f%%\n", accuracy)
}