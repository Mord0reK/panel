package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceSecretsCipherRoundTrip(t *testing.T) {
	cipher, err := NewServiceSecretsCipher("test-jwt-secret")
	require.NoError(t, err)

	encrypted, err := cipher.EncryptString("super-secret-token")
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := cipher.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "super-secret-token", decrypted)
}

func TestServiceSecretsCipherRandomNonce(t *testing.T) {
	cipher, err := NewServiceSecretsCipher("test-jwt-secret")
	require.NoError(t, err)

	encA, err := cipher.EncryptString("same-value")
	require.NoError(t, err)

	encB, err := cipher.EncryptString("same-value")
	require.NoError(t, err)

	assert.NotEqual(t, encA, encB)
}

func TestServiceSecretsCipherInvalidPayload(t *testing.T) {
	cipher, err := NewServiceSecretsCipher("test-jwt-secret")
	require.NoError(t, err)

	_, err = cipher.DecryptString("invalid-payload")
	require.Error(t, err)
}

func TestServiceSecretsCipherEmptyJWTSecret(t *testing.T) {
	_, err := NewServiceSecretsCipher("")
	require.Error(t, err)
}
