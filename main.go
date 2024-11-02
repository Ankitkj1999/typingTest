package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
	// syscall
	"syscall"

	"github.com/nsf/termbox-go"
)

type Stats struct {
	startTime    time.Time
	endTime      time.Time
	totalChars   int
	correctChars int
	typedText    string
	targetText   string
	isInfinite   bool
	words        []string  // Store word list for infinite mode
}
// Added structure to handle wrapped text display
type WrappedText struct {
	lines    []string
	charPos  map[int]Position // Maps absolute position to line and column
	totalLen int
}

type Position struct {
	line   int
	column int
}

func wrapText(text string, maxWidth int) WrappedText {
	words := strings.Split(text, " ")
	var lines []string
	var currentLine string
	charPos := make(map[int]Position)
	absolutePos := 0

	currentLineNum := 0
	currentColumn := 0

	for i, word := range words {
		// Check if adding this word would exceed the width
		var proposedLine string
		if currentLine == "" {
			proposedLine = word
		} else {
			proposedLine = currentLine + " " + word
		}

		if utf8.RuneCountInString(proposedLine) <= maxWidth {
			// Add the space before the word (except for first word)
			if currentLine != "" {
				// Map position for the space
				charPos[absolutePos] = Position{line: currentLineNum, column: currentColumn}
				absolutePos++
				currentColumn++
				currentLine += " "
			}

			// Map positions for each character in the word
			for range word {
				charPos[absolutePos] = Position{line: currentLineNum, column: currentColumn}
				absolutePos++
				currentColumn++
			}
			currentLine = proposedLine
		} else {
			// Line is full, start a new line
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
				currentLineNum++
				currentColumn = 0

				// Map positions for the new word
				for range word {
					charPos[absolutePos] = Position{line: currentLineNum, column: currentColumn}
					absolutePos++
					currentColumn++
				}
			}
		}

		// Handle the last word
		if i == len(words)-1 && currentLine != "" {
			lines = append(lines, currentLine)
		}
	}

	return WrappedText{
		lines:    lines,
		charPos:  charPos,
		totalLen: absolutePos,
	}
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

func getWordCount() (int, bool, error) {
	fmt.Println("\nSelect mode:")
	fmt.Println("1: Fixed number of words")
	fmt.Println("2: Infinite mode (type until you exit)")
	
	var choice int
	fmt.Print("Enter your choice (1-2): ")
	fmt.Scan(&choice)

	if choice == 2 {
		return 100, true, nil // Start with 100 words for infinite mode
	}

	fmt.Print("Enter number of words to type (5-50): ")
	var count int
	fmt.Scan(&count)

	if count < 5 || count > 50 {
		return 0, false, fmt.Errorf("word count must be between 5 and 50")
	}
	return count, false, nil
}

func generateText(words []string, wordCount int) string {
	rand.Seed(time.Now().UnixNano())
	var result []string
	for i := 0; i < wordCount; i++ {
		result = append(result, words[rand.Intn(len(words))])
	}
	return strings.Join(result, " ")
}

