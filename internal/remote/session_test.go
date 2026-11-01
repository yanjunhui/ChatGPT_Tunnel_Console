// Created by Yanjunhui
package remote

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func waitOutput(t *testing.T, m *Manager, name, needle string) SessionOutput {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		out, err := m.Read("", name, 0)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.Output.Text, needle) {
			return out
		}
		time.Sleep(20 * time.Millisecond)
	}
	out, _ := m.Read("", name, 0)
	t.Fatalf("missing %q in %+v", needle, out)
	return out
}

func TestPersistentTerminal(t *testing.T) {
	m, _ := newTestManager(t)
	if _, err := m.Open(t.Context(), OpenInput{Name: "shell"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Write(t.Context(), "", "shell", "export MARK=preserved\ncd /\n", false); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Write(t.Context(), "", "shell", "printf 'CHECK:%s:%s:END\\n' \"$MARK\" \"$PWD\"\n", false); err != nil {
		t.Fatal(err)
	}
	out := waitOutput(t, m, "shell", "CHECK:preserved:/:END")
	next, err := m.Read("", "shell", out.Output.NextCursor)
	if err != nil || strings.Contains(next.Output.Text, "CHECK:preserved") {
		t.Fatalf("cursor replayed old data: %+v %v", next, err)
	}
	if _, err = m.CloseSession(t.Context(), "", "shell"); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Read("", "shell", 0); err == nil {
		t.Fatal("closed session still retained")
	}
}

func TestBackgroundLifecycle(t *testing.T) {
	m, _ := newTestManager(t)
	if _, err := m.Open(t.Context(), OpenInput{Name: "job", Type: "background", Command: "cat; exit 3"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Write(t.Context(), "", "job", "payload", true); err != nil {
		t.Fatal(err)
	}
	waitOutput(t, m, "job", "payload")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		list, err := m.List("")
		if err != nil {
			t.Fatal(err)
		}
		if len(list) == 1 && list[0].Status == "exited" {
			if list[0].ExitCode == nil || *list[0].ExitCode != 3 {
				t.Fatalf("bad exit: %+v", list)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("background job did not exit")
}

func TestConcurrentSessionsAndDuplicate(t *testing.T) {
	m, _ := newTestManager(t)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := m.Open(t.Context(), OpenInput{Name: fmt.Sprint(i), Type: "background", Command: "sleep 30"})
			if err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if _, err := m.Open(t.Context(), OpenInput{Name: "0", Type: "background", Command: "printf duplicate"}); err == nil {
		t.Fatal("accepted duplicate")
	}
	list, err := m.List("")
	if err != nil || len(list) != 4 {
		t.Fatalf("list: %+v %v", list, err)
	}
	m.Close()
	if _, err = m.Open(t.Context(), OpenInput{Name: "after-close"}); err == nil {
		t.Fatal("opened after shutdown")
	}
}

func TestSessionInvalidInputs(t *testing.T) {
	m, _ := newTestManager(t)
	for _, in := range []OpenInput{{}, {Name: "x", Type: "wrong"}, {Name: "x", Type: "background"}, {Name: "x", Rows: -1}, {Name: "x", Command: "ignored"}} {
		if _, err := m.Open(t.Context(), in); err == nil {
			t.Errorf("accepted %+v", in)
		}
	}
}
