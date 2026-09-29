package safeguards

import (
	"reflect"
	"testing"
)

func TestParseLine(t *testing.T) {
	cases := []struct {
		text string
		want []Command
	}{
		{"make check", []Command{{Line: 7, Text: "make check", Kind: KindMake, Ref: "check"}}},
		{"@$(MAKE) -j4 check", []Command{{Line: 7, Text: "$(MAKE) -j4 check", Kind: KindMake, Ref: "check"}}},
		{"make", []Command{{Line: 7, Text: "make", Kind: KindMake}}},
		{"make -f other.mk lint", []Command{{Line: 7, Text: "make -f other.mk lint", Kind: KindMake, Ref: "lint"}}},
		{"make -j 4 -I inc -o old -W new lint", []Command{{Line: 7, Text: "make -j 4 -I inc -o old -W new lint", Kind: KindMake, Ref: "lint"}}},
		{"@$(MAKE) -C sub check", []Command{{Line: 7, Text: "$(MAKE) -C sub check", Kind: KindOther}}},
		{"make --directory=sub check", []Command{{Line: 7, Text: "make --directory=sub check", Kind: KindOther}}},
		{"make check || exit 1", []Command{{Line: 7, Text: "make check || exit 1", Kind: KindOther}}},
		{"make check 2>&1 | tee log", []Command{{Line: 7, Text: "make check 2>&1 | tee log", Kind: KindOther}}},
		{"make y; make z", []Command{{Line: 7, Text: "make y; make z", Kind: KindOther}}},
		{"@go vet `go list`", []Command{{Line: 7, Text: "go vet `go list`", Kind: KindOther}}},
		{"make a && make b", []Command{
			{Line: 7, Text: "make a", Kind: KindMake, Ref: "a"},
			{Line: 7, Text: "make b", Kind: KindMake, Ref: "b"},
		}},
		{"make --file x.mk --makefile=y.mk --include-dir inc --old-file o --what-if w --jobs 4 lint", []Command{{Line: 7, Text: "make --file x.mk --makefile=y.mk --include-dir inc --old-file o --what-if w --jobs 4 lint", Kind: KindMake, Ref: "lint"}}},
		{"make --file=x.mk lint", []Command{{Line: 7, Text: "make --file=x.mk lint", Kind: KindMake, Ref: "lint"}}},
		{"make -Csub check", []Command{{Line: 7, Text: "make -Csub check", Kind: KindOther}}},
		{"make --directory x check", []Command{{Line: 7, Text: "make --directory x check", Kind: KindOther}}},
		{"make check | tee", []Command{{Line: 7, Text: "make check | tee", Kind: KindOther}}},
		{"make check > log", []Command{{Line: 7, Text: "make check > log", Kind: KindOther}}},
		{"make check < in", []Command{{Line: 7, Text: "make check < in", Kind: KindOther}}},
		{"make check &", []Command{{Line: 7, Text: "make check &", Kind: KindOther}}},
		{"make -j check", []Command{{Line: 7, Text: "make -j check", Kind: KindOther}}},
		{"make -j 4 check", []Command{{Line: 7, Text: "make -j 4 check", Kind: KindMake, Ref: "check"}}},
		{"make -j4 check", []Command{{Line: 7, Text: "make -j4 check", Kind: KindMake, Ref: "check"}}},
		{"make check -j", []Command{{Line: 7, Text: "make check -j", Kind: KindMake, Ref: "check"}}},
		{"make -l 2 test", []Command{{Line: 7, Text: "make -l 2 test", Kind: KindMake, Ref: "test"}}},
		{"make -sC sub check", []Command{{Line: 7, Text: "make -sC sub check", Kind: KindOther}}},
		{"make -kj 4 x", []Command{{Line: 7, Text: "make -kj 4 x", Kind: KindOther}}},
		{"make -s -k x", []Command{{Line: 7, Text: "make -s -k x", Kind: KindMake, Ref: "x"}}},
		{"make lint test", []Command{
			{Line: 7, Text: "make lint test", Kind: KindMake, Ref: "lint"},
			{Line: 7, Text: "make lint test", Kind: KindMake, Ref: "test"},
		}},
		{"make COVERAGE_MIN=80 test", []Command{{Line: 7, Text: "make COVERAGE_MIN=80 test", Kind: KindMake, Ref: "test"}}},
		{"sh scripts/gate.sh coverage.out", []Command{{Line: 7, Text: "sh scripts/gate.sh coverage.out", Kind: KindScript, Ref: "scripts/gate.sh"}}},
		{"bash ./scripts/gate.sh", []Command{{Line: 7, Text: "bash ./scripts/gate.sh", Kind: KindScript, Ref: "scripts/gate.sh"}}},
		{"./scripts/gate.sh", []Command{{Line: 7, Text: "./scripts/gate.sh", Kind: KindScript, Ref: "scripts/gate.sh"}}},
		{"sh -c 'go vet'", []Command{{Line: 7, Text: "sh -c 'go vet'", Kind: KindOther}}},
		{"sh", []Command{{Line: 7, Text: "sh", Kind: KindOther}}},
		{"-go vet ./...", []Command{{Line: 7, Text: "go vet ./...", Kind: KindGoVet}}},
		{"go test -race ./...", []Command{{Line: 7, Text: "go test -race ./...", Kind: KindGoTest}}},
		{"go build ./...", []Command{{Line: 7, Text: "go build ./...", Kind: KindGoBuild}}},
		{"go run ./cmd/tool", []Command{{Line: 7, Text: "go run ./cmd/tool", Kind: KindGoRun}}},
		{"go mod tidy", []Command{{Line: 7, Text: "go mod tidy", Kind: KindOther}}},
		{"go", []Command{{Line: 7, Text: "go", Kind: KindOther}}},
		{"golangci-lint run ./...", []Command{{Line: 7, Text: "golangci-lint run ./...", Kind: KindGolangci}}},
		{"gofmt -l .", []Command{{Line: 7, Text: "gofmt -l .", Kind: KindGofmt}}},
		{"git config core.hooksPath .githooks", []Command{{Line: 7, Text: "git config core.hooksPath .githooks", Kind: KindHooksPath, Ref: ".githooks"}}},
		{"git config --local core.hooksPath hooks/", []Command{{Line: 7, Text: "git config --local core.hooksPath hooks/", Kind: KindHooksPath, Ref: "hooks"}}},
		{"git config core.hooksPath", []Command{{Line: 7, Text: "git config core.hooksPath", Kind: KindOther}}},
		{"git status", []Command{{Line: 7, Text: "git status", Kind: KindOther}}},
		{"lefthook install", []Command{{Line: 7, Text: "lefthook install", Kind: KindLefthookInstall}}},
		{"pre-commit install --hook-type pre-push", []Command{{Line: 7, Text: "pre-commit install --hook-type pre-push", Kind: KindPreCommitInstall}}},
		{"pre-commit run", []Command{{Line: 7, Text: "pre-commit run", Kind: KindOther}}},
		{"echo done", []Command{{Line: 7, Text: "echo done", Kind: KindOther}}},
		{"go vet ./... && make lint", []Command{
			{Line: 7, Text: "go vet ./...", Kind: KindGoVet},
			{Line: 7, Text: "make lint", Kind: KindMake, Ref: "lint"},
		}},
		{"   ", nil},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			got := parseLine(7, c.text)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("parseLine(%q) = %+v, want %+v", c.text, got, c.want)
			}
		})
	}
}

func TestJoinLines(t *testing.T) {
	src := "a \\\n  b\n\nc\\\n"
	want := []sourceLine{{Line: 1, Text: "a b"}, {Line: 3, Text: ""}, {Line: 4, Text: "c"}}
	got := joinLines(src)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("joinLines = %+v, want %+v", got, want)
	}
}

func TestParseScript(t *testing.T) {
	src := "#!/bin/sh\n# A comment.\nset -eu\n\nmake check\nif true; then :; fi\n"
	want := []Command{
		{Line: 5, Text: "make check", Kind: KindMake, Ref: "check"},
		{Line: 6, Text: "if true; then :; fi", Kind: KindOther},
	}
	got := parseScript(src, 1)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseScript = %+v, want %+v", got, want)
	}
}