// New function to add more words to the target text
func (s *Stats) extendText() {
	rand.Seed(time.Now().UnixNano())
	var newWords []string
	for i := 0; i < 50; i++ { // Add 50 more words
		newWords = append(newWords, s.words[rand.Intn(len(s.words))])
	}
	
	if s.targetText != "" {
		s.targetText += " " // Add space between old and new text
	}
	s.targetText += strings.Join(newWords, " ")
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

	// Get terminal size
	width, height := termbox.Size()
	maxWidth := width - 4 // Leave some margin

	// Draw header
	header := "Type the following text (ESC or Ctrl+C to exit):"
	headerX := 2
	headerY := 1
	for i, char := range header {
		termbox.SetCell(headerX+i, headerY, char, termbox.ColorWhite, termbox.ColorDefault)
	}

	// Wrap the target text
	wrapped := wrapText(stats.targetText, maxWidth)

	// Calculate the visible area
	visibleLines := height - 5 // Reserve space for header and stats
	startY := 3                // Starting Y position for text

	// Draw the wrapped text
	for lineNum, line := range wrapped.lines {
		if lineNum >= visibleLines {
			break
		}

		currentX := 2
		for i, char := range line {
			color := termbox.ColorWhite | termbox.AttrDim
			bgColor := termbox.ColorDefault

			// Find the absolute position for this character
			absPos := -1
			for pos, coord := range wrapped.charPos {
				if coord.line == lineNum && coord.column == i {
					absPos = pos
					break
				}
			}

			if absPos != -1 {
				if absPos < len(stats.typedText) {
					if string(stats.typedText[absPos]) == string(char) {
						color = termbox.ColorYellow
					} else {
						color = termbox.ColorRed
					}
				} else if absPos == len(stats.typedText) {
					if showCursor {
						color = termbox.ColorWhite
					} else {
						color = termbox.ColorWhite | termbox.AttrDim
					}
				}
			}

			termbox.SetCell(currentX, startY+lineNum, char, color, bgColor)
			currentX++
		}
	}

	// Draw stats at the bottom of the visible area
	wpm, cpm, accuracy := calculateLiveStats(stats)
	statsText := fmt.Sprintf("WPM: %.1f | CPM: %.1f | Accuracy: %.1f%%", wpm, cpm, accuracy)
	statsY := height - 1 // Place stats at the bottom
	for i, char := range statsText {
		if i < width-2 { // Prevent stats from going off-screen
			termbox.SetCell(2+i, statsY, char, termbox.ColorCyan, termbox.ColorDefault)
		}
	}

	termbox.Flush()
}

// func main() {
// 	// Get word list choice from user
// 	words, err := getWordList()
// 	if err != nil {
// 		fmt.Printf("Error: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Get word count and mode from user
// 	wordCount, isInfinite, err := getWordCount()
// 	if err != nil {
// 		fmt.Printf("Error: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Clear the screen before starting the game
// 	fmt.Print("\033[H\033[2J")
// 	if isInfinite {
// 		fmt.Println("Starting infinite typing test... Press Enter to begin!")
// 	} else {
// 		fmt.Println("Starting typing test... Press Enter to begin!")
// 	}
// 	fmt.Println("Press ESC or Ctrl+C to exit")
// 	fmt.Scanln()// Wait for user to press Enter

// 	err = termbox.Init()
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer termbox.Close()

// 	stats := Stats{
// 		targetText: generateText(words, wordCount),
// 		typedText:  "",
// 		startTime:  time.Now(),
// 	}

// 	// Create a ticker for cursor blinking
// 	ticker := time.NewTicker(500 * time.Millisecond)
// 	defer ticker.Stop()

// 	// Create a ticker for stats updates
// 	statsTicker := time.NewTicker(100 * time.Millisecond)
// 	defer statsTicker.Stop()

// 	showCursor := true
// 	drawText(stats, showCursor)

// 	// Setup signal handling for Ctrl+C
// 	sigChan := make(chan os.Signal, 1)
// 	signal.Notify(sigChan, os.Interrupt)

// 	// Channel for handling keyboard events
// 	eventQueue := make(chan termbox.Event)
// 	go func() {
// 		for {
// 			eventQueue <- termbox.PollEvent()
// 		}
// 	}()

// mainloop:
// 	for {
// 		select {
// 		case <-sigChan:
// 			// Handle Ctrl+C
// 			break mainloop

// 		case ev := <-eventQueue:
// 			if ev.Type == termbox.EventError {
// 				panic(ev.Err)
// 			}
// 			if ev.Type == termbox.EventResize {
// 				drawText(stats, showCursor)
// 				continue
// 			}
// 			if ev.Type != termbox.EventKey {
// 				continue
// 			}

// 			if ev.Key == termbox.KeyEsc {
// 				break mainloop
// 			}

// 			if ev.Key == termbox.KeyEnter && len(stats.typedText) >= len(stats.targetText) {
// 				break mainloop
// 			}

