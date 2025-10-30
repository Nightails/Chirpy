package auth

import (
	"strings"
	"testing"
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
