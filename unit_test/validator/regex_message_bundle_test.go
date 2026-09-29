// Package validator_test guards the one link between this project's validator
// rules and the client-visible message they produce.
//
// A rule is registered with a *bundle key*, not message text:
//
//	v.AddRegex(RuleScore, scoreRange, "SCORE_RANGE_REGEX")
//
// and nexcommon's formator looks that key up in the common.constanta bundle at
// response time. Nothing checks that the key exists anywhere, and nexcommon
// ships no i18n files of its own - so a rule whose key is missing from this
// project's bundle emits the raw key to the caller: "Telepon harus mengikuti
// format : PHONE_NUMBER_REGEX". That is what a live run showed before these
// keys were added, on both `phone` and `score`.
//
// It reads the validator as source rather than importing it, for the reason
// unit_test/router/controller_test.go documents: importing the project's own
// packages drags in config's init(), which calls log.Fatal without a
// POSTGRESQL_* environment and kills the test binary before it prints anything.
package validator_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// ruleNameArgs finds the third argument of every AddRegex call - the one the
// validator uses as the message's bundle key. The other two Add* methods take
// no bundle key: AddEnum's second argument is an enum name and AddDateFormat's
// is a layout, and neither is looked up as a label.
var ruleNameArgs = regexp.MustCompile(`AddRegex\([^\n]*?"([A-Z_]+)"\)`)

func TestEveryRegexRuleHasALabelInEveryBundle(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "validator", "tag_validator.go"))
	if err != nil {
		t.Fatalf("reading validator/tag_validator.go: %v", err)
	}

	matches := ruleNameArgs.FindAllStringSubmatch(string(src), -1)
	if len(matches) == 0 {
		t.Fatal("found no AddRegex calls in validator/tag_validator.go - " +
			"the pattern no longer matches how rules are registered, so this guard " +
			"is checking nothing")
	}

	for _, bundle := range []string{"id-ID", "en-US"} {
		t.Run(bundle, func(t *testing.T) {
			path := filepath.Join("..", "..", "i18n", "common", "constanta", bundle+".json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", path, err)
			}

			labels := map[string]string{}
			if err := json.Unmarshal(raw, &labels); err != nil {
				t.Fatalf("parsing %s: %v", path, err)
			}

			for _, match := range matches {
				key := match[1]
				if labels[key] == "" {
					t.Errorf("regex rule %q has no label in the %s bundle, so a "+
						"rejection using it sends the raw key to the caller", key, bundle)
				}
			}
		})
	}
}
