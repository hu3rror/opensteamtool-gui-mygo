package view

import (
	"strings"
	"testing"

	"opensteamtool-gui-mygo/internal/app"
)

func TestBothLanguagesCarryEveryKey(t *testing.T) {
	for k, english := range copyTable {
		if strings.TrimSpace(string(k)) == "" || strings.TrimSpace(english) == "" {
			t.Errorf("key %q has a blank side in %q", k, english)
		}
		if string(k) == english {
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
