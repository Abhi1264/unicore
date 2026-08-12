package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestJSONErrorStopsHandlerChain(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if errors.Is(err, ErrResponseWritten) {
				return nil
			}
			return err
		},
	})

	app.Post("/pay", func(c *fiber.Ctx) error {
		key, err := requireIdempotencyKey(c)
		if err != nil {
			return err
		}
		return JSON(c, fiber.StatusOK, fiber.Map{"key": key, "leaked": true})
	})

	req := httptest.NewRequest(fiber.MethodPost, "/pay", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("body not json: %s", body)
	}
	if parsed["code"] != "IDEMPOTENCY_KEY_REQUIRED" {
		t.Fatalf("code = %v, body = %s", parsed["code"], body)
	}
	if _, ok := parsed["leaked"]; ok {
		t.Fatal("handler continued after JSONError")
	}
}

func TestParseBodyRejectsInvalidJSON(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if errors.Is(err, ErrResponseWritten) {
				return nil
			}
			return err
		},
	})

	app.Post("/x", func(c *fiber.Ctx) error {
		var body struct {
			ID uuid.UUID `json:"id"`
		}
		if err := parseBody(c, &body); err != nil {
			return err
		}
		return JSON(c, fiber.StatusOK, fiber.Map{"ok": true})
	})

	req := httptest.NewRequest(fiber.MethodPost, "/x", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
