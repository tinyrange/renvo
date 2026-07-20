package forms

import (
	"testing"

	"renvo.dev/std/graphics"
)

type themedControlExpectation struct {
	name       string
	control    *Control
	background graphics.Color
	foreground graphics.Color
}

func TestEveryBuiltInControlAppliesCompleteTheme(t *testing.T) {
	theme := DarkTheme()
	transparent := graphics.Color{}
	controls := []themedControlExpectation{
		{"Control", NewControl(), theme.Surface, theme.Text},
		{"Label", &NewLabel().Control, transparent, theme.Text},
		{"Button", &NewButton().Control, theme.Accent, theme.AccentText},
		{"TextBox", &NewTextBox().Control, theme.Field, theme.Text},
		{"TextArea", &NewTextArea().Control, theme.Field, theme.Text},
		{"CheckBox", &NewCheckBox().Control, transparent, theme.Text},
		{"RadioButton", &NewRadioButton().Control, transparent, theme.Text},
		{"PictureBox", &NewPictureBox().Control, theme.SurfaceRaised, theme.Text},
		{"Panel", &NewPanel().Control, theme.SurfaceRaised, theme.Text},
		{"ComboBox", &NewComboBox().Control, theme.Field, theme.Text},
		{"ListBox", &NewListBox().Control, theme.Field, theme.Text},
		{"ListView", &NewListView().Control, theme.Field, theme.Text},
		{"TreeView", &NewTreeView().Control, theme.Field, theme.Text},
		{"TabControl", &NewTabControl().Control, theme.Surface, theme.Text},
		{"ProgressBar", &NewProgressBar().Control, theme.SurfaceRaised, theme.Text},
		{"NumericUpDown", &NewNumericUpDown().Control, theme.Field, theme.Text},
		{"Slider", &NewSlider().Control, transparent, theme.Text},
		{"GroupBox", &NewGroupBox().Control, transparent, theme.Text},
		{"SplitContainer", &NewSplitContainer().Control, theme.Surface, theme.Text},
		{"ToolBar", &NewToolBar().Control, theme.SurfaceRaised, theme.Text},
		{"StatusBar", &NewStatusBar().Control, theme.SurfaceRaised, theme.Text},
		{"MenuBar", &NewMenuBar().Control, theme.SurfaceRaised, theme.Text},
	}

	var form Form
	form.Initialize(640, 480)
	for i := 0; i < len(controls); i++ {
		form.Add(controls[i].control)
	}
	form.ApplyTheme(theme)

	if form.Background() != theme.Window || form.Theme() != theme {
		t.Fatalf("form theme = %#v / %#v", form.Background(), form.Theme())
	}
	for i := 0; i < len(controls); i++ {
		item := controls[i]
		if item.control.applyTheme == nil && item.name != "Control" {
			t.Errorf("%s has no specialized theme handler", item.name)
		}
		if item.control.Background() != item.background || item.control.Foreground() != item.foreground || item.control.Theme() != theme {
			t.Errorf("%s theme = background %#v, foreground %#v, theme %#v", item.name, item.control.Background(), item.control.Foreground(), item.control.Theme())
		}
	}
}

func TestThemeAppliesToLateAndApplicationDefinedControls(t *testing.T) {
	var form Form
	form.Initialize(320, 200)
	theme := DarkTheme()
	theme.Accent = graphics.RGBA(200, 80, 140, 255)
	theme.Hover = graphics.RGBA(70, 40, 62, 255)
	form.ApplyTheme(theme)

	lateButton := NewButton()
	form.Add(&lateButton.Control)
	if lateButton.Background() != theme.Accent || lateButton.Theme() != theme {
		t.Fatalf("late button theme = %#v / %#v", lateButton.Background(), lateButton.Theme())
	}

	custom := NewControl()
	calls := 0
	custom.SetThemeHandler(func(active Theme) {
		calls++
		custom.SetBackground(active.Hover)
		custom.SetForeground(active.Accent)
	})
	form.Add(custom)
	if calls != 2 || custom.Background() != theme.Hover || custom.Foreground() != theme.Accent || custom.Theme() != theme {
		t.Fatalf("custom control theme = calls %d, background %#v, foreground %#v, theme %#v", calls, custom.Background(), custom.Foreground(), custom.Theme())
	}

	light := LightTheme()
	form.ApplyTheme(light)
	if calls != 3 || custom.Background() != light.Hover || lateButton.Background() != light.Accent {
		t.Fatalf("runtime theme switch = calls %d, custom %#v, button %#v", calls, custom.Background(), lateButton.Background())
	}
}

func TestStandaloneControlThemeDrivesThemeAwarePainting(t *testing.T) {
	combo := NewComboBox()
	theme := DarkTheme()
	theme.Border = graphics.RGBA(220, 40, 90, 255)
	combo.ApplyTheme(theme)
	if combo.Theme() != theme || combo.Background() != theme.Field {
		t.Fatalf("standalone combo theme = %#v / %#v", combo.Theme(), combo.Background())
	}
}
