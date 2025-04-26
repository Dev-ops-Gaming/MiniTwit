package db_test

import (
	"errors"
	"testing"
	"time"

	"minitwit/db"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// setupMockDB creates a mock database connection
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	dialector := mysql.New(mysql.Config{
		Conn:                      mockDB,
		SkipInitializeWithVersion: true,
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

// Test GormGetUserId function
func TestGormGetUserId(t *testing.T) {
	gormDB, mock := setupMockDB(t)

	// Test case 1: User exists
	username := "testuser"
	mock.ExpectQuery("SELECT").
		WithArgs(username, 1).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).
			AddRow(123))

	userID, err := db.GormGetUserId(gormDB, username)
	assert.NoError(t, err)
	assert.Equal(t, 123, userID)

	// Test case 2: User does not exist
	username = "nonexistentuser"
	mock.ExpectQuery("SELECT").
		WithArgs(username, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	userID, err = db.GormGetUserId(gormDB, username)
	assert.Error(t, err)
	assert.Equal(t, 0, userID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test IsUserFollowing function
func TestIsUserFollowing(t *testing.T) {
	gormDB, mock := setupMockDB(t)

	// Test case 1: User is following
	followerID := 123
	followedID := 456
	mock.ExpectQuery("SELECT count").
		WithArgs(followerID, followedID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	isFollowing, err := db.IsUserFollowing(gormDB, followerID, followedID)
	assert.NoError(t, err)
	assert.True(t, isFollowing)

	// Test case 2: User is not following
	mock.ExpectQuery("SELECT count").
		WithArgs(followerID, 789).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	isFollowing, err = db.IsUserFollowing(gormDB, followerID, 789)
	assert.NoError(t, err)
	assert.False(t, isFollowing)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test QueryPublicTimeline function
func TestQueryPublicTimeline(t *testing.T) {
	gormDB, mock := setupMockDB(t)

	currentTime := time.Now().Unix()

	rows := sqlmock.NewRows([]string{"message_id", "author_id", "username", "email", "text", "pub_date"}).
		AddRow(1, 123, "user1", "user1@example.com", "Test message 1", currentTime).
		AddRow(2, 456, "user2", "user2@example.com", "Test message 2", currentTime)

	mock.ExpectQuery("SELECT").
		WithArgs(30).
		WillReturnRows(rows)

	messages, err := db.QueryPublicTimeline(gormDB)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)

	if len(messages) > 0 {
		assert.Equal(t, 1, messages[0].Message_id)
		assert.Equal(t, uint(123), messages[0].Author_id)
		assert.Equal(t, "Test message 1", messages[0].Text)
		assert.Equal(t, "user1", messages[0].Author)
	}

	mock.ExpectQuery("SELECT").
		WithArgs(30).
		WillReturnError(errors.New("database error"))

	messages, err = db.QueryPublicTimeline(gormDB)
	assert.Error(t, err)
	assert.Nil(t, messages)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test QueryTimeline function
func TestQueryTimeline(t *testing.T) {
	gormDB, mock := setupMockDB(t)

	userID := 123
	currentTime := time.Now().Unix()

	mock.ExpectQuery("SELECT").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"whom_id"}).
			AddRow(456))

	rows := sqlmock.NewRows([]string{"message_id", "author_id", "username", "email", "text", "pub_date"}).
		AddRow(1, userID, "testuser", "test@example.com", "Own message", currentTime).
		AddRow(2, 456, "followed", "followed@example.com", "Followed user message", currentTime)

	mock.ExpectQuery("SELECT").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 30).
		WillReturnRows(rows)

	messages, err := db.QueryTimeline(gormDB, userID)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)

	if len(messages) > 0 {
		assert.Equal(t, 1, messages[0].Message_id)
		assert.Equal(t, uint(userID), messages[0].Author_id)
		assert.Equal(t, "Own message", messages[0].Text)
		assert.Equal(t, "testuser", messages[0].Author)
	}

	mock.ExpectQuery("SELECT").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"whom_id"}))

	mock.ExpectQuery("SELECT").
		WithArgs(userID, 30).
		WillReturnError(errors.New("database error"))

	messages, err = db.QueryTimeline(gormDB, userID)
	assert.Error(t, err)
	assert.Nil(t, messages)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test QueryUserTimeline function
func TestQueryUserTimeline(t *testing.T) {
	gormDB, mock := setupMockDB(t)

	username := "profileuser"
	currentTime := time.Now().Unix()

	rows := sqlmock.NewRows([]string{"message_id", "author_id", "username", "email", "text", "pub_date"}).
		AddRow(1, 456, username, "profile@example.com", "User message 1", currentTime).
		AddRow(2, 456, username, "profile@example.com", "User message 2", currentTime)

	mock.ExpectQuery("SELECT").
		WithArgs(username, 30).
		WillReturnRows(rows)

	messages, err := db.QueryUserTimeline(gormDB, username)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)

	if len(messages) > 0 {
		assert.Equal(t, 1, messages[0].Message_id)
		assert.Equal(t, uint(456), messages[0].Author_id)
		assert.Equal(t, "User message 1", messages[0].Text)
		assert.Equal(t, username, messages[0].Author)
	}

	mock.ExpectQuery("SELECT").
		WithArgs(username, 30).
		WillReturnError(errors.New("database error"))

	messages, err = db.QueryUserTimeline(gormDB, username)
	assert.Error(t, err)
	assert.Nil(t, messages)

	assert.NoError(t, mock.ExpectationsWereMet())
}
