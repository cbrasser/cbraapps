
package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	textInput textinput.Model
	words     []string
	score     int
	gameOver  bool
}

var (
	enterKey = key.NewBinding(key.WithKeys("enter"))
	quitKey  = key.NewBinding(key.WithKeys("esc", "ctrl+c"))
)

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Type here..."
	ti.Focus()

	return Model{
		textInput: ti,
		words:     generateWords(),
		score:     0,
		gameOver:  false,
	}
}

func generateWords() []string {
	wordList := []string{
		"go", "tui", "bubbletea", "bubbles",
		"game", "type", "code", "model",
	}

	rand.Seed(time.Now().UnixNano())

	words := make([]string, 5)
	for i := range words {
		words[i] = wordList[rand.Intn(len(wordList))]
	}

	return words
}

func (m *Model) updateGame() {
	if m.gameOver {
		return
	}

	index := -1
	for i, word := range m.words {
		if strings.TrimSpace(m.textInput.Value()) == word {
			index = i
			break
		}
	}

	if index != -1 {
		m.words = append(m.words[:index], m.words[index+1:]...)
		m.score += 10
		m.textInput.SetValue("")
	}

	if len(m.words) == 0 {
		m.words = generateWords()
	}

	if m.score >= 100 {
		m.gameOver = true
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, enterKey):
			if m.gameOver {
				return NewModel(), nil
			}

		case key.Matches(msg, quitKey):
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	m.updateGame()

	return m, cmd
}

func (m Model) View() string {
	if m.gameOver {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFAA")).
			Render(fmt.Sprintf(
				"\n🎮 GAME OVER\nFinal Score: %d\n\nPress ENTER to restart\nPress ESC to quit\n",
				m.score,
			))
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EEEEEE")).
		Render(fmt.Sprintf(
			"🎮 Typing Game\n\nWords:\n%s\n\nScore: %d\n\n%s\n\n(ESC to quit)",
			strings.Join(m.words, ", "),
			m.score,
			m.textInput.View(),
		))
}

func main() {
	p := tea.NewProgram(NewModel())

	if err := p.Start(); err != nil {
		fmt.Println("Error:", err)
	}
}
