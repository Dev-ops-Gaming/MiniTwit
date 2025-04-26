package models_test

import (
	"testing"
	"time"

	"minitwit/models"
	"minitwit/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&models.User{}, &models.Message{})
	require.NoError(t, err)

	return db
}

// TestUserModel tests the User model
func TestUserModel(t *testing.T) {
	db := setupTestDB(t)

	// Test creating a user
	t.Run("CreateUser", func(t *testing.T) {
		user := models.User{
			Username: "testuser",
			Email:    "test@example.com",
			PwHash:   "hashedpassword",
		}

		result := db.Create(&user)
		assert.NoError(t, result.Error)
		assert.NotEqual(t, 0, user.User_id, "User ID should be set after creation")

		var retrievedUser models.User
		err := db.First(&retrievedUser, user.User_id).Error
		assert.NoError(t, err)
		assert.Equal(t, "testuser", retrievedUser.Username)
		assert.Equal(t, "test@example.com", retrievedUser.Email)
		assert.Equal(t, "hashedpassword", retrievedUser.PwHash)
	})

	// Test getting a user by username
	t.Run("GetUserByUsername", func(t *testing.T) {
		// Create a test user
		user := models.User{
			Username: "findme",
			Email:    "find@example.com",
			PwHash:   "password123",
		}
		db.Create(&user)

		// Test finding the user
		foundUser, err := models.GetUserByUsername(db, "findme")
		assert.NoError(t, err)
		assert.NotNil(t, foundUser)
		assert.Equal(t, "findme", foundUser.Username)
		assert.Equal(t, "find@example.com", foundUser.Email)

		_, err = models.GetUserByUsername(db, "nonexistent")
		assert.Error(t, err, "Should return error for non-existent user")
	})

	// Test relationships between users (followers)
	t.Run("UserFollowersRelationship", func(t *testing.T) {
		// Create two users
		user1 := models.User{
			Username: "user1",
			Email:    "user1@example.com",
			PwHash:   "password1",
		}
		user2 := models.User{
			Username: "user2",
			Email:    "user2@example.com",
			PwHash:   "password2",
		}
		db.Create(&user1)
		db.Create(&user2)

		// Create a follower relationship
		follower := models.Follower{
			Who_id:  user1.User_id,
			Whom_id: user2.User_id,
		}
		result := db.Create(&follower)
		assert.NoError(t, result.Error)

		// Check if the relationship was created
		var count int64
		db.Model(&models.Follower{}).Where("who_id = ? AND whom_id = ?", user1.User_id, user2.User_id).Count(&count)
		assert.Equal(t, int64(1), count, "Follower relationship should exist")
	})
}

// TestMessageModel tests the Message model
func TestMessageModel(t *testing.T) {
	db := setupTestDB(t)

	// Create a test user
	user := models.User{
		Username: "msguser",
		Email:    "msguser@example.com",
		PwHash:   "hashedpassword",
	}
	db.Create(&user)

	// Test creating a message
	t.Run("CreateMessage", func(t *testing.T) {
		currentTime := time.Now().Unix()
		message := models.Message{
			Author_id: uint(user.User_id),
			Text:      "This is a test message",
			Pub_date:  currentTime,
			Flagged:   0,
		}

		result := db.Create(&message)
		assert.NoError(t, result.Error)
		assert.NotEqual(t, 0, message.Message_id, "Message ID should be set after creation")

		var retrievedMessage models.Message
		err := db.First(&retrievedMessage, message.Message_id).Error
		assert.NoError(t, err)
		assert.Equal(t, uint(user.User_id), retrievedMessage.Author_id)
		assert.Equal(t, "This is a test message", retrievedMessage.Text)
		assert.Equal(t, currentTime, retrievedMessage.Pub_date)
	})

	// Test the relationship between User and Message
	t.Run("UserMessagesRelationship", func(t *testing.T) {
		// Clear existing messages to ensure consistent count
		db.Where("author_id = ?", user.User_id).Delete(&models.Message{})

		// Create multiple messages for a user
		currentTime := time.Now().Unix()

		for i := 0; i < 3; i++ {
			message := models.Message{
				Author_id: uint(user.User_id),
				Text:      "Message number %d",
				Pub_date:  currentTime + int64(i),
				Flagged:   0,
			}
			db.Create(&message)
		}

		var retrievedUser models.User
		err := db.Preload("Messages").First(&retrievedUser, user.User_id).Error
		assert.NoError(t, err)

		assert.Equal(t, 3, len(retrievedUser.Messages))

		for _, msg := range retrievedUser.Messages {
			assert.Equal(t, uint(user.User_id), msg.Author_id)
		}
	})

	// Test the PubDate formatting
	t.Run("MessagePubDateFormatting", func(t *testing.T) {
		message := models.Message{
			Author_id: uint(user.User_id),
			Text:      "Testing timestamp formatting",
			Pub_date:  time.Now().Unix(),
			Flagged:   0,
		}

		formattedTime := utils.FormatTime(message.Pub_date)

		message.PubDate = formattedTime

		// Verify the format matches what's expected (DD-MM-YYYY HH:MM:SS)
		// The regex was made with AI assistance
		assert.Regexp(t, `\d{2}-\d{2}-\d{4} \d{2}:\d{2}:\d{2}`, message.PubDate)
	})
}

// TestFollowerModel tests the Follower model
func TestFollowerModel(t *testing.T) {
	db := setupTestDB(t)

	// Create test users
	user1 := models.User{
		Username: "follower",
		Email:    "follower@example.com",
		PwHash:   "password1",
	}
	user2 := models.User{
		Username: "followed",
		Email:    "followed@example.com",
		PwHash:   "password2",
	}
	db.Create(&user1)
	db.Create(&user2)

	t.Run("CreateFollowerRelationship", func(t *testing.T) {
		follower := models.Follower{
			Who_id:  user1.User_id,
			Whom_id: user2.User_id,
		}

		result := db.Create(&follower)
		assert.NoError(t, result.Error)

		var count int64
		db.Model(&models.Follower{}).Where("who_id = ? AND whom_id = ?", user1.User_id, user2.User_id).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("DeleteFollowerRelationship", func(t *testing.T) {
		result := db.Where("who_id = ? AND whom_id = ?", user1.User_id, user2.User_id).Delete(&models.Follower{})
		assert.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)

		var count int64
		db.Model(&models.Follower{}).Where("who_id = ? AND whom_id = ?", user1.User_id, user2.User_id).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("PreventDuplicateFollows", func(t *testing.T) {
		// Clear any existing relationships to ensure clean test
		db.Where("who_id = ? AND whom_id = ?", user1.User_id, user2.User_id).Delete(&models.Follower{})

		follower1 := models.Follower{
			Who_id:  user1.User_id,
			Whom_id: user2.User_id,
		}
		result1 := db.Create(&follower1)
		assert.NoError(t, result1.Error)

		follower2 := models.Follower{
			Who_id:  user1.User_id,
			Whom_id: user2.User_id,
		}
		result2 := db.Create(&follower2)

		if result2.Error != nil {
			assert.Contains(t, result2.Error.Error(), "UNIQUE constraint failed")
		} else {
			var count int64
			db.Model(&models.Follower{}).Where("who_id = ? AND whom_id = ?", user1.User_id, user2.User_id).Count(&count)
			assert.Equal(t, int64(1), count, "Should only have one follow relationship")
		}
	})
}
