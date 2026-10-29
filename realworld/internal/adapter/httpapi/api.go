// Package httpapi exposes the document use cases through Fiber v3.
package httpapi

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"pdf-playground/realworld/internal/document"
)

// New wires the local showcase routes. Reviewer identity is configured server-side.
func New(s *document.Service, actor string) *fiber.App {
	app := fiber.New(fiber.Config{AppName: "PDFSign showcase", BodyLimit: 11 << 20, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, Immutable: true, ErrorHandler: handleError})
	app.Use(requestid.New(), recover.New())
	app.Get("/healthz", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	api := app.Group("/api/v1")
	api.Post("/documents/generate", func(c fiber.Ctx) error {
		var input document.Contract
		if err := c.Bind().JSON(&input); err != nil {
			return fiber.NewError(400, "invalid JSON body")
		}
		d, err := s.Generate(c.Context(), input, actor)
		if err != nil {
			return err
		}
		return accepted(c, d)
	})
	api.Post("/documents/upload", func(c fiber.Ctx) error {
		header, err := c.FormFile("file")
		if err != nil {
			return fiber.NewError(400, "multipart file is required")
		}
		if header.Size > 10<<20 {
			return fiber.NewError(413, "PDF exceeds 10 MiB")
		}
		f, err := header.Open()
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, (10<<20)+1))
		if err != nil {
			return err
		}
		d, err := s.Upload(c.Context(), c.FormValue("title"), header.Filename, data, actor)
		if err != nil {
			return err
		}
		c.Set("Location", "/api/v1/documents/"+d.ID)
		return c.Status(201).JSON(d)
	})
	api.Get("/documents", func(c fiber.Ctx) error {
		limit, offset := 50, 0
		for key, target := range map[string]*int{"limit": &limit, "offset": &offset} {
			if value := c.Query(key); value != "" {
				n, err := strconv.Atoi(value)
				if err != nil || n < 0 {
					return fiber.NewError(400, "invalid pagination")
				}
				*target = n
			}
		}
		if limit < 1 || limit > 100 {
			return fiber.NewError(400, "limit must be 1..100")
		}
		docs, err := s.List(c.Context())
		if err != nil {
			return err
		}
		total := len(docs)
		if offset > total {
			offset = total
		}
		end := offset + limit
		if end > total {
			end = total
		}
		return c.JSON(fiber.Map{"items": docs[offset:end], "total": total, "offset": offset, "limit": limit})
	})
	api.Get("/documents/:id", func(c fiber.Ctx) error {
		d, err := s.View(c.Context(), c.Params("id"))
		if err != nil {
			return err
		}
		return c.JSON(d)
	})
	api.Post("/documents/:id/submit", func(c fiber.Ctx) error {
		d, err := s.Submit(c.Context(), c.Params("id"), actor)
		if err != nil {
			return err
		}
		return c.JSON(d)
	})
	api.Post("/documents/:id/approve", func(c fiber.Ctx) error {
		reason, err := bindReason(c)
		if err != nil {
			return err
		}
		d, err := s.Approve(c.Context(), c.Params("id"), reason, actor)
		if err != nil {
			return err
		}
		return accepted(c, d)
	})
	api.Post("/documents/:id/reject", func(c fiber.Ctx) error {
		reason, err := bindReason(c)
		if err != nil {
			return err
		}
		d, err := s.Reject(c.Context(), c.Params("id"), reason, actor)
		if err != nil {
			return err
		}
		return c.JSON(d)
	})
	api.Post("/documents/:id/retry", func(c fiber.Ctx) error {
		d, err := s.Retry(c.Context(), c.Params("id"), actor)
		if err != nil {
			return err
		}
		return accepted(c, d)
	})
	api.Get("/documents/:id/download", func(c fiber.Ctx) error {
		d, err := s.View(c.Context(), c.Params("id"))
		if err != nil {
			return err
		}
		version := c.Query("version", "signed")
		var data []byte
		switch version {
		case "original":
			data = d.OriginalPDF
		case "signed":
			data = d.SignedPDF
		default:
			return fiber.NewError(400, "version must be original or signed")
		}
		if len(data) == 0 {
			return fiber.NewError(409, "requested PDF version is not available")
		}
		c.Set("Content-Type", "application/pdf")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.pdf"`, d.ID, version))
		c.Set("Cache-Control", "no-store")
		return c.Send(data)
	})
	api.Get("/documents/:id/verify", func(c fiber.Ctx) error {
		report, err := s.Verify(c.Context(), c.Params("id"))
		if err != nil {
			return err
		}
		return c.JSON(report)
	})
	return app
}

func bindReason(c fiber.Ctx) (string, error) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind().JSON(&input); err != nil {
		return "", fiber.NewError(400, "invalid JSON body")
	}
	return strings.TrimSpace(input.Reason), nil
}
func accepted(c fiber.Ctx, d document.Document) error {
	c.Set("Location", "/api/v1/documents/"+d.ID)
	c.Set("Retry-After", "1")
	return c.Status(202).JSON(d)
}
func handleError(c fiber.Ctx, err error) error {
	code, message := 500, "internal server error"
	var fe *fiber.Error
	switch {
	case errors.Is(err, document.ErrInvalid):
		code, message = 400, err.Error()
	case errors.Is(err, document.ErrNotFound):
		code, message = 404, document.ErrNotFound.Error()
	case errors.Is(err, document.ErrConflict):
		code, message = 409, document.ErrConflict.Error()
	case errors.As(err, &fe):
		code, message = fe.Code, fe.Message
	default:
		slog.Error("request failed", "request_id", c.GetRespHeader("X-Request-ID"), "error", err)
	}
	return c.Status(code).JSON(fiber.Map{"error": message, "request_id": c.GetRespHeader("X-Request-ID")})
}
