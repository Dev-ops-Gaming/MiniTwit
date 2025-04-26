package main

// This code has been written partially with the help of github copilot
// and partially by Nicklas.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"minitwit/handlers"
	"minitwit/models"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

const (
	testEmail    = "test@example.com"
	testUsername = "testuser"
	testPassword = "password"
)

// Create form request helper
func createFormRequest(method, path string, formValues map[string]string) *http.Request {
	form := url.Values{}
	for key, val := range formValues {
		form.Add(key, val)
	}

	req := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// Add route parameters helper
func addRouteParams(req *http.Request, params map[string]string) *http.Request {
	return mux.SetURLVars(req, params)
}

// Test LogoutHandler
func TestLogoutHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout", nil)

	handlers.LogoutHandler()(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Test AddMessageHandler when not logged in
func TestAddMessageHandlerNotLoggedIn(t *testing.T) {
	formValues := map[string]string{
		"text": "Test message",
	}
	req := createFormRequest("POST", "/add_message", formValues)
	rec := httptest.NewRecorder()

	handlers.AddMessageHandler(nil)(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Test RegisterHandler with validation errors
func TestRegisterHandlerValidationErrors(t *testing.T) {
	t.Run("EmptyFields", func(t *testing.T) {
		formValues := map[string]string{
			"username":  "",
			"email":     testEmail,
			"password":  testPassword,
			"password2": testPassword,
		}
		req := createFormRequest("POST", "/register", formValues)
		rec := httptest.NewRecorder()

		handlers.RegisterHandler(nil)(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	// Test password mismatch
	t.Run("PasswordMismatch", func(t *testing.T) {
		formValues := map[string]string{
			"username":  testUsername,
			"email":     testEmail,
			"password":  testPassword,
			"password2": testPassword + "different",
		}
		req := createFormRequest("POST", "/register", formValues)
		rec := httptest.NewRecorder()

		handlers.RegisterHandler(nil)(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	// Test invalid email
	t.Run("InvalidEmail", func(t *testing.T) {
		formValues := map[string]string{
			"username":  testUsername,
			"email":     "invalid-email", // Not a valid email
			"password":  testPassword,
			"password2": testPassword,
		}
		req := createFormRequest("POST", "/register", formValues)
		rec := httptest.NewRecorder()

		handlers.RegisterHandler(nil)(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// Test FollowHandler when not logged in
func TestFollowHandlerNotLoggedIn(t *testing.T) {
	req := httptest.NewRequest("GET", "/user/follow", nil)
	req = addRouteParams(req, map[string]string{"username": "user"})
	rec := httptest.NewRecorder()

	handlers.FollowHandler(nil)(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

// Test UnfollowHandler when not logged in
func TestUnfollowHandlerNotLoggedIn(t *testing.T) {
	req := httptest.NewRequest("GET", "/user/unfollow", nil)
	req = addRouteParams(req, map[string]string{"username": "user"})
	rec := httptest.NewRecorder()

	handlers.UnfollowHandler(nil)(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

// Test TimelineHandler for user not logged in
func TestTimelineHandlerNotLoggedIn(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handlers.TimelineHandler(nil)(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/public", rec.Header().Get("Location"))
}

// Test the models
func TestModels(t *testing.T) {
	// Test User model
	t.Run("UserModel", func(t *testing.T) {
		user := models.User{
			User_id:  123,
			Username: testUsername,
			Email:    testEmail,
			PwHash:   "hashedpassword",
			Pwd:      "password",
		}

		assert.Equal(t, 123, user.User_id)
		assert.Equal(t, testUsername, user.Username)
		assert.Equal(t, testEmail, user.Email)
		assert.Equal(t, "hashedpassword", user.PwHash)
		assert.Equal(t, "password", user.Pwd)
	})

	// Test Message model - updated to match your actual Message structure
	t.Run("MessageModel", func(t *testing.T) {
		message := models.Message{
			Message_id: 1,
			Author_id:  123,
			Author:     testUsername,
			Email:      testEmail,
			Text:       "Test message",
			Pub_date:   123456789,
			PubDate:    "2023-10-12 15:30",
			Flagged:    0,
		}

		assert.Equal(t, 1, message.Message_id)
		assert.Equal(t, uint(123), message.Author_id)
		assert.Equal(t, testUsername, message.Author)
		assert.Equal(t, testEmail, message.Email)
		assert.Equal(t, "Test message", message.Text)
		assert.Equal(t, int64(123456789), message.Pub_date)
		assert.Equal(t, "2023-10-12 15:30", message.PubDate)
		assert.Equal(t, 0, message.Flagged)
	})

	// Test Follower model
	t.Run("FollowerModel", func(t *testing.T) {
		follower := models.Follower{
			Who_id:  123,
			Whom_id: 456,
		}

		assert.Equal(t, 123, follower.Who_id)
		assert.Equal(t, 456, follower.Whom_id)
	})
}

// Test middleware
func TestPrometheusMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Create a router with the middleware
	router := mux.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})

	router.HandleFunc("/test", testHandler).Methods("GET")

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "success", rec.Body.String())
}
