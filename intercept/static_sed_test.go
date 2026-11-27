package intercept

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"mvdan.cc/sh/v3/syntax"
)

func TestStaticSedSnapshotsMatchNativeSed(t *testing.T) {
	dir := t.TempDir()
	source := "probe2_results.txt probe2_results.txt\nbodies2/a\naXb a.b\n"
	path := filepath.Join(dir, "old.sh")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	for _, program := range []string{"s#probe2_results.txt#probe3_results.txt#; s#bodies2/#bodies3/#", "s/a.b/MATCH/", "s/a.b/MATCH/g", "s/a\\.b/LITERAL/", "s#bodies2/#$literal/#"} {
		t.Run(program, func(t *testing.T) {
			want, err := exec.Command("sed", program, path).Output()
			if err != nil {
				t.Fatal(err)
			}
			in := reviewCommandForTest(t, dir, "sed '"+program+"' old.sh > new.sh; bash new.sh")
			if in.ScriptCaptureStatus != "complete" || len(in.Scripts) != 1 || in.Scripts[0].Content != string(want) || in.Scripts[0].Source != "command" {
				t.Fatalf("snapshot differs from sed: %+v want=%q", in, want)
			}
		})
	}
	if got, _ := os.ReadFile(path); string(got) != source {
		t.Fatal("source mutated")
	}
	if _, err := os.Stat(filepath.Join(dir, "new.sh")); !os.IsNotExist(err) {
		t.Fatal("analysis wrote the generated script")
	}
}
func TestStaticSedRejectsOpaqueOrUnsafeLayouts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.sh"), []byte("echo a a\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{
		"python3 -c 'open(\"old.sh\",\"w\").write(\"changed\")'; sed 's/a/b/' old.sh > new.sh; bash new.sh",
		"rm old.sh; sed 's/a/b/' old.sh > new.sh; bash new.sh",
		"sed -i 's/a/changed/' old.sh; sed 's/a/b/' old.sh > new.sh; bash new.sh",
		"change(){ cd /unknown; }; change; sed 's/a/b/' old.sh > new.sh; bash new.sh",
		"cd(){ echo shadowed; }; cd /unknown; sed 's/a/b/' old.sh > new.sh; bash new.sh",
		"sed 's/a/echo hidden/e' old.sh > new.sh; bash new.sh",
		"sed 's/a/b/p' old.sh > new.sh; bash new.sh",
		"sed 's/a/b/w extra' old.sh > new.sh; bash new.sh",
		"sed 's/a/&/' old.sh > new.sh; bash new.sh",
		"sed -n 's/a/b/' old.sh > new.sh; bash new.sh",
		"sed 's/a/b/' old.sh >> new.sh; bash new.sh",
		"sed 's/a/b/' old.sh > old.sh; bash old.sh",
		"sed 's/a/b/' old.sh > new.sh 2>&1; bash new.sh",
		"sed 's/a/b/' missing.sh > new.sh; bash new.sh",
		"PATH=/unexpected; sed 's/a/b/' old.sh > new.sh; bash new.sh",
		"env PATH=/unexpected sed 's/a/b/' old.sh > new.sh; bash new.sh",
		"sed(){ echo hidden; }; sed 's/a/b/' old.sh > new.sh; bash new.sh",
		"alias sed='echo hidden'; sed 's/a/b/' old.sh > new.sh; bash new.sh",
		"eval 'sed(){ echo hidden; }'; sed 's/a/b/' old.sh > new.sh; bash new.sh",
	} {
		if in := reviewCommandForTest(t, dir, command); in.ScriptCaptureStatus != "incomplete" {
			t.Fatalf("opaque transformation accepted: %s %+v", command, in)
		}
	}
	link := filepath.Join(dir, "alias.sh")
	if err := os.Link(filepath.Join(dir, "old.sh"), link); err != nil {
		t.Fatal(err)
	}
	if in := reviewCommandForTest(t, dir, "sed 's/a/b/' old.sh > alias.sh; bash alias.sh"); in.ScriptCaptureStatus != "incomplete" {
		t.Fatal("hardlink alias ignored")
	}
	for _, content := range []string{"no final newline", "echo 中文\n"} {
		if err := os.WriteFile(filepath.Join(dir, "old.sh"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if in := reviewCommandForTest(t, dir, "sed 's/a/b/' old.sh > new.sh; bash new.sh"); in.ScriptCaptureStatus != "incomplete" {
			t.Fatal("ambiguous locale/newline accepted")
		}
	}
}
func TestStaticSedGeneratedInputAndBounds(t *testing.T) {
	dir := t.TempDir()
	command := "cat > old.sh <<'SH'\necho old\nSH\nsed 's/old/new/' old.sh > new.sh; bash new.sh"
	in := reviewCommandForTest(t, dir, command)
	if in.ScriptCaptureStatus != "complete" || in.Scripts[0].Content != "echo new\n" {
		tree, parseErr := syntax.NewParser().Parse(strings.NewReader(command), "review")
		if parseErr == nil && len(tree.Stmts) >= 2 {
			allowSed := staticSedCommandUnshadowed(tree)
			remaining := 8192
			prefixKnown := staticSedPrefixKnown(tree, tree.Stmts[1], &remaining)
			generated := map[string]scriptReference{}
			catCall, catOK := tree.Stmts[0].Cmd.(*syntax.CallExpr)
			var catWrites []scriptReference
			if catOK {
				catWrites = generatedScriptWrites(tree.Stmts[0], catCall.Args, dir, true, generated, false)
			}
			for _, ref := range catWrites {
				generated[ref.path] = ref
			}
			sedCall, sedOK := tree.Stmts[1].Cmd.(*syntax.CallExpr)
			var sedWrites []scriptReference
			if sedOK {
				sedWrites = generatedScriptWrites(tree.Stmts[1], sedCall.Args, dir, true, generated, allowSed && prefixKnown)
			}
			t.Logf("PATH=%q allowSed=%t prefixKnown=%t catWrites=%+v sedWrites=%+v", os.Getenv("PATH"), allowSed, prefixKnown, catWrites, sedWrites)
		} else {
			t.Logf("diagnostic parse error=%v", parseErr)
		}
		for _, name := range []string{"cat", "sed"} {
			path, err := exec.LookPath(name)
			resolved, resolveErr := filepath.EvalSymlinks(path)
			t.Logf("command=%s path=%q err=%v resolved=%q resolveErr=%v native=%t", name, path, err, resolved, resolveErr, nativeInterpreterFile(resolved))
		}
		t.Fatalf("known virtual input ignored: %+v", in)
	}
	command = "cat > old.sh <<'SH'\necho old\nSH\nsed 's/old/next/' old.sh > next.sh; sed 's/next/final/' next.sh > final.sh; bash final.sh"
	in = reviewCommandForTest(t, dir, command)
	if in.ScriptCaptureStatus != "complete" || in.Scripts[0].Content != "echo final\n" {
		t.Fatalf("known transform chain lost: %+v", in)
	}
	subs, ok := parseStaticSed("s/a/" + strings.Repeat("b", 4096) + "/g")
	if !ok {
		t.Fatal("fixture rejected")
	}
	if _, ok := applyStaticSed(strings.Repeat("a", 1024)+"\n", subs); ok {
		t.Fatal("unbounded expansion accepted")
	}
	if _, ok := parseStaticSed(strings.Repeat("s/a/b/;", 33)); ok {
		t.Fatal("unbounded substitution list accepted")
	}
}
func TestStaticSedPrefixAnalysisBound(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.sh"), []byte("echo old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	command := strings.Repeat("echo harmless; ", 2000) + "sed 's/old/new/' old.sh > new.sh; bash new.sh"
	if in := reviewCommandForTest(t, dir, command); in.ScriptCaptureStatus != "incomplete" {
		t.Fatal("unbounded prefix analysis accepted")
	}
}
