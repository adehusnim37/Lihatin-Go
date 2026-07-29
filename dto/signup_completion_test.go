package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestSignupCompletionStatusRequestTokenFormat(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	validate.SetTagName("binding")
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{name: "valid", token: "0123456789abcdef0123456789abcdef0123456789abcdef"},
		{name: "missing", token: "", wantErr: true},
		{name: "too short", token: "0123456789abcdef", wantErr: true},
		{name: "non hexadecimal", token: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validate.Struct(SignupCompletionStatusRequest{SignupToken: tt.token})
			if (err != nil) != tt.wantErr {
				t.Fatalf("validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
