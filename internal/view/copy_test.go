package view

import (
	"strings"
	"testing"

	"opensteamtool-gui-mygo/internal/app"
)

func TestBothLanguagesCarryEveryKey(t *testing.T) {
	same := map[key]bool{languageChinese: true, languageEnglish: true}
	for k, english := range copyTable {
		if strings.TrimSpace(string(k)) == "" || strings.TrimSpace(english) == "" {
			t.Errorf("key %q has a blank side in %q", k, english)
		}
		if string(k) == english && !same[k] {
			t.Errorf("key %q is not translated", k)
		}
	}
}

func TestEveryWordAndExplanationResolvesThroughTheTable(t *testing.T) {
	for w := app.MainApplied; w <= app.MainStoppingSteam; w++ {
		k := mainStateKeys[w]
		if _, ok := copyTable[k]; !ok {
			t.Errorf("word %d uses %q, which has no English", w, k)
		}
		if text(app.English, k) == string(k) || text(app.English, k) == "" {
			t.Errorf("word %d has no English text", w)
		}
	}
	for e := app.NoExplanation; e <= app.PatchNotDownloaded; e++ {
		c, ok := explanationCopies[e]
		if e == app.NoExplanation {
			if ok {
				t.Errorf("no explanation resolved to %+v", c)
			}
			continue
		}
		if !ok {
			t.Fatalf("explanation %d has no copy", e)
		}
		for _, k := range []key{c.title, c.body} {
			if _, exists := copyTable[k]; !exists {
				t.Errorf("explanation %d uses %q, which has no English", e, k)
			}
		}
	}
}

func TestTitleFollowsTheLanguage(t *testing.T) {
	if got := Title(app.Chinese); got != "OpenSteamTool 一键管理工具" {
		t.Errorf("chinese title %q", got)
	}
	if got := Title(app.English); got != "OpenSteamTool Manager" {
		t.Errorf("english title %q", got)
	}
}

func TestEveryStatusBarSlotResolvesThroughTheTable(t *testing.T) {
	for _, running := range []bool{true, false} {
		k := steamWord(running)
		if _, ok := copyTable[k]; !ok {
			t.Errorf("the Steam slot for %v uses %q, which has no English", running, k)
		}
	}
	for kind := app.BusyApplyPatch; kind <= app.BusyStopSteam; kind++ {
		state, ok := app.Stage(kind)
		if !ok {
			t.Errorf("stage %d has no state", kind)
			continue
		}
		k := mainStateKeys[state]
		if _, exists := copyTable[k]; !exists {
			t.Errorf("stage %d uses %q, which has no English", kind, k)
		}
		if word := mainStateKeys[app.Derive(app.Facts{Busy: kind}).MainState]; word != k {
			t.Errorf("the stage of %d reads %q while the main word reads %q", kind, k, word)
		}
	}
	for r := app.NoResult; r <= app.StartSteamFailed; r++ {
		k, ok := resultCopies[r]
		if r == app.NoResult {
			if ok {
				t.Errorf("no result resolved to %q", k)
			}
			continue
		}
		if !ok {
			t.Fatalf("result %d has no copy", r)
		}
		if _, exists := copyTable[k]; !exists {
			t.Errorf("result %d uses %q, which has no English", r, k)
		}
	}
}

func TestEveryTabCarriesALabel(t *testing.T) {
	tabs := app.Tabs()
	if got := tabLabels(app.Chinese); len(got) != len(tabs) {
		t.Fatalf("the strip has %d labels for %d tabs", len(got), len(tabs))
	}
	for _, tab := range tabs {
		k, ok := tabCopies[tab]
		if !ok {
			t.Errorf("tab %d has no copy", tab)
			continue
		}
		if _, ok := copyTable[k]; !ok {
			t.Errorf("tab %d uses %q, which has no English", tab, k)
		}
	}
}

func TestALanguageIsNamedInItsOwnLanguage(t *testing.T) {
	if got := text(app.English, languageChinese); got != "简体中文" {
		t.Errorf("the English interface names the Chinese preference %q", got)
	}
	if got := text(app.Chinese, languageEnglish); got != "English" {
		t.Errorf("the Chinese interface names the English preference %q", got)
	}
}
