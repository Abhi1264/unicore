package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/Abhi1264/unicore/api/internal/ws"
	"github.com/gofiber/fiber/v2"
)

type AnnouncementsHandler struct {
	svc *services.AnnouncementsService
	hub *ws.Hub
}

func NewAnnouncementsHandler(svc *services.AnnouncementsService, hub *ws.Hub) *AnnouncementsHandler {
	return &AnnouncementsHandler{svc: svc, hub: hub}
}

func (h *AnnouncementsHandler) Create(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	var body struct {
		Title          string          `json:"title"`
		Body           string          `json:"body"`
		AudienceScope  string          `json:"audience_scope"`
		AudienceFilter json.RawMessage `json:"audience_filter"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if err := requireText("title", body.Title, 200); err != nil {
		return err
	}
	if err := requireText("body", body.Body, 10000); err != nil {
		return err
	}
	a, err := h.svc.Create(c.Context(), tenantID, services.CreateAnnouncementInput{
		AuthorID: claims.UserID, Title: body.Title, Body: body.Body,
		AudienceScope: body.AudienceScope, AudienceFilter: body.AudienceFilter,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	if h.hub != nil {
		payload, _ := json.Marshal(a)
		h.hub.Publish(tenantID, payload)
	}
	return JSON(c, fiber.StatusCreated, a)
}

func (h *AnnouncementsHandler) List(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	list, err := h.svc.List(c.Context(), tenantID, 50)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"announcements": list})
}

func (h *AnnouncementsHandler) Stream(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	if h.hub == nil {
		return JSONError(c, fiber.StatusServiceUnavailable, "UNAVAILABLE", "hub not configured")
	}
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	ch := h.hub.Subscribe(tenantID)
	defer h.hub.Unsubscribe(tenantID, ch)

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				_, _ = fmt.Fprintf(w, "event: announcement\ndata: %s\n\n", msg)
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				_, _ = fmt.Fprintf(w, ": ping\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
}
