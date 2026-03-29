package labels

import (
	"regexp"
	"sort"
	"strings"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/c3po-protocol1/holocron/internal/config"
)

// GroupMode determines how sessions are grouped in the TUI.
type GroupMode string

const (
	GroupNone      GroupMode = "none"
	GroupByAgent   GroupMode = "agent"
	GroupByChannel GroupMode = "channel"
)

// SessionGroup represents a group of sessions sharing a label value.
type SessionGroup struct {
	Label    string
	Sessions []collector.SessionState
	Active   int
}

// CycleGroupMode returns the next group mode in the cycle: none → agent → channel → none.
func CycleGroupMode(current GroupMode) GroupMode {
	switch current {
	case GroupNone:
		return GroupByAgent
	case GroupByAgent:
		return GroupByChannel
	default:
		return GroupNone
	}
}

// ApplyLabels applies config label rules to a session. Rules are evaluated in order;
// later rules override earlier ones. Provider-set labels are preserved unless explicitly overwritten.
func ApplyLabels(s *collector.SessionState, rules []config.LabelRule) {
	if s.Labels == nil {
		s.Labels = make(map[string]string)
	}

	for _, rule := range rules {
		if matchesRule(s, rule.Match) {
			for k, v := range rule.Set {
				s.Labels[k] = v
			}
		}
	}
}

// matchesRule checks if a session matches all fields in a rule's match criteria.
func matchesRule(s *collector.SessionState, match map[string]string) bool {
	for field, pattern := range match {
		value := fieldValue(s, field)
		if !globMatch(pattern, value) {
			return false
		}
	}
	return true
}

// globMatch performs simple glob matching where * matches any characters (including /)
// and ? matches a single character.
func globMatch(pattern, value string) bool {
	// Convert glob pattern to regex: * → .*, ? → ., escape the rest
	var b strings.Builder
	b.WriteString("^")
	for _, ch := range pattern {
		switch ch {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return false
	}
	return re.MatchString(value)
}

// fieldValue returns the session field value for matching.
func fieldValue(s *collector.SessionState, field string) string {
	switch field {
	case "source":
		return s.Source
	case "workspace":
		return s.Workspace
	case "sessionId":
		return s.SessionID
	case "sessionKey":
		return s.Labels["sessionKey"]
	default:
		return ""
	}
}

// GroupSessions groups sessions by the given mode's label key.
// Groups with active sessions sort first. "unlabeled" group is always last.
func GroupSessions(sessions []collector.SessionState, mode GroupMode) []SessionGroup {
	if len(sessions) == 0 {
		return nil
	}

	if mode == GroupNone {
		active := countActive(sessions)
		return []SessionGroup{{
			Label:    "",
			Sessions: sessions,
			Active:   active,
		}}
	}

	labelKey := string(mode)

	// Build groups by label value
	groupMap := make(map[string][]collector.SessionState)
	var order []string
	for _, s := range sessions {
		val := ""
		if s.Labels != nil {
			val = s.Labels[labelKey]
		}
		if val == "" {
			val = "unlabeled"
		}
		if _, exists := groupMap[val]; !exists {
			order = append(order, val)
		}
		groupMap[val] = append(groupMap[val], s)
	}

	// Build SessionGroup slice
	groups := make([]SessionGroup, 0, len(order))
	for _, label := range order {
		ss := groupMap[label]
		groups = append(groups, SessionGroup{
			Label:    label,
			Sessions: ss,
			Active:   countActive(ss),
		})
	}

	// Sort: active-first, unlabeled always last
	sort.SliceStable(groups, func(i, j int) bool {
		iUnlabeled := groups[i].Label == "unlabeled"
		jUnlabeled := groups[j].Label == "unlabeled"

		if iUnlabeled != jUnlabeled {
			return !iUnlabeled // unlabeled goes last
		}
		if iUnlabeled && jUnlabeled {
			return false
		}

		iActive := groups[i].Active > 0
		jActive := groups[j].Active > 0
		if iActive != jActive {
			return iActive // active groups first
		}
		return false // preserve order otherwise
	})

	return groups
}

func countActive(sessions []collector.SessionState) int {
	n := 0
	for _, s := range sessions {
		if s.Status == collector.StatusThinking || s.Status == collector.StatusToolRunning {
			n++
		}
	}
	return n
}
