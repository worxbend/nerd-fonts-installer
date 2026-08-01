package tui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/worxbend/nerd-fonts-installer/internal/nerdfonts"
)

func TestModelSelectsReleaseAndFamilies(t *testing.T) {
	m := newModel([]nerdfonts.Release{
		{
			Name:     "v3.4.0",
			TagName:  "v3.4.0",
			Families: []string{"Hack", "JetBrainsMono"},
		},
	}, "/tmp/fonts", true, IconAuto)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = requireModel(t, next)
	if m.step != stepFamilies {
		t.Fatalf("step = %v, want stepFamilies", m.step)
	}
	if m.selectedRelease.TagName != "v3.4.0" {
		t.Fatalf("selectedRelease = %q", m.selectedRelease.TagName)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = requireModel(t, next)
	if m.selectedCount() != 1 {
		t.Fatalf("selectedCount() = %d, want 1", m.selectedCount())
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = requireModel(t, next)
	if m.selectedCount() != 2 {
		t.Fatalf("selectedCount() = %d, want 2", m.selectedCount())
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = requireModel(t, next)
	if m.selectedCount() != 0 {
		t.Fatalf("selectedCount() = %d, want 0", m.selectedCount())
	}
}

func TestFamilyItemsUseNerdFontCheckboxesAndIcons(t *testing.T) {
	m := newModel([]nerdfonts.Release{
		{
			Name:     "v3.4.0",
			TagName:  "v3.4.0",
			Families: []string{"Hack", "JetBrainsMono"},
		},
	}, "/tmp/fonts", true, IconNerd)
	m.selectedRelease = m.releases[0]
	m.selectedFamilies = map[string]bool{"Hack": true}

	items := m.familyItems()
	hack := requireItem(t, items[0])
	jetBrains := requireItem(t, items[1])

	if !strings.Contains(hack.title, "󰄲") {
		t.Fatalf("Hack title = %q, want checked box", hack.title)
	}
	if !strings.Contains(hack.title, "󰌌") {
		t.Fatalf("Hack title = %q, want Hack icon", hack.title)
	}
	if !strings.Contains(jetBrains.title, "󰄱") {
		t.Fatalf("JetBrainsMono title = %q, want unchecked box", jetBrains.title)
	}
	if !strings.Contains(jetBrains.title, "") {
		t.Fatalf("JetBrainsMono title = %q, want JetBrains icon", jetBrains.title)
	}
}

func TestFamilyItemsDefaultToUnicodeIcons(t *testing.T) {
	m := newModel([]nerdfonts.Release{
		{
			Name:     "v3.4.0",
			TagName:  "v3.4.0",
			Families: []string{"Hack"},
		},
	}, "/tmp/fonts", true, IconAuto)
	m.selectedRelease = m.releases[0]

	items := m.familyItems()
	hack := requireItem(t, items[0])

	if !strings.Contains(hack.title, "☐") {
		t.Fatalf("Hack title = %q, want unicode checkbox", hack.title)
	}
	if strings.Contains(hack.title, "󰌌") {
		t.Fatalf("Hack title = %q, should not require Nerd Font glyphs by default", hack.title)
	}
}

func TestModelHandlesWindowSizeBeforeFamilyListExists(t *testing.T) {
	m := newModel([]nerdfonts.Release{
		{
			Name:     "v3.4.0",
			TagName:  "v3.4.0",
			Families: []string{"Hack"},
		},
	}, "/tmp/fonts", true, IconAuto)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 96, Height: 12})
	if _, ok := next.(model); !ok {
		t.Fatalf("Update() = %T, want model", next)
	}
}

// TestViewFitsTerminalHeight guards the layout arithmetic in listSize: a frame
// even one row taller than the terminal makes Bubble Tea's renderer truncate the
// top of the view, which silently eats the banner's top border.
func TestViewFitsTerminalHeight(t *testing.T) {
	families := make([]string, 40)
	for i := range families {
		families[i] = "Family" + strconv.Itoa(i)
	}
	releases := []nerdfonts.Release{
		{Name: "v3.4.0", TagName: "v3.4.0", Families: families},
		{Name: "v3.3.0", TagName: "v3.3.0", Families: families},
	}

	for _, size := range []tea.WindowSizeMsg{
		{Width: 40, Height: 10}, // below the safeWidth/safeHeight floors
		{Width: 60, Height: 20},
		{Width: 80, Height: 24},
		{Width: 104, Height: 25}, // first wide layout, still a compact banner
		{Width: 100, Height: 30},
		{Width: 112, Height: 34},
		{Width: 160, Height: 50},
		{Width: 200, Height: 60},
	} {
		for _, step := range []step{stepRelease, stepFamilies} {
			m := newModel(releases, "~/.local/share/fonts/NerdFonts", true, IconUnicode)
			if step == stepFamilies {
				next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m = requireModel(t, next)
			}
			next, _ := m.Update(size)
			m = requireModel(t, next)

			// safeHeight, not size.Height: terminals shorter than the floor
			// cannot fit the frame at all and are deliberately laid out as if
			// they were that tall.
			if got, want := lipgloss.Height(m.View()), m.safeHeight(); got > want {
				t.Errorf("step %v at %dx%d: view height = %d, want <= %d",
					step, size.Width, size.Height, got, want)
			}
		}
	}
}

func newFamilyStepModel(t *testing.T) model {
	t.Helper()
	m := newModel([]nerdfonts.Release{
		{Name: "v3.4.0", TagName: "v3.4.0", Families: []string{"Hack", "JetBrainsMono"}},
	}, "/tmp/fonts", true, IconAuto)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // advance release -> families
	return requireModel(t, next)
}

func TestUpdateFamilyKeyEnterRequiresSelection(t *testing.T) {
	m := newFamilyStepModel(t)

	// Enter with nothing selected is a no-op: must not advance to stepDone.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = requireModel(t, next)
	if m.step != stepFamilies {
		t.Fatalf("step = %v, want stepFamilies (enter with 0 selected is a no-op)", m.step)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = requireModel(t, next)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = requireModel(t, next)
	if m.step != stepDone {
		t.Fatalf("step = %v, want stepDone after selecting then enter", m.step)
	}
	if cmd == nil {
		t.Fatal("expected a quit command after confirming selection")
	}
}

func TestUpdateFamilyKeyTogglesOff(t *testing.T) {
	m := newFamilyStepModel(t)
	for range 2 { // toggle the same family on, then off
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
		m = requireModel(t, next)
	}
	if m.selectedCount() != 0 {
		t.Fatalf("selectedCount() = %d, want 0 after toggling off", m.selectedCount())
	}
}

func TestUpdateFamilyKeyBackNavigation(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{name: "esc", msg: tea.KeyMsg{Type: tea.KeyEsc}},
		{name: "b", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newFamilyStepModel(t)
			next, _ := m.Update(tt.msg)
			m = requireModel(t, next)
			if m.step != stepRelease {
				t.Fatalf("step = %v, want stepRelease after %s", m.step, tt.name)
			}
		})
	}
}

