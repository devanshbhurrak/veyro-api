package auth

import "testing"

func TestPasswordHasherRoundTrip(t *testing.T) {
	hasher := NewPasswordHasher(PasswordParams{Memory: 8192, Iterations: 1, Parallelism: 1})
	hash, err := hasher.Hash("correct horse 42")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "correct horse 42" {
		t.Fatal("hash must not equal the plaintext")
	}
	if !hasher.Verify("correct horse 42", hash) {
		t.Fatal("expected password to verify")
	}
	if hasher.Verify("wrong password 42", hash) {
		t.Fatal("wrong password must not verify")
	}
	if hasher.Verify("correct horse 42", "not-a-hash") {
		t.Fatal("malformed hashes must not verify")
	}
}

func TestValidatePasswordPolicy(t *testing.T) {
	valid := []string{"abcdefgh12", "Password123", "a123456789"}
	for _, password := range valid {
		if err := ValidatePassword(password); err != nil {
			t.Errorf("expected %q to be valid: %v", password, err)
		}
	}
	invalid := []string{"short1", "allletters", "1234567890"}
	for _, password := range invalid {
		if err := ValidatePassword(password); err == nil {
			t.Errorf("expected %q to be rejected", password)
		}
	}
}

func TestTokenHashingAndEncryption(t *testing.T) {
	token, err := NewToken()
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	if len(token) < 32 {
		t.Fatalf("token is too short: %d", len(token))
	}
	if string(HashToken(token)) == token {
		t.Fatal("token hash must differ from the raw token")
	}

	encryptor, err := NewEncryptor("unit-test-encryption-key")
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	encrypted, err := encryptor.Encrypt(token)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if encrypted == token {
		t.Fatal("ciphertext must differ from plaintext")
	}
	decrypted, err := encryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if decrypted != token {
		t.Fatal("decrypted token does not match")
	}
	if _, err := encryptor.Decrypt("invalid"); err == nil {
		t.Fatal("tampered ciphertext must fail to decrypt")
	}
}
