package tests

import (
	"bytes"
	"context"
	"testing"

	"github.com/johndosdos/chatter/components/auth"
	"github.com/stretchr/testify/assert"
)

func TestLoginComponent(t *testing.T) {
	// Create a new buffer to write the component to
	var buf bytes.Buffer

	// Render the component
	err := auth.Login().Render(context.Background(), &buf)

	// Assert that there was no error
	assert.NoError(t, err)

	// Get the rendered HTML
	html := buf.String()

	// Assert that the HTML contains the expected elements
	assert.Contains(t, html, "Welcome to Chatter!")
	assert.Contains(t, html, "<form")
	assert.Contains(t, html, `hx-post="/account/login"`)
	assert.Contains(t, html, `name="email"`)
	assert.Contains(t, html, `name="password"`)
	assert.Contains(t, html, `type="submit"`)
	assert.Contains(t, html, "Sign In")
	assert.Contains(t, html, `hx-get="/account/signup"`)
	assert.Contains(t, html, "Sign up")
}

func TestSignupComponent(t *testing.T) {
	// Create a new buffer to write the component to
	var buf bytes.Buffer

	// Render the component
	err := auth.Signup().Render(context.Background(), &buf)

	// Assert that there was no error
	assert.NoError(t, err)

	// Get the rendered HTML
	html := buf.String()

	// Assert that the HTML contains the expected elements
	assert.Contains(t, html, "Create your account")
	assert.Contains(t, html, "<form")
	assert.Contains(t, html, `hx-post="/account/signup"`)
	assert.Contains(t, html, `name="username"`)
	assert.Contains(t, html, `name="email"`)
	assert.Contains(t, html, `name="password"`)
	assert.Contains(t, html, `name="confirm_password"`)
	assert.Contains(t, html, `type="submit"`)
	assert.Contains(t, html, "Create account")
	assert.Contains(t, html, `hx-get="/account/login"`)
	assert.Contains(t, html, "Sign in")
}
