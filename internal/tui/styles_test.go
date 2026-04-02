package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleStyles_NonNil(t *testing.T) {
	// Verify all role-based styles render without panic
	assert.NotEmpty(t, userStyle.Render("test"))
	assert.NotEmpty(t, assistantStyle.Render("test"))
	assert.NotEmpty(t, toolStyle.Render("test"))
	assert.NotEmpty(t, toolResultStyle.Render("test"))
	assert.NotEmpty(t, verboseSepStyle.Render("test"))
}
