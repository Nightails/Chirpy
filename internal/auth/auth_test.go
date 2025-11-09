package auth

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "mySecurePassword123",
			wantErr:  false,
		},
		{
			name:     "short password",
			password: "abc",
			wantErr:  false,
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 1000),
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false,
		},
		{
			name:     "password with special characters",
			password: "p@ssw0rd!#$%^&*()",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}
				if hash == tt.password {
					t.Error("HashPassword() returned plaintext password")
				}
				// Argon2 hashes should start with $argon2
				if !strings.HasPrefix(hash, "$argon2") {
					t.Errorf("HashPassword() returned invalid hash format: %s", hash)
				}
			}
		})
	}
}

func TestHashPassword_UniqueHashes(t *testing.T) {
	password := "samePassword"
	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Each hash should be unique due to salt
	if hash1 == hash2 {
		t.Error("HashPassword() generated identical hashes for same password")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	validPassword := "correctPassword123"
	validHash, err := HashPassword(validPassword)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "correct password",
			password: validPassword,
			hash:     validHash,
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "wrongPassword",
			hash:     validHash,
			want:     false,
		},
		{
			name:     "empty password with valid hash",
			password: "",
			hash:     validHash,
			want:     false,
		},
		{
			name:     "valid password with empty hash",
			password: validPassword,
			hash:     "",
			want:     false,
		},
		{
			name:     "valid password with invalid hash format",
			password: validPassword,
			hash:     "not-a-valid-hash",
			want:     false,
		},
		{
			name:     "case sensitive check",
			password: "PASSWORD123",
			hash:     validHash,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPasswordHash(tt.password, tt.hash)
			if got != tt.want {
				t.Errorf("CheckPasswordHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPasswordHash_WithMultiplePasswords(t *testing.T) {
	passwords := []string{"password1", "password2", "password3"}
	hashes := make([]string, len(passwords))

	// Create hashes for each password
	for i, pwd := range passwords {
		hash, err := HashPassword(pwd)
		if err != nil {
			t.Fatalf("Failed to setup test: %v", err)
		}
		hashes[i] = hash
	}

	// Each password should only match its own hash
	for i, pwd := range passwords {
		for j, hash := range hashes {
			match := CheckPasswordHash(pwd, hash)
			if i == j && !match {
				t.Errorf("Password %d should match hash %d", i, j)
			}
			if i != j && match {
				t.Errorf("Password %d should not match hash %d", i, j)
			}
		}
	}
}

func TestMakeJWT(t *testing.T) {
	testUserID := uuid.New()
	testSecret := "test-secret-key"

	tests := []struct {
		name      string
		userID    uuid.UUID
		secret    string
		expiresIn time.Duration
		wantErr   bool
	}{
		{
			name:      "valid token with 1 hour expiration",
			userID:    testUserID,
			secret:    testSecret,
			expiresIn: time.Hour,
			wantErr:   false,
		},
		{
			name:      "valid token with 24 hour expiration",
			userID:    testUserID,
			secret:    testSecret,
			expiresIn: 24 * time.Hour,
			wantErr:   false,
		},
		{
			name:      "valid token with 1 minute expiration",
			userID:    testUserID,
			secret:    testSecret,
			expiresIn: time.Minute,
			wantErr:   false,
		},
		{
			name:      "valid token with different user ID",
			userID:    uuid.New(),
			secret:    testSecret,
			expiresIn: time.Hour,
			wantErr:   false,
		},
		{
			name:      "valid token with empty secret",
			userID:    testUserID,
			secret:    "",
			expiresIn: time.Hour,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := MakeJWT(tt.userID, tt.secret, tt.expiresIn)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if token == "" {
					t.Error("MakeJWT() returned empty token")
				}
				// JWT should have three parts separated by dots
				parts := strings.Split(token, ".")
				if len(parts) != 3 {
					t.Errorf("MakeJWT() returned invalid JWT format, expected 3 parts, got %d", len(parts))
				}
			}
		})
	}
}

func TestMakeJWT_UniqueTokens(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	expiresIn := time.Hour

	// Create two tokens at slightly different times
	token1, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	time.Sleep(1 * time.Second) // Need to sleep at least 1 second since JWT timestamps are in seconds

	token2, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	// Tokens should be different due to different IssuedAt times
	if token1 == token2 {
		t.Error("MakeJWT() generated identical tokens when created at different times")
	}
}

func TestMakeJWT_DeterministicForSameTime(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	expiresIn := time.Hour

	// Create two tokens immediately one after another
	token1, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	token2, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	// Tokens might be identical if created within the same second
	// This is expected behavior for JWT since timestamps are second-precision
	if token1 != token2 {
		t.Log("Tokens are different (created in different seconds)")
	} else {
		t.Log("Tokens are identical (created in same second) - this is expected JWT behavior")
	}
}

func TestValidateJWT(t *testing.T) {
	testUserID := uuid.New()
	testSecret := "test-secret-key"
	differentSecret := "different-secret"

	// Create a valid token
	validToken, err := MakeJWT(testUserID, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	// Create an expired token
	expiredToken, err := MakeJWT(testUserID, testSecret, -time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	tests := []struct {
		name        string
		tokenString string
		secret      string
		wantUserID  uuid.UUID
		wantErr     bool
	}{
		{
			name:        "valid token",
			tokenString: validToken,
			secret:      testSecret,
			wantUserID:  testUserID,
			wantErr:     false,
		},
		{
			name:        "expired token",
			tokenString: expiredToken,
			secret:      testSecret,
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "wrong secret",
			tokenString: validToken,
			secret:      differentSecret,
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "empty token",
			tokenString: "",
			secret:      testSecret,
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "malformed token",
			tokenString: "not.a.valid.jwt",
			secret:      testSecret,
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "invalid token format",
			tokenString: "invalid-token",
			secret:      testSecret,
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "empty secret",
			tokenString: validToken,
			secret:      "",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUserID, err := ValidateJWT(tt.tokenString, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUserID != tt.wantUserID {
				t.Errorf("ValidateJWT() gotUserID = %v, want %v", gotUserID, tt.wantUserID)
			}
		})
	}
}

func TestMakeJWT_ValidateJWT_Integration(t *testing.T) {
	tests := []struct {
		name      string
		userID    uuid.UUID
		secret    string
		expiresIn time.Duration
	}{
		{
			name:      "standard integration test",
			userID:    uuid.New(),
			secret:    "integration-secret",
			expiresIn: time.Hour,
		},
		{
			name:      "different user",
			userID:    uuid.New(),
			secret:    "another-secret",
			expiresIn: 30 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a token
			token, err := MakeJWT(tt.userID, tt.secret, tt.expiresIn)
			if err != nil {
				t.Fatalf("MakeJWT() error = %v", err)
			}

			// Validate the token
			userID, err := ValidateJWT(token, tt.secret)
			if err != nil {
				t.Fatalf("ValidateJWT() error = %v", err)
			}

			// Check if the user ID matches
			if userID != tt.userID {
				t.Errorf("ValidateJWT() returned userID = %v, want %v", userID, tt.userID)
			}
		})
	}
}

func TestValidateJWT_DifferentSecrets(t *testing.T) {
	userID := uuid.New()
	secret1 := "secret-one"
	secret2 := "secret-two"

	// Create token with secret1
	token, err := MakeJWT(userID, secret1, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	// Validate with secret1 should succeed
	gotUserID, err := ValidateJWT(token, secret1)
	if err != nil {
		t.Errorf("ValidateJWT() with correct secret failed: %v", err)
	}
	if gotUserID != userID {
		t.Errorf("ValidateJWT() returned userID = %v, want %v", gotUserID, userID)
	}

	// Validate with secret2 should fail
	_, err = ValidateJWT(token, secret2)
	if err == nil {
		t.Error("ValidateJWT() with wrong secret should have failed")
	}
}

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		headers   http.Header
		wantToken string
		wantErr   bool
	}{
		{
			name: "valid bearer token",
			headers: http.Header{
				"Authorization": []string{"Bearer valid-token-123"},
			},
			wantToken: "valid-token-123",
			wantErr:   false,
		},
		{
			name: "valid bearer token with JWT format",
			headers: http.Header{
				"Authorization": []string{"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"},
			},
			wantToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			wantErr:   false,
		},
		{
			name:      "missing authorization header",
			headers:   http.Header{},
			wantToken: "",
			wantErr:   true,
		},
		{
			name: "empty authorization header value",
			headers: http.Header{
				"Authorization": []string{""},
			},
			wantToken: "",
			wantErr:   true,
		},
		{
			name: "authorization header without Bearer prefix",
			headers: http.Header{
				"Authorization": []string{"token-without-bearer"},
			},
			wantToken: "",
			wantErr:   true,
		},
		{
			name: "authorization header with lowercase bearer",
			headers: http.Header{
				"Authorization": []string{"bearer lowercase-token"},
			},
			wantToken: "",
			wantErr:   true,
		},
		{
			name: "authorization header with Basic auth",
			headers: http.Header{
				"Authorization": []string{"Basic dXNlcjpwYXNz"},
			},
			wantToken: "",
			wantErr:   true,
		},
		{
			name: "bearer with no token",
			headers: http.Header{
				"Authorization": []string{"Bearer "},
			},
			wantToken: "",
			wantErr:   false,
		},
		{
			name: "bearer with extra spaces",
			headers: http.Header{
				"Authorization": []string{"Bearer  token-with-spaces"},
			},
			wantToken: " token-with-spaces",
			wantErr:   false,
		},
		{
			name: "bearer token with special characters",
			headers: http.Header{
				"Authorization": []string{"Bearer token_with-special.chars123"},
			},
			wantToken: "token_with-special.chars123",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, err := GetBearerToken(tt.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBearerToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotToken != tt.wantToken {
				t.Errorf("GetBearerToken() gotToken = %v, want %v", gotToken, tt.wantToken)
			}
		})
	}
}

func TestGetAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		headers     http.Header
		expectedKey string
		expectError bool
	}{
		{
			name: "valid API key",
			headers: http.Header{
				"Authorization": []string{"ApiKey my-secret-api-key-123"},
			},
			expectedKey: "my-secret-api-key-123",
			expectError: false,
		},
		{
			name:        "missing Authorization header",
			headers:     http.Header{},
			expectedKey: "",
			expectError: true,
		},
		{
			name: "wrong prefix - Bearer instead of ApiKey",
			headers: http.Header{
				"Authorization": []string{"Bearer some-token"},
			},
			expectedKey: "",
			expectError: true,
		},
		{
			name: "ApiKey with no space",
			headers: http.Header{
				"Authorization": []string{"ApiKeyno-space"},
			},
			expectedKey: "",
			expectError: true,
		},
		{
			name: "empty ApiKey value",
			headers: http.Header{
				"Authorization": []string{"ApiKey "},
			},
			expectedKey: "",
			expectError: false,
		},
		{
			name: "case sensitive - lowercase apikey",
			headers: http.Header{
				"Authorization": []string{"apikey my-secret-key"},
			},
			expectedKey: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, err := GetAPIKey(tt.headers)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if apiKey != tt.expectedKey {
				t.Errorf("expected key %q, got %q", tt.expectedKey, apiKey)
			}
		})
	}
}
