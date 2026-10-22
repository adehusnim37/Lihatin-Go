package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func TestFormatWrappedValidationError(t *testing.T) {
	validate, err := New()
	if err != nil {
		t.Fatal(err)
	}
	input := struct {
		Status string `validate:"oneof=open closed" json:"status" label:"Status tiket"`
		Name   string `validate:"min=3" label:"Nama"`
	}{Status: "invalid", Name: "a"}
	err = validate.Struct(input)
	if err == nil {
		t.Fatal("expected validation error")
	}
	response := FormatError(fmt.Errorf("request: %w", err), &input)
	if got := response.ErrorMap["Status"]; got != "Status tiket harus salah satu dari: open, closed" {
		t.Fatalf("oneof message = %q", got)
	}
	if got := response.ErrorMap["Name"]; got != "Nama minimal 3 karakter" {
		t.Fatalf("min message = %q", got)
	}
}

func TestWrappedBindingErrors(t *testing.T) {
	input := struct {
		Count int `json:"count" label:"Jumlah"`
	}{}
	tests := []struct{ payload, field, message string }{
		{`{"count":"wrong"}`, "count", "Jumlah harus bertipe angka"},
		{`{"count":}`, "json", "Format JSON tidak valid"},
	}
	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			err := json.Unmarshal([]byte(tt.payload), &input)
			if err == nil {
				t.Fatal("expected binding error")
			}
			response := FormatError(fmt.Errorf("binding: %w", err), &input)
			if got := response.ErrorMap[tt.field]; got != tt.message {
				t.Fatalf("message = %q, want %q", got, tt.message)
			}
		})
	}
}

func TestRegisterPreservesBindingTagsAndResponseEnvelope(t *testing.T) {
	validate := validator.New()
	validate.SetTagName("binding")
	if err := Register(validate); err != nil {
		t.Fatal(err)
	}
	input := struct {
		Username string `json:"username" binding:"required,username" label:"Nama pengguna"`
		Ignored  string `validate:"required"`
	}{Username: "invalid!"}
	err := validate.Struct(input)
	if err == nil {
		t.Fatal("expected username error from binding tag")
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	SendValidationError(ctx, fmt.Errorf("binding: %w", err), &input)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
	var body struct {
		Success bool              `json:"success"`
		Data    any               `json:"data"`
		Message string            `json:"message"`
		Error   map[string]string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Success || body.Data != nil || body.Message != "Validasi gagal" || len(body.Error) != 1 || !strings.Contains(body.Error["Username"], "Nama pengguna") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestConditionalValidationMessage(t *testing.T) {
	validate, err := New()
	if err != nil {
		t.Fatal(err)
	}
	input := struct {
		Kind string
		Code string `validate:"required_if=Kind premium" label:"Kode"`
	}{Kind: "premium"}
	response := FormatError(validate.Struct(input), &input)
	if got := response.ErrorMap["Code"]; got != "Kode wajib diisi ketika Kind adalah premium" {
		t.Fatalf("conditional message = %q", got)
	}
}