// 			if ev.Key == termbox.KeyBackspace || ev.Key == termbox.KeyBackspace2 {
// 				if len(stats.typedText) > 0 {
// 					stats.typedText = stats.typedText[:len(stats.typedText)-1]
// 					stats.totalChars--
// 					if len(stats.typedText) < len(stats.targetText) &&
// 						string(stats.typedText[len(stats.typedText)-1]) == string(stats.targetText[len(stats.typedText)-1]) {
// 						stats.correctChars--
// 					}
// 				}
// 			} else {
// 				var charToAdd string
// 				if ev.Key == termbox.KeySpace {
// 					charToAdd = " "
// 				} else if ev.Ch != 0 {
// 					charToAdd = string(ev.Ch)
// 				}

// 				if charToAdd != "" && len(stats.typedText) < len(stats.targetText) {
// 					stats.typedText += charToAdd
// 					stats.totalChars++

// 					if string(stats.targetText[len(stats.typedText)-1]) == charToAdd {
// 						stats.correctChars++
// 					}
// 				}
// 			}

// 			drawText(stats, showCursor)

// 			if len(stats.typedText) >= len(stats.targetText) {
// 				break mainloop
// 			}

// 		case <-ticker.C:
// 			showCursor = !showCursor
// 			drawText(stats, showCursor)

// 		case <-statsTicker.C:
// 			drawText(stats, showCursor)
// 		}
// 	}

// 	stats.endTime = time.Now()
// 	wpm, cpm, accuracy := calculateLiveStats(stats)

// 	termbox.Close()
// 	completed := float64(len(stats.typedText)) / float64(len(stats.targetText)) * 100
// 	fmt.Printf("\nTyping Test Results (%.1f%% completed):\n", completed)
// 	fmt.Printf("WPM: %.1f\n", wpm)
// 	fmt.Printf("CPM: %.1f\n", cpm)
// 	fmt.Printf("Accuracy: %.1f%%\n", accuracy)
// }


func main() {
	// Get word list choice from user
	words, err := getWordList()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Get word count and mode from user
	wordCount, isInfinite, err := getWordCount()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Clear the screen before starting the game
	fmt.Print("\033[H\033[2J")
	if isInfinite {
		fmt.Println("Starting infinite typing test... Press Enter to begin!")
	} else {
		fmt.Println("Starting typing test... Press Enter to begin!")
	}
	fmt.Println("Press ESC or Ctrl+C to exit")
	fmt.Scanln()

	err = termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	stats := Stats{
		targetText:  generateText(words, wordCount),
		typedText:   "",
		startTime:   time.Now(),
		isInfinite:  isInfinite,
		words:       words,
	}

	// Create a ticker for cursor blinking
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Create a ticker for stats updates
	statsTicker := time.NewTicker(100 * time.Millisecond)
	defer statsTicker.Stop()

	// Setup signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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
		case <-sigChan:
			break mainloop

		case ev := <-eventQueue:
			if ev.Type == termbox.EventError {
				panic(ev.Err)
			}
			if ev.Type == termbox.EventResize {
				drawText(stats, showCursor)
				continue
			}
			if ev.Type != termbox.EventKey {
				continue
			}

			if ev.Key == termbox.KeyEsc {
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

					// In infinite mode, add more words when approaching the end
					if stats.isInfinite && len(stats.typedText) >= len(stats.targetText)-100 {
						stats.extendText()
						drawText(stats, showCursor)
					}
				}
			}

			drawText(stats, showCursor)

			// Don't break the loop in infinite mode when reaching the end
			if !stats.isInfinite && len(stats.typedText) >= len(stats.targetText) {
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
	
	// Show results
	if stats.isInfinite {
		fmt.Printf("\nInfinite Mode Results:\n")
		fmt.Printf("Total Words Typed: %.0f\n", float64(stats.correctChars)/5.0)
	} else {
		completed := float64(len(stats.typedText)) / float64(len(stats.targetText)) * 100
		fmt.Printf("\nTyping Test Results (%.1f%% completed):\n", completed)
	}
	fmt.Printf("WPM: %.1f\n", wpm)
	fmt.Printf("CPM: %.1f\n", cpm)
	fmt.Printf("Accuracy: %.1f%%\n", accuracy)
}

// TODO: Add an toption to restart the test