func TestUpdateQuitKeysCancel(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{name: "q", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}},
		{name: "ctrl-c", key: tea.KeyMsg{Type: tea.KeyCtrlC}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel([]nerdfonts.Release{{TagName: "v3.4.0", Families: []string{"Hack"}}}, "/tmp", true, IconAuto)
			next, _ := m.Update(tt.key)
			if got := requireModel(t, next); !got.cancelled {
				t.Fatalf("cancelled = false, want true")
			}
		})
	}
}

func TestRunRejectsEmptyReleases(t *testing.T) {
	if _, err := Run(t.Context(), nil, Options{}); err == nil {
		t.Fatal("Run() error = nil, want error for empty releases")
	}
}

func TestModelResultSortsSelectedFamilies(t *testing.T) {
	m := newModel([]nerdfonts.Release{
		{TagName: "v3.4.0", Families: []string{"Hack", "JetBrainsMono", "FiraCode"}},
	}, "/tmp/fonts", true, IconAuto)
	m.selectedRelease = m.releases[0]
	m.selectedFamilies = map[string]bool{"JetBrainsMono": true, "FiraCode": true, "Hack": false}

	result, err := m.result()
	if err != nil {
		t.Fatalf("result() error = %v", err)
	}
	if result.Cancelled {
		t.Fatal("result Cancelled = true, want a config")
	}
	got := result.Config
	if want := []string{"FiraCode", "JetBrainsMono"}; len(got.Families) != 2 || got.Families[0] != want[0] || got.Families[1] != want[1] {
		t.Fatalf("Families = %#v, want %#v (sorted, only selected)", got.Families, want)
	}
	if got.Release != "v3.4.0" || got.Destination != "/tmp/fonts" || !got.RefreshFontCache {
		t.Fatalf("Config = %#v, want release/destination/refresh carried through", got)
	}
}

func TestModelResultCancelsWhenNoneSelected(t *testing.T) {
	m := newModel([]nerdfonts.Release{{TagName: "v3.4.0", Families: []string{"Hack"}}}, "/tmp", true, IconAuto)
	m.selectedRelease = m.releases[0]

	result, err := m.result()
	if err != nil {
		t.Fatalf("result() error = %v", err)
	}
	if !result.Cancelled {
		t.Fatal("result Cancelled = false, want true when nothing is selected")
	}
}

func requireModel(t *testing.T, got tea.Model) model {
	t.Helper()

	m, ok := got.(model)
	if !ok {
		t.Fatalf("model = %T, want tui.model", got)
	}
	return m
}

func requireItem(t *testing.T, got list.Item) item {
	t.Helper()

	i, ok := got.(item)
	if !ok {
		t.Fatalf("item = %T, want tui.item", got)
	}
	return i
}
