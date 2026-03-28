package openclaw

import (
	"fmt"
	"time"

	"github.com/c3po-protocol1/holocron/internal/collector"
	"github.com/google/uuid"
)

const sourceName = "openclaw"

// Differ compares consecutive snapshots and produces MonitorEvents for changes.
type Differ struct {
	previous        map[string]OCSession // keyed by session Key
	prevStatus      map[string]collector.SessionStatus
	idleThresholdMs int64
}

// NewDiffer creates a Differ with the given idle threshold in milliseconds.
func NewDiffer(idleThresholdMs int64) *Differ {
	return &Differ{
		previous:        make(map[string]OCSession),
		prevStatus:      make(map[string]collector.SessionStatus),
		idleThresholdMs: idleThresholdMs,
	}
}

// Diff compares current sessions against the previous snapshot and returns events.
func (d *Differ) Diff(current []OCSession) []collector.MonitorEvent {
	var events []collector.MonitorEvent

	currentMap := make(map[string]OCSession, len(current))
	for _, s := range current {
		currentMap[s.Key] = s
	}

	// Detect disappeared sessions
	for key, prev := range d.previous {
		if _, exists := currentMap[key]; !exists {
			events = append(events, mapToMonitorEvent(prev.AgentID, prev, collector.EventSessionEnd, collector.StatusDone))
			delete(d.prevStatus, key)
		}
	}

	// Detect new and changed sessions
	for key, curr := range currentMap {
		prev, existed := d.previous[key]
		if !existed {
			// New session
			status := inferStatus(curr, d.idleThresholdMs)
			events = append(events, mapToMonitorEvent(curr.AgentID, curr, collector.EventSessionStart, status))
			d.prevStatus[key] = status
			continue
		}

		// Check for abort
		if curr.AbortedLastRun && !prev.AbortedLastRun {
			events = append(events, mapToMonitorEvent(curr.AgentID, curr, collector.EventError, collector.StatusError))
			d.prevStatus[key] = collector.StatusError
			continue
		}

		// Check for updatedAt change → activity
		if curr.UpdatedAt != prev.UpdatedAt {
			status := inferStatus(curr, d.idleThresholdMs)
			events = append(events, mapToMonitorEvent(curr.AgentID, curr, collector.EventStatusChange, status))
			d.prevStatus[key] = status
			continue
		}

		// Check for idle transition (same updatedAt, age grew past threshold)
		newStatus := inferStatus(curr, d.idleThresholdMs)
		if oldStatus, ok := d.prevStatus[key]; ok && oldStatus != newStatus {
			events = append(events, mapToMonitorEvent(curr.AgentID, curr, collector.EventStatusChange, newStatus))
			d.prevStatus[key] = newStatus
		}
	}

	// Store current as previous
	d.previous = currentMap

	return events
}

func inferStatus(s OCSession, idleThresholdMs int64) collector.SessionStatus {
	if s.AbortedLastRun {
		return collector.StatusError
	}
	if s.Age < idleThresholdMs {
		return collector.StatusThinking
	}
	return collector.StatusIdle
}

func mapToMonitorEvent(agent string, sess OCSession, eventType collector.EventType, status collector.SessionStatus) collector.MonitorEvent {
	keyInfo := ParseSessionKey(sess.Key)

	ev := collector.MonitorEvent{
		ID:        uuid.New().String(),
		Source:    sourceName,
		SessionID: sess.SessionID,
		Timestamp: time.Now().UnixMilli(),
		Event:     eventType,
		Status:    status,
		Detail: &collector.EventDetail{
			Message: fmt.Sprintf("%s [%s]", agent, sess.Key),
			TokenUsage: &collector.TokenUsage{
				Input:     sess.InputTokens,
				Output:    sess.OutputTokens,
				CacheRead: sess.CacheRead,
			},
		},
		Labels: keyInfo.ToLabels(),
	}

	// Add model to labels if present
	if sess.Model != "" {
		ev.Labels["model"] = sess.Model
	}

	// Add token budget info to labels if available
	if sess.PercentUsed != nil {
		ev.Labels["percent_used"] = fmt.Sprintf("%d", *sess.PercentUsed)
	}
	if sess.TotalTokens != nil {
		ev.Labels["total_tokens"] = fmt.Sprintf("%d", *sess.TotalTokens)
	}

	return ev
}
